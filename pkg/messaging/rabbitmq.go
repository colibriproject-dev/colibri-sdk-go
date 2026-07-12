package messaging

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	rabbitMQDefaultURL = "amqp://guest:guest@localhost:5672/"
	rabbitMQURLEnvVar  = "RABBITMQ_URL"

	messageRejectedToDLQ = "message %s from queue %s rejected without requeue due to: %s"
)

type rabbitMQMessaging struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

type rabbitMQOriginalMessage struct {
	d amqp.Delivery
}

func (r rabbitMQOriginalMessage) Ack() error {
	return r.d.Ack(false)
}

func (r rabbitMQOriginalMessage) Nack(requeue bool, _ error) error {
	return r.d.Reject(requeue)
}

func newRabbitMQMessaging() *rabbitMQMessaging {
	url := os.Getenv(rabbitMQURLEnvVar)
	if url == "" {
		url = rabbitMQDefaultURL
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		logging.Fatal(context.Background()).Err(err).Msg(connectionError)
	}

	ch, err := conn.Channel()
	if err != nil {
		logging.Fatal(context.Background()).Err(err).Msg(connectionError)
	}

	return &rabbitMQMessaging{
		conn: conn,
		ch:   ch,
	}
}

func (m *rabbitMQMessaging) producer(ctx context.Context, p *Producer, msg *ProviderMessage) error {
	body := []byte(msg.String())
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return m.ch.PublishWithContext(
		ctx,
		p.topic,
		p.topic,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			MessageId:    msg.ID.String(),
		},
	)
}

func (m *rabbitMQMessaging) consumer(ctx context.Context, c *consumer) (chan *ProviderMessage, error) {
	msgs, err := m.startConsuming(c.queue)
	if err != nil {
		return nil, err
	}

	providerMsgs := make(chan *ProviderMessage, 1)

	go m.processMessages(ctx, c, msgs, providerMsgs)

	return providerMsgs, nil
}

// startConsuming sets QoS and starts consuming from the queue
func (m *rabbitMQMessaging) startConsuming(queueName string) (<-chan amqp.Delivery, error) {
	if err := m.ch.Qos(
		1,
		0,
		false,
	); err != nil {
		return nil, err
	}

	return m.ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
}

// processMessages handles incoming messages from RabbitMQ until the delivery
// channel closes or the consumer is canceled
func (m *rabbitMQMessaging) processMessages(ctx context.Context, c *consumer, msgs <-chan amqp.Delivery, providerMsgs chan<- *ProviderMessage) {
	defer c.Done()

	for {
		select {
		case <-c.done:
			return
		case d, ok := <-msgs:
			if !ok {
				return
			}
			m.handleMessage(ctx, c, d, providerMsgs)
		}
	}
}

// handleMessage processes with a single message
func (m *rabbitMQMessaging) handleMessage(ctx context.Context, c *consumer, d amqp.Delivery, providerMsgs chan<- *ProviderMessage) {
	var pm ProviderMessage

	if err := json.Unmarshal(d.Body, &pm); err != nil {
		m.handleUnmarshalError(ctx, c, d, err)
		return
	}

	pm.addOriginBrokerNotification(rabbitMQOriginalMessage{d: d})
	providerMsgs <- &pm
}

// handleUnmarshalError rejects a malformed message without requeue so the broker
// routes it to the configured DLX instead of redelivering or dropping it silently
func (m *rabbitMQMessaging) handleUnmarshalError(ctx context.Context, c *consumer, d amqp.Delivery, err error) {
	logging.Error(ctx).Err(err).Msgf(couldNotReadMsgBody, d.MessageId, c.queue)

	logging.Debug(ctx).Msgf(messageRejectedToDLQ, d.MessageId, c.queue, err.Error())
	if rejectErr := d.Reject(false); rejectErr != nil {
		logging.Error(ctx).Err(rejectErr).Msgf("could not reject message %s", d.MessageId)
	}
}
