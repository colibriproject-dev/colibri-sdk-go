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

func TestAwsReceiptMetadata(t *testing.T) {
	t.Run("Should flatten system and message attributes and derive the delivery attempt", func(t *testing.T) {
		msg := &sqs.Message{
			Attributes: map[string]*string{
				"ApproximateReceiveCount": aws.String("4"),
			},
			MessageAttributes: map[string]*sqs.MessageAttributeValue{
				"tenant": {StringValue: aws.String("acme")},
			},
		}

		attrs := awsAttributes(msg)
		assert.Equal(t, "4", attrs["ApproximateReceiveCount"])
		assert.Equal(t, "acme", attrs["tenant"])

		attempt := awsDeliveryAttempt(msg)
		if assert.NotNil(t, attempt) {
			assert.Equal(t, 4, *attempt)
		}
	})

	t.Run("Should return nil metadata when the message carries none", func(t *testing.T) {
		msg := &sqs.Message{}

		assert.Nil(t, awsAttributes(msg))
		assert.Nil(t, awsDeliveryAttempt(msg))
	})
}

func TestRabbitMQReceiptMetadata(t *testing.T) {
	t.Run("Should stringify headers and derive the delivery attempt from x-death", func(t *testing.T) {
		d := amqp.Delivery{
			Headers: amqp.Table{
				"x-death": []interface{}{
					amqp.Table{"count": int64(6), "reason": "rejected"},
				},
			},
		}

		attrs := rabbitMQAttributes(d)
		assert.Contains(t, attrs, "x-death")

		attempt := rabbitMQDeliveryAttempt(d)
		if assert.NotNil(t, attempt) {
			assert.Equal(t, 6, *attempt)
		}
	})

	t.Run("Should return nil metadata when there are no headers", func(t *testing.T) {
		d := amqp.Delivery{}

		assert.Nil(t, rabbitMQAttributes(d))
		assert.Nil(t, rabbitMQDeliveryAttempt(d))
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
