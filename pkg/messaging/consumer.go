package messaging

import (
	"context"
	"fmt"
	"sync"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
)

type consumer struct {
	sync.WaitGroup
	queue string
	fn    func(ctx context.Context, message *ProviderMessage) error
	done  chan any
}

type consumerObserver struct {
	c *consumer
}

func (o consumerObserver) Close() {
	o.c.close()
}

func NewConsumer(qc QueueConsumer) {
	if instance == nil {
		logging.Fatal(context.Background()).Msg(messagingNotInitialized)
	}

	c := &consumer{
		WaitGroup: sync.WaitGroup{},
		queue:     qc.QueueName(),
		fn:        qc.Consume,
		done:      make(chan any),
	}

	observer.Attach(consumerObserver{c: c})
	startListener(c)
}

func startListener(c *consumer) {
	ch := createConsumer(c)

	go func() {
		for {
			msg := <-ch
			ctx := context.Background()
			msg.AuthContext.SetInContext(ctx)

			if err := c.fn(ctx, msg); err != nil {
				logging.Error(ctx).Err(err).Msgf(couldNotProcessMsg, msg.ID)
			}
		}
	}()
}

func createConsumer(c *consumer) chan *ProviderMessage {
	txn, ctx := monitoring.StartTransaction(context.Background(), fmt.Sprintf(messagingConsumerTransaction, c.queue))
	defer monitoring.EndTransaction(txn)

	ch, err := instance.consumer(ctx, c)
	if err != nil {
		logging.Error(ctx).Err(err).Msgf(createQueueError, c.queue)
		monitoring.NoticeError(txn, err)
		return nil
	}

	return ch
}

func (c *consumer) close() {
	logging.Info(context.Background()).Msgf(closingQueueConsumer, c.queue)
	close(c.done)
	c.Wait()
}

func (c *consumer) isCanceled() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}
