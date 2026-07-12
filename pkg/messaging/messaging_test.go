package messaging

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/cloud"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	amqp "github.com/rabbitmq/amqp091-go"

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

// messagingProviderOpts carries the provider-specific hooks and expectations
// for the shared messaging test suite.
type messagingProviderOpts struct {
	// publishRaw publishes raw bytes to the topic, bypassing the ProviderMessage envelope
	publishRaw func(t *testing.T, topic string, body []byte)
	// assertDLQ waits until the fail-queue DLQ holds at least min messages; nil skips the check
	assertDLQ func(t *testing.T, min int)
	// redelivers is true when Nack(false) causes redelivery (AWS visibility timeout, GCP retry
	// policy) and false when it routes the message straight to the DLQ (RabbitMQ DLX)
	redelivers  bool
	waitTimeout time.Duration
}

func TestMessaging(t *testing.T) {
	logging.Initialize()

	t.Run("TestMessaging_GCP", func(t *testing.T) {
		test.InitializeGcpEmulator()
		Initialize()
		executeMessagingTest(t, messagingProviderOpts{
			publishRaw:  gcpPublishRaw,
			redelivers:  true,
			waitTimeout: 15 * time.Second,
		})
		t.Cleanup(func() {
			instance = nil
			logging.Info(context.Background()).Msg("Cleaning up GCP emulator")
		})
	})

	t.Run("TestMessaging_AWS", func(t *testing.T) {
		test.InitializeTestLocalstack()
		Initialize()
		executeMessagingTest(t, messagingProviderOpts{
			publishRaw:  awsPublishRaw,
			assertDLQ:   awsAssertDLQ,
			redelivers:  true,
			waitTimeout: 15 * time.Second,
		})
		t.Cleanup(func() {
			instance = nil
			logging.Info(context.Background()).Msg("Cleaning up AWS localstack")
		})
	})

	t.Run("TestMessaging_RabbitMQ", func(t *testing.T) {
		test.InitializeRabbitmq()
		Initialize()
		executeMessagingTest(t, messagingProviderOpts{
			publishRaw:  rabbitmqPublishRaw,
			assertDLQ:   rabbitmqAssertDLQ,
			redelivers:  false,
			waitTimeout: 15 * time.Second,
		})
		t.Cleanup(func() {
			instance = nil
			_ = os.Unsetenv("RABBITMQ_URL")
			_ = os.Unsetenv("COLIBRI_MESSAGING")
			config.COLIBRI_MESSAGING = config.MESSAGING_CLOUD_DEFAULT
			logging.Info(context.Background()).Msg("Cleaning up RabbitMQ container")
		})
	})
}

func executeMessagingTest(t *testing.T, opts messagingProviderOpts) {

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

		NewConsumer(&qc)

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

	t.Run("Should survive poison message and settle failed message according to broker semantics", func(t *testing.T) {
		done := make(chan struct{})
		var invocations atomic.Int32

		qc := queueConsumerTest{
			fn: func(ctx context.Context, message *ProviderMessage) error {
				n := invocations.Add(1)

				if opts.redelivers {
					// first delivery fails; the broker must redeliver so the second succeeds
					if n == 1 {
						return fmt.Errorf("email not valid")
					}
					if n == 2 {
						close(done)
					}
					return nil
				}

				// terminal nack (RabbitMQ): a failed handler routes the message straight
				// to the DLX, so it must be invoked exactly once and never redelivered
				if n == 1 {
					close(done)
				}
				return fmt.Errorf("email not valid")
			},
			qName: testFailQueueName,
		}

		producer := NewProducer(testFailTopicName)

		NewConsumer(&qc)

		// a malformed payload must be skipped/rejected without killing the consumer loop
		opts.publishRaw(t, testFailTopicName, []byte("this is not a valid provider message"))

		model := userMessageTest{"User Name", "user@email.com"}
		if err := producer.Publish(context.Background(), "create", model); err != nil {
			t.Fatal(err)
		}

		select {
		case <-done:
		case <-time.After(opts.waitTimeout):
			t.Fatalf("Test didn't finish after %s (invocations: %d)", opts.waitTimeout, invocations.Load())
		}

		if opts.redelivers {
			assert.GreaterOrEqual(t, invocations.Load(), int32(2), "failed message should have been redelivered")
			if opts.assertDLQ != nil {
				// only the poison message dead-letters; the valid one succeeded on redelivery
				opts.assertDLQ(t, 1)
			}
		} else if opts.assertDLQ != nil {
			// poison message + failed valid message are both routed to the DLQ
			opts.assertDLQ(t, 2)

			assert.Equal(t, int32(1), invocations.Load(), "terminal nack should not redeliver")
		}
	})
}

func awsPublishRaw(t *testing.T, topic string, body []byte) {
	snsClient := sns.New(cloud.GetAwsSession())
	topicArn := fmt.Sprintf("arn:%s:sns:%s:%s:%s",
		cloud.GetAwsARN().Partition,
		*cloud.GetAwsSession().Config.Region,
		cloud.GetAwsARN().AccountID,
		topic,
	)

	if _, err := snsClient.Publish(&sns.PublishInput{
		Message:  aws.String(string(body)),
		TopicArn: aws.String(topicArn),
	}); err != nil {
		t.Fatalf("could not publish raw message to topic %s: %v", topic, err)
	}
}

func awsAssertDLQ(t *testing.T, min int) {
	sqsClient := sqs.New(cloud.GetAwsSession())
	queueUrl, err := sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(testFailQueueName + "_DLQ"),
	})
	if err != nil {
		t.Fatalf("could not get DLQ url: %v", err)
	}

	received := 0
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		out, err := sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            queueUrl.QueueUrl,
			MaxNumberOfMessages: aws.Int64(10),
			WaitTimeSeconds:     aws.Int64(1),
		})
		if err == nil {
			received += len(out.Messages)
		}
		if received >= min {
			return
		}
	}

	t.Fatalf("expected at least %d messages in DLQ, got %d", min, received)
}

func gcpPublishRaw(t *testing.T, topic string, body []byte) {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, os.Getenv("PUBSUB_PROJECT_ID"))
	if err != nil {
		t.Fatalf("could not create pubsub client: %v", err)
	}
	defer client.Close()

	if _, err := client.Publisher(topic).Publish(ctx, &pubsub.Message{Data: body}).Get(ctx); err != nil {
		t.Fatalf("could not publish raw message to topic %s: %v", topic, err)
	}
}

func rabbitmqPublishRaw(t *testing.T, topic string, body []byte) {
	conn, err := amqp.Dial(os.Getenv(rabbitMQURLEnvVar))
	if err != nil {
		t.Fatalf("could not connect to rabbitmq: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("could not open rabbitmq channel: %v", err)
	}
	defer ch.Close()

	if err := ch.PublishWithContext(context.Background(), topic, topic, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	}); err != nil {
		t.Fatalf("could not publish raw message to topic %s: %v", topic, err)
	}
}

func rabbitmqAssertDLQ(t *testing.T, min int) {
	conn, err := amqp.Dial(os.Getenv(rabbitMQURLEnvVar))
	if err != nil {
		t.Fatalf("could not connect to rabbitmq: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("could not open rabbitmq channel: %v", err)
	}
	defer ch.Close()

	var count int
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		q, err := ch.QueueDeclarePassive(testFailQueueName+".dlq", true, false, false, false, nil)
		if err != nil {
			t.Fatalf("could not inspect DLQ: %v", err)
		}

		count = q.Messages
		if count >= min {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("expected at least %d messages in DLQ, got %d", min, count)
}
