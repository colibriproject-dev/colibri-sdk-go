package messaging

import (
	"context"
	"fmt"
	"sync"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
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
			processMessage(c, msg)
		}
	}()
}

func processMessage(c *consumer, msg *ProviderMessage) {
	ctxRoot := context.WithValue(context.Background(), logging.CorrelationIDParam, msg.CorrelationID)

	txn, ctx := monitoring.StartTransaction(ctxRoot, fmt.Sprintf(messagingConsumerTransaction, c.queue), colibrimonitoringbase.SpanKindConsumer)
	monitoring.AddTransactionAttribute(txn, logging.CorrelationIDParam, msg.CorrelationID)
	monitoring.AddTransactionAttribute(txn, "action", msg.Action)
	monitoring.AddTransactionAttribute(txn, "messageId", msg.ID.String())
	monitoring.AddTransactionAttribute(txn, "span.kind", "CONSUMER")
	defer monitoring.EndTransactionSegment(txn)

	msg.AuthContext.SetInContext(ctx)

	if err := c.fn(ctx, msg); err != nil {
		logging.Error(ctx).Err(err).Msgf(couldNotProcessMsg, msg.ID)
		if err := msg.Nack(false, err); err != nil {
			logging.Error(ctx).Err(err).Msgf("error sending nack for message %s", msg.ID)
		}
		monitoring.NoticeError(txn, err)
		return
	}

	if err := msg.Ack(); err != nil {
		logging.Error(ctx).Err(err).Msgf("error sending ack for message %s", msg.ID)
	}

	logging.Debug(ctx).Msgf("message %s processed", msg.ID)
}

func createConsumer(c *consumer) chan *ProviderMessage {
	ctx := context.Background()
	ch, err := instance.consumer(ctx, c)
	if err != nil {
		logging.Fatal(ctx).Err(err).Msgf(createQueueError, c.queue)
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
