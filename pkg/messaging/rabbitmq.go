package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
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
	conn         *amqp.Connection
	channelMutex sync.Mutex
	channels     map[string]*amqp.Channel
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

	return &rabbitMQMessaging{
		conn:     conn,
		channels: make(map[string]*amqp.Channel),
	}
}

func (m *rabbitMQMessaging) getChannel(name string) (*amqp.Channel, error) {
	m.channelMutex.Lock()
	defer m.channelMutex.Unlock()

	if ch, exists := m.channels[name]; exists {
		return ch, nil
	}

	ch, err := m.conn.Channel()
	if err != nil {
		return nil, err
	}

	m.channels[name] = ch
	return ch, nil
}

func (m *rabbitMQMessaging) producer(ctx context.Context, p *Producer, msg *ProviderMessage) error {
	ch, err := m.getChannel(fmt.Sprintf("producer-%s", p.topic))
	if err != nil {
		return err
	}

	// Declare the exchange
	err = ch.ExchangeDeclare(
		p.topic,      // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return err
	}

	// Convert message to JSON
	body := []byte(msg.String())

	// Publish the message
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(
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
	ch, err := m.getChannel(fmt.Sprintf("consumer-%s", c.queue))
	if err != nil {
		return nil, err
	}

	// Declare the queue
	q, err := ch.QueueDeclare(
		c.queue, // name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return nil, err
	}

	// Bind the queue to the exchange with a routing key
	// Note: In RabbitMQ, we need to know which exchange to bind to.
	// Since we don't have that information directly, we'll use the queue name as the exchange name
	// This assumes that producers are publishing to an exchange with the same name as the queue
	err = ch.QueueBind(
		q.Name,  // queue name
		"#",     // routing key (# is a wildcard that matches everything)
		c.queue, // exchange
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return nil, err
	}

	// Set QoS
	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return nil, err
	}

	// Consume messages
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return nil, err
	}

	// Create a channel for provider messages
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
					d.Ack(false)
					continue
				}

				pm.addOriginBrokerNotification(d)
				providerMsgs <- &pm
				d.Ack(false)
			}
		}
	}()

	return providerMsgs, nil
}
