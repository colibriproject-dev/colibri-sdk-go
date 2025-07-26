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
	exchangeType       = "topic"
	dlqExchange        = "dlq.exchange"
	dlqSuffix          = ".dlq"

	dlqErrorReasonHeader   = "x-error-reason"
	dlqOriginalQueueHeader = "x-original-queue"
	dlqFailedAtHeader      = "x-failed-at"
	dlqMessageIdHeader     = "x-message-id"

	couldNotSetupDLQ  = "could not setup DLQ for queue %s"
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

func (r rabbitMQOriginalMessage) Nack(requeue bool, err error) error {
	if err := r.m.sendToDLQ(context.Background(), r.d, r.q, err.Error()); err != nil {
		logging.Error(context.Background()).Err(err).Msgf(couldNotSendToDLQ, r.d.MessageId)
	}

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

func (m *rabbitMQMessaging) createExchange(topicName string) error {
	return m.ch.ExchangeDeclare(
		topicName,
		exchangeType,
		true,
		false,
		false,
		false,
		nil,
	)
}

func (m *rabbitMQMessaging) setupDLQ(originalQueueName string) error {
	if err := m.ch.ExchangeDeclare(
		dlqExchange,
		exchangeType,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	dlqName := originalQueueName + dlqSuffix
	if _, err := m.ch.QueueDeclare(
		dlqName,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	return m.ch.QueueBind(
		dlqName,
		originalQueueName,
		dlqExchange,
		false,
		nil,
	)
}

func (m *rabbitMQMessaging) sendToDLQ(ctx context.Context, originalMessage amqp.Delivery, queueName string, errorReason string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	headers := amqp.Table{
		dlqErrorReasonHeader:   errorReason,
		dlqOriginalQueueHeader: queueName,
		dlqFailedAtHeader:      time.Now().Format(time.RFC3339),
		dlqMessageIdHeader:     originalMessage.MessageId,
	}

	if originalMessage.Headers != nil {
		for k, v := range originalMessage.Headers {
			if _, exists := headers[k]; !exists {
				headers[k] = v
			}
		}
	}

	err := m.ch.PublishWithContext(
		ctx,
		dlqExchange,
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:     originalMessage.ContentType,
			ContentEncoding: originalMessage.ContentEncoding,
			Body:            originalMessage.Body,
			DeliveryMode:    amqp.Persistent,
			MessageId:       originalMessage.MessageId,
			Headers:         headers,
		},
	)

	return err
}

func (m *rabbitMQMessaging) producer(ctx context.Context, p *Producer, msg *ProviderMessage) error {
	if err := m.createExchange(p.topic); err != nil {
		return err
	}

	body := []byte(msg.String())
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := m.ch.PublishWithContext(
		ctx,
		p.topic,
		msg.Action,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			MessageId:    msg.ID.String(),
		},
	)

	return err
}

func (m *rabbitMQMessaging) consumer(ctx context.Context, c *consumer) (chan *ProviderMessage, error) {
	// Setup exchange and queue
	if err := m.setupConsumerInfrastructure(ctx, c); err != nil {
		return nil, err
	}

	// Declare and bind queue
	q, err := m.declareAndBindQueue(ctx, c)
	if err != nil {
		return nil, err
	}

	// Set QoS and start consuming
	msgs, err := m.startConsuming(q.Name)
	if err != nil {
		return nil, err
	}

	// Create channel for provider messages
	providerMsgs := make(chan *ProviderMessage, 1)

	// Start message processing goroutine
	go m.processMessages(ctx, c, msgs, providerMsgs)

	return providerMsgs, nil
}

// setupConsumerInfrastructure creates the exchange and sets up the DLQ
func (m *rabbitMQMessaging) setupConsumerInfrastructure(ctx context.Context, c *consumer) error {
	if err := m.createExchange(c.topicName); err != nil {
		return err
	}

	if err := m.setupDLQ(c.queue); err != nil {
		logging.Error(ctx).Err(err).Msgf(couldNotSetupDLQ, c.queue)
		// Continue despite DLQ setup error
	}

	if c.topicName == "" {
		logging.Fatal(ctx).Msgf("Topic name is not set for queue %s", c.queue)
	}

	return nil
}

// declareAndBindQueue declares the queue and binds it to the exchange
func (m *rabbitMQMessaging) declareAndBindQueue(ctx context.Context, c *consumer) (amqp.Queue, error) {
	q, err := m.ch.QueueDeclare(
		c.queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return amqp.Queue{}, err
	}

	err = m.ch.QueueBind(
		q.Name,
		"#",
		c.topicName,
		false,
		nil,
	)
	if err != nil {
		return amqp.Queue{}, err
	}

	return q, nil
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

// handleMessage processes a single message
func (m *rabbitMQMessaging) handleMessage(ctx context.Context, c *consumer, d amqp.Delivery, providerMsgs chan<- *ProviderMessage) {
	var pm ProviderMessage

	if err := json.Unmarshal(d.Body, &pm); err != nil {
		m.handleUnmarshalError(ctx, c, d, err)
		return
	}

	// Successfully unmarshalled message
	pm.addOriginBrokerNotification(rabbitMQOriginalMessage{m: m, d: d, q: c.queue})
	providerMsgs <- &pm
}

// handleUnmarshalError handles errors when unmarshalling a message
func (m *rabbitMQMessaging) handleUnmarshalError(ctx context.Context, c *consumer, d amqp.Delivery, err error) {
	logging.Error(ctx).Err(err).Msgf(couldNotReadMsgBody, d.MessageId, c.queue)

	// Try to send to DLQ
	if dlqErr := m.sendToDLQ(ctx, d, c.queue, err.Error()); dlqErr != nil {
		logging.Error(ctx).Err(dlqErr).Msgf(couldNotSendToDLQ, d.MessageId)

		// Failed to send to DLQ, reject the message
		if nackErr := d.Reject(false); nackErr != nil {
			logging.Error(ctx).Err(nackErr).Msgf("Failed to nack message %s", d.MessageId)
		}
		return
	}

	// Successfully sent to DLQ
	logging.Info(ctx).Msgf(messageSentToDLQ, d.MessageId, c.queue+dlqSuffix, err.Error())
	if ackErr := d.Ack(false); ackErr != nil {
		logging.Error(ctx).Err(ackErr).Msgf("Could not ack message %s after DLQ", d.MessageId)
	}
}
