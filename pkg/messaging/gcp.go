package messaging

import (
	"context"
	"encoding/json"
	"os"

	"cloud.google.com/go/pubsub/v2"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

type gcpMessaging struct {
	client *pubsub.Client
}

type gcpOriginalMessage struct {
	msg *pubsub.Message
}

// Ack acknowledges the message after successful processing.
func (g gcpOriginalMessage) Ack(_ context.Context) error {
	g.msg.Ack()
	return nil
}

// Nack rejects the message; the subscription's retry policy redelivers it and the
// dead-letter policy (if configured) forwards it to the dead-letter topic.
func (g gcpOriginalMessage) Nack(_ context.Context, _ bool, _ error) error {
	g.msg.Nack()
	return nil
}

func newGcpMessaging() *gcpMessaging {
	client, err := pubsub.NewClient(context.Background(), os.Getenv("PUBSUB_PROJECT_ID"))
	if err != nil {
		logging.Fatal(context.Background()).Err(err).Msg(connectionError)
	}

	return &gcpMessaging{client}
}

// close releases the Pub/Sub client during the graceful shutdown.
func (m *gcpMessaging) close() error {
	return m.client.Close()
}

func (m *gcpMessaging) producer(ctx context.Context, p *Producer, msg *ProviderMessage) error {
	topic := m.client.Publisher(p.topic)
	result := topic.Publish(ctx, &pubsub.Message{Data: []byte(msg.String())})
	_, err := result.Get(ctx)
	return err
}

func (m *gcpMessaging) consumer(ctx context.Context, c *consumer) (chan *ProviderMessage, error) {
	ch := make(chan *ProviderMessage, 1)
	sub := m.client.Subscriber(c.queue)
	sub.ReceiveSettings.MaxOutstandingMessages = 1
	sub.ReceiveSettings.NumGoroutines = 1

	go func() {
		receiveCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// stop receiving when the consumer is closed; canceling an already
		// canceled context is a no-op
		go func() {
			<-c.done
			cancel()
		}()

		if err := sub.Receive(receiveCtx, func(_ context.Context, msg *pubsub.Message) {
			m.handleMessage(ctx, c, msg, ch)
		}); err != nil {
			logging.Error(ctx).Err(err).Msgf(couldNotReceiveMsg, c.queue)
		}
	}()

	return ch, nil
}

// handleMessage delivers a received message to the consumer channel with a real ack/nack
// bound to the Pub/Sub message. A message the listener can no longer take is nacked, so the
// subscription's retry policy redelivers it right away — unlike SQS, which only makes it
// visible again once the visibility timeout expires.
func (m *gcpMessaging) handleMessage(ctx context.Context, c *consumer, msg *pubsub.Message, ch chan *ProviderMessage) {
	var pm ProviderMessage
	if err := json.Unmarshal(msg.Data, &pm); err != nil {
		logging.Error(ctx).Err(err).Msgf(couldNotReadMsgBody, msg.ID, c.queue)
		msg.Nack()
		return
	}

	pm.addOriginBrokerNotification(gcpOriginalMessage{msg: msg})
	pm.setReceiptMetadata(msg.Attributes, msg.DeliveryAttempt)

	select {
	case ch <- &pm:
	case <-c.done:
		// nobody is listening anymore; let the retry policy redeliver instead of
		// blocking this callback and holding Receive open
		msg.Nack()
	}
}
