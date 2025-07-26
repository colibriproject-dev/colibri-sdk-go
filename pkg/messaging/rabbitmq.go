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

	// DLQ error headers
	dlqErrorReasonHeader   = "x-error-reason"
	dlqOriginalQueueHeader = "x-original-queue"
	dlqFailedAtHeader      = "x-failed-at"
	dlqMessageIdHeader     = "x-message-id"

	// DLQ error messages
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
	if err := m.createExchange(c.topicName); err != nil {
		return nil, err
	}

	if err := m.setupDLQ(c.queue); err != nil {
		logging.Error(ctx).Err(err).Msgf(couldNotSetupDLQ, c.queue)
	}

	q, err := m.ch.QueueDeclare(
		c.queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if c.topicName == "" {
		logging.Fatal(ctx).Msgf("Topic name is not set for queue %s", c.queue)
	}

	if err = m.ch.QueueBind(
		q.Name,
		"#",
		c.topicName,
		false,
		nil,
	); err != nil {
		return nil, err
	}

	if err = m.ch.Qos(
		1,
		0,
		false,
	); err != nil {
		return nil, err
	}

	msgs, err := m.ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	providerMsgs := make(chan *ProviderMessage, 1)

	go func() {
		defer c.Done()

		for d := range msgs {
			var pm ProviderMessage

			if err := json.Unmarshal(d.Body, &pm); err != nil {
				logging.Error(ctx).Err(err).Msgf(couldNotReadMsgBody, d.MessageId, c.queue)

				if dlqErr := m.sendToDLQ(ctx, d, c.queue, err.Error()); dlqErr != nil {
					logging.Error(ctx).Err(dlqErr).Msgf(couldNotSendToDLQ, d.MessageId)

					if nackErr := d.Reject(false); nackErr != nil {
						logging.Error(ctx).Err(nackErr).Msgf("Failed to nack message %s", d.MessageId)
					}
					continue
				}

				logging.Info(ctx).Msgf(messageSentToDLQ, d.MessageId, c.queue+dlqSuffix, err.Error())
				if ackErr := d.Ack(false); ackErr != nil {
					logging.Error(ctx).Err(ackErr).Msgf("Could not ack message %s after DLQ", d.MessageId)
				}
				continue
			}

			pm.addOriginBrokerNotification(rabbitMQOriginalMessage{m: m, d: d, q: c.queue})
			providerMsgs <- &pm
		}
	}()

	return providerMsgs, nil
}
