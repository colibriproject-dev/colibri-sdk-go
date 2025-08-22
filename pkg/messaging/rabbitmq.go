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
	dlqExchangeSuffix  = "dlx"

	couldNotSendToDLQ = "could not send message %s to DLQ"
	messageSentToDLQ  = "message %s sent to DLQ %s due to: %s"
)

type rabbitMQMessaging struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

type rabbitMQOriginalMessage struct {
	m *rabbitMQMessaging
	d amqp.Delivery
	q string
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

// processMessages handles incoming messages from RabbitMQ
func (m *rabbitMQMessaging) processMessages(ctx context.Context, c *consumer, msgs <-chan amqp.Delivery, providerMsgs chan<- *ProviderMessage) {
	defer c.Done()

	for d := range msgs {
		m.handleMessage(ctx, c, d, providerMsgs)
	}
}

// handleMessage processes with a single message
func (m *rabbitMQMessaging) handleMessage(ctx context.Context, c *consumer, d amqp.Delivery, providerMsgs chan<- *ProviderMessage) {
	var pm ProviderMessage

	if err := json.Unmarshal(d.Body, &pm); err != nil {
		m.handleUnmarshalError(ctx, c, d, err)
		return
	}

	pm.addOriginBrokerNotification(rabbitMQOriginalMessage{m: m, d: d, q: c.queue})
	providerMsgs <- &pm
}

// handleUnmarshalError handles errors when unmarshalling a message
func (m *rabbitMQMessaging) handleUnmarshalError(ctx context.Context, c *consumer, d amqp.Delivery, err error) {
	logging.Error(ctx).Err(err).Msgf(couldNotReadMsgBody, d.MessageId, c.queue)

	logging.Debug(ctx).Msgf(messageSentToDLQ, d.MessageId, c.queue, err.Error())
	if ackErr := d.Ack(false); ackErr != nil {
		logging.Error(ctx).Err(ackErr).Msgf("Could not ack message %s after DLQ", d.MessageId)
	}
}
