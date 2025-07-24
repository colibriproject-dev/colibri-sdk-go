package messaging

import (
	"context"
	"fmt"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"os"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/test"
	"github.com/stretchr/testify/assert"
)

type userMessageTest struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

const (
	testTopicName     = "COLIBRI_PROJECT_USER_CREATE"
	testQueueName     = "COLIBRI_PROJECT_USER_CREATE_APP_CONSUMER"
	testFailTopicName = "COLIBRI_PROJECT_FAIL_USER_CREATE"
	testFailQueueName = "COLIBRI_PROJECT_FAIL_USER_CREATE_APP_CONSUMER"
)

type queueConsumerTest struct {
	fn    func(ctx context.Context, n *ProviderMessage) error
	qName string
}

func (q *queueConsumerTest) Consume(ctx context.Context, pm *ProviderMessage) error {
	return q.fn(ctx, pm)
}

func (q *queueConsumerTest) QueueName() string {
	return q.qName
}

func TestMessaging(t *testing.T) {
	t.Run("TestMessaging_GCP", func(t *testing.T) {
		test.InitializeGcpEmulator()
		Initialize()
		executeMessagingTest(t)
		t.Cleanup(func() {
			instance = nil
			logging.Info(context.Background()).Msg("Cleaning up GCP emulator")
		})
	})

	t.Run("TestMessaging_AWS", func(t *testing.T) {
		test.InitializeTestLocalstack()
		Initialize()
		executeMessagingTest(t)
		t.Cleanup(func() {
			instance = nil
			logging.Info(context.Background()).Msg("Cleaning up AWS localstack")
		})
	})

	t.Run("TestMessaging_RabbitMQ", func(t *testing.T) {
		test.InitializeRabbitmq()
		Initialize()
		executeMessagingTest(t)
		t.Cleanup(func() {
			instance = nil
			_ = os.Unsetenv("RABBITMQ_URL")
			_ = os.Unsetenv("USE_RABBITMQ")
			logging.Info(context.Background()).Msg("Cleaning up RabbitMQ container")
		})
	})
}

func executeMessagingTest(t *testing.T) {

	t.Run("Should return nil when process message with success", func(t *testing.T) {
		chSuccess := make(chan string)

		qc := queueConsumerTest{
			fn: func(ctx context.Context, message *ProviderMessage) error {
				successfulProcessMessage := fmt.Sprintf("processing message: %v", message)
				logging.Info(ctx).Msgf("Received message: %v", message)
				chSuccess <- successfulProcessMessage
				return nil
			},
			qName: testQueueName,
		}

		producer := NewProducer(testTopicName)
		NewConsumerWithTopic(&qc, testTopicName)

		model := userMessageTest{"User Name", "user@email.com"}
		if err := producer.Publish(context.Background(), "create", model); err != nil {
			logging.Error(context.Background()).Err(err).Msg(
				fmt.Sprintf("Error publishing message to topic %s", testTopicName),
			)
			t.Fatal(err)
		}

		timeout := time.After(2 * time.Second)
		select {
		case msgProcessing := <-chSuccess:
			assert.NotEmpty(t, msgProcessing)
		case <-timeout:
			t.Fatal("Test didn't finish after 2s")
		}
	})

	t.Run("Should return error when process message with error and send message to dlq", func(t *testing.T) {
		chFail := make(chan string)
		qc := queueConsumerTest{
			fn: func(ctx context.Context, message *ProviderMessage) error {
				err := fmt.Errorf("email not valid")
				chFail <- err.Error()
				return err
			},
			qName: testFailQueueName,
		}

		producer := NewProducer(testFailTopicName)
		NewConsumerWithTopic(&qc, testFailTopicName)

		model := userMessageTest{"User Name", "user@email.com"}
		if err := producer.Publish(context.Background(), "create", model); err != nil {
			t.Fatal(err)
		}

		timeout := time.After(2 * time.Second)
		select {
		case msgDLQ := <-chFail:
			assert.Equal(t, "email not valid", msgDLQ)
		case <-timeout:
			t.Fatal("Test didn't finish after 2s")
		}
	})
}
