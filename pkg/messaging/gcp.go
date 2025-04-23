package messaging

import (
	"context"
	"encoding/json"
	"os"

	"cloud.google.com/go/pubsub"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/logging"
)

type gcpMessaging struct {
	client *pubsub.Client
}

func newGcpMessaging() *gcpMessaging {
	client, err := pubsub.NewClient(context.Background(), os.Getenv("PUBSUB_PROJECT_ID"))
	if err != nil {
		logging.Fatal(context.Background()).Err(err).Msg(connectionError)
	}

	return &gcpMessaging{client}
}

func (m *gcpMessaging) producer(ctx context.Context, p *Producer, msg *ProviderMessage) error {
	topic := m.client.Topic(p.topic)
	result := topic.Publish(ctx, &pubsub.Message{Data: []byte(msg.String())})
	_, err := result.Get(ctx)
	return err
}

func (m *gcpMessaging) consumer(ctx context.Context, c *consumer) (chan *ProviderMessage, error) {
	ch := make(chan *ProviderMessage, 1)
	sub := m.client.Subscription(c.queue)

	go func() {
		if err := sub.Receive(ctx, func(innerCtx context.Context, msg *pubsub.Message) {
			if c.isCanceled() {
				c.Done()
				return
			}

			var pm ProviderMessage
			if err := json.Unmarshal(msg.Data, &pm); err != nil {
				logging.Error(ctx).Err(err).Msgf(couldNotReadMsgBody, msg.ID, c.queue)
				return
			}

			ch <- &pm
			msg.Ack()
		}); err != nil {
			logging.Error(ctx).Err(err).Msgf(couldNotReceiveMsg, c.queue)
		}
	}()

	return ch, nil
}
