package messaging

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestInitializeUnsupportedCloud(t *testing.T) {
	t.Run("Should panic when cloud provider has no messaging support", func(t *testing.T) {
		logging.Initialize()

		previousInstance := instance
		previousCloud := config.CLOUD
		previousMessaging := config.COLIBRI_MESSAGING
		t.Cleanup(func() {
			instance = previousInstance
			config.CLOUD = previousCloud
			config.COLIBRI_MESSAGING = previousMessaging
		})

		instance = nil
		config.CLOUD = config.CLOUD_NONE
		config.COLIBRI_MESSAGING = config.MESSAGING_CLOUD_DEFAULT

		assert.Panics(t, func() { Initialize() })
	})
}

func TestAwsHandleMessage(t *testing.T) {
	t.Run("Should skip message when body is not a valid notification", func(t *testing.T) {
		m := &awsMessaging{}
		c := &consumer{queue: "test-queue", done: make(chan any)}
		ch := make(chan *ProviderMessage, 1)

		m.handleMessage(context.Background(), c, &sqs.GetQueueUrlOutput{}, &sqs.Message{
			MessageId: aws.String("msg-1"),
			Body:      aws.String("this is not a json body"),
		}, ch)

		assert.Empty(t, ch)
	})
}

func TestRabbitMQProcessMessages(t *testing.T) {
	m := &rabbitMQMessaging{}

	t.Run("Should stop when consumer is closed", func(t *testing.T) {
		c := &consumer{queue: "test-queue", done: make(chan any)}
		msgs := make(chan amqp.Delivery)
		out := make(chan *ProviderMessage, 1)

		c.Add(1)
		close(c.done)
		m.processMessages(context.Background(), c, msgs, out)

		assert.Empty(t, out)
	})

	t.Run("Should stop when delivery channel closes", func(t *testing.T) {
		c := &consumer{queue: "test-queue", done: make(chan any)}
		msgs := make(chan amqp.Delivery)
		out := make(chan *ProviderMessage, 1)

		c.Add(1)
		close(msgs)
		m.processMessages(context.Background(), c, msgs, out)

		assert.Empty(t, out)
	})
}

func TestRabbitMQHandleUnmarshalError(t *testing.T) {
	t.Run("Should not panic when reject fails", func(t *testing.T) {
		m := &rabbitMQMessaging{}
		c := &consumer{queue: "test-queue", done: make(chan any)}

		assert.NotPanics(t, func() {
			m.handleUnmarshalError(context.Background(), c, amqp.Delivery{MessageId: "msg-1"}, errors.New("invalid body"))
		})
	})
}
