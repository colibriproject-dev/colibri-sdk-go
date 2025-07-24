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

	maxRetries = 3
)

type rabbitMQMessaging struct {
	conn *amqp.Connection
	ch   *amqp.Channel
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
		topicName,    // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
}

// setupDLQ creates the DLQ infrastructure for a given queue
func (m *rabbitMQMessaging) setupDLQ(originalQueueName string) error {
	// Create the DLQ exchange
	if err := m.ch.ExchangeDeclare(
		dlqExchange,  // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	); err != nil {
		return err
	}

	// Create the DLQ queue
	dlqName := originalQueueName + dlqSuffix
	if _, err := m.ch.QueueDeclare(
		dlqName,
		true,  // durable
		false, // auto-deleted
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return err
	}

	// Bind the DLQ queue to the DLQ exchange
	return m.ch.QueueBind(
		dlqName,           // queue name
		originalQueueName, // routing key
		dlqExchange,       // exchange
		false,             // no-wait
		nil,               // arguments
	)
}

// sendToDLQ sends a failed message to the DLQ with error metadata
func (m *rabbitMQMessaging) sendToDLQ(ctx context.Context, originalMessage amqp.Delivery, queueName string, errorReason string) error {
	// Create a timeout context for the publish operation
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Create headers with error metadata
	headers := amqp.Table{
		dlqErrorReasonHeader:   errorReason,
		dlqOriginalQueueHeader: queueName,
		dlqFailedAtHeader:      time.Now().Format(time.RFC3339),
		dlqMessageIdHeader:     originalMessage.MessageId,
	}

	// Add any existing headers from the original message
	if originalMessage.Headers != nil {
		for k, v := range originalMessage.Headers {
			if _, exists := headers[k]; !exists {
				headers[k] = v
			}
		}
	}

	// Publish the message to the DLQ exchange
	err := m.ch.PublishWithContext(
		ctx,
		dlqExchange, // exchange
		queueName,   // routing key (original queue name)
		false,       // mandatory
		false,       // immediate
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

	// Convert a message to JSON
	body := []byte(msg.String())

	// Publish the message
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := m.ch.PublishWithContext(
		ctx,
		p.topic,    // exchange
		msg.Action, // routing key
		false,      // mandatory
		false,      // immediate
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

	// Setup DLQ for this queue
	if err := m.setupDLQ(c.queue); err != nil {
		logging.Error(ctx).Err(err).Msgf(couldNotSetupDLQ, c.queue)
		// Continue even if DLQ setup fails, as the main queue functionality should still work
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

	// Bind the queue to the exchange with a routing key
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

				xDeathHeaders, ok := d.Headers["x-death"].([]interface{})
				retryCount := 0
				if ok {
					for _, xd := range xDeathHeaders {
						if deathEntry, isMap := xd.(amqp.Table); isMap {
							if reason, exists := deathEntry["reason"].(string); exists && reason == "rejected" {
								if count, cOk := deathEntry["count"].(int64); cOk {
									retryCount += int(count)
								}
							}
						}
					}
				}

				requeue := retryCount < maxRetries
				if dlqErr := m.sendToDLQ(ctx, d, c.queue, err.Error()); dlqErr != nil {
					logging.Error(ctx).Err(dlqErr).Msgf(couldNotSendToDLQ, d.MessageId)
					if nackErr := d.Reject(requeue); nackErr != nil {
						logging.Error(ctx).Err(nackErr).Msgf("Failed to nack message %s", d.MessageId)
					}
					continue
				}

				logging.Info(ctx).Msgf(messageSentToDLQ, d.MessageId, c.queue+dlqSuffix, err.Error())

				if ackErr := d.Reject(requeue); ackErr != nil {
					logging.Error(ctx).Err(ackErr).Msgf("Could not ack message %s after DLQ", d.MessageId)
				}
				continue
			}

			pm.addOriginBrokerNotification(d)
			providerMsgs <- &pm
			if err := d.Ack(false); err != nil {
				logging.Error(ctx).Err(err).Msgf("could not ack message %s", d.MessageId)
			}
		}
	}()

	return providerMsgs, nil
}
