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

	// Start a goroutine to process messages
	c.Add(1)
	go func() {
		defer c.Done()

		for {
			select {
			case <-c.done:
				return
			case d, ok := <-msgs:
				if !ok {
					return
				}

				var pm ProviderMessage
				if err := json.Unmarshal(d.Body, &pm); err != nil {
					logging.Error(ctx).Err(err).Msgf(couldNotReadMsgBody, d.MessageId, c.queue)
					if err := d.Ack(false); err != nil {
						logging.Error(ctx).Err(err).Msgf("could not ack message %s", d.MessageId)
					}
					// TODO sendo do DLQ
					continue
				}

				pm.addOriginBrokerNotification(d)
				providerMsgs <- &pm
				if err := d.Ack(false); err != nil {
					logging.Error(ctx).Err(err).Msgf("could not ack message %s", d.MessageId)
				}
			}
		}
	}()

	return providerMsgs, nil
}
