package messaging

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

var (
	// ErrMessagingNotInitialized is returned when a consumer is created before Initialize.
	ErrMessagingNotInitialized = errors.New(messagingNotInitialized)
	// ErrNilConsumerChannel is returned when the message broker reports no error but hands
	// back no channel to consume from.
	ErrNilConsumerChannel = errors.New("message broker returned a nil consumer channel")
	// ErrMessagingClosed is returned when a consumer is created after the graceful shutdown
	// has started, so it would never receive the stop signal.
	ErrMessagingClosed = errors.New("messaging module is shutting down")
)

// maxAckTimeout is the upper bound for the acknowledgement of a message.
const maxAckTimeout = 15 * time.Second

// ackTimeout bounds the acknowledgement of a message. Without it, a broker that stops
// responding during the shutdown would hold the drain open until the module wide timeout
// expires, because the WaitGroup covers the goroutine that acknowledges. It is derived from
// that same timeout so the bound stays below it whatever WAIT_GROUP_TIMEOUT_SECONDS is
// configured to, instead of only while the default 90s is in place.
func ackTimeout() time.Duration {
	half := time.Duration(config.WAIT_GROUP_TIMEOUT_SECONDS) * time.Second / 2
	if half <= 0 {
		return maxAckTimeout
	}

	return min(half, maxAckTimeout)
}

type consumer struct {
	sync.WaitGroup
	queue    string
	fn       func(ctx context.Context, message *ProviderMessage) error
	done     chan any
	stopOnce sync.Once
}

// NewConsumer creates a queue consumer, aborting the process when it cannot be created.
// Use NewConsumerWithError to handle the failure instead.
func NewConsumer(qc QueueConsumer) {
	if err := NewConsumerWithError(qc); err != nil {
		logging.Fatal(context.Background()).Err(err).Msgf(createQueueError, qc.QueueName())
	}
}

// NewConsumerWithError creates a queue consumer and returns the reason it could not be
// created, leaving nothing half registered when it fails.
func NewConsumerWithError(qc QueueConsumer) error {
	_, err := newConsumer(qc)

	return err
}

// newConsumer wires a consumer only after the broker has handed back a usable channel.
// Registering before that point would leave a consumer counted by the WaitGroup with no
// goroutine to release it, and the shutdown would then wait on it forever.
func newConsumer(qc QueueConsumer) (*consumer, error) {
	provider := moduleInstance()
	if provider == nil {
		return nil, ErrMessagingNotInitialized
	}

	c := &consumer{
		queue: qc.QueueName(),
		fn:    qc.Consume,
		done:  make(chan any),
	}

	if err := registerConsumer(c); err != nil {
		return nil, err
	}

	ch, err := provider.consumer(moduleContext(), c)
	if err != nil {
		c.abandon()
		return nil, fmt.Errorf(createQueueError+": %w", c.queue, err)
	}

	if ch == nil {
		c.abandon()
		return nil, fmt.Errorf("%w for queue %s", ErrNilConsumerChannel, c.queue)
	}

	// the shutdown may have started while the broker was creating the consumer, so the
	// registration on the running counter is only valid if the module is still open
	if err = activateConsumer(); err != nil {
		c.abandon()
		return nil, err
	}

	c.Add(1)

	// no observer is attached per consumer: the module observer signals every consumer
	// registerConsumer recorded, and the observer list has no way to detach, so one entry
	// per consumer would grow without bound in an application creating them dynamically
	go c.listen(ch)

	return c, nil
}

// listen processes the messages delivered by the broker. Both WaitGroups cover this
// goroutine, so the shutdown waits for the message being processed to finish. Messages
// already buffered but not picked up are left unacknowledged on purpose: the broker
// redelivers them, which is preferable to delaying the shutdown.
func (c *consumer) listen(ch chan *ProviderMessage) {
	// defers run last-in-first-out: releasing the module counter first means that once
	// c.Wait returns, the module drain has already been accounted for too
	defer c.Done()
	defer observer.DoneRunning()

	for {
		select {
		case <-c.done:
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			if msg == nil {
				continue
			}

			processMessage(c, msg)
		}
	}
}

func processMessage(c *consumer, msg *ProviderMessage) {
	ctxRoot := context.WithValue(context.Background(), logging.CorrelationIDParam, msg.CorrelationID)

	txn, ctx := monitoring.StartTransaction(ctxRoot, fmt.Sprintf(messagingConsumerTransaction, c.queue), colibrimonitoringbase.SpanKindConsumer)
	monitoring.AddTransactionAttribute(txn, logging.CorrelationIDParam, msg.CorrelationID)
	monitoring.AddTransactionAttribute(txn, "action", msg.Action)
	monitoring.AddTransactionAttribute(txn, "messageId", msg.ID.String())
	monitoring.AddTransactionAttribute(txn, "span.kind", "CONSUMER")
	defer monitoring.EndTransactionSegment(txn)

	// registered after the transaction defer so it runs before it: the panic must be
	// recorded while the transaction is still open. Consume is user code running on a bare
	// goroutine, so a panic here would otherwise take the whole process down.
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic processing message %s: %v", msg.ID, r)
			logging.Error(ctx).Msgf("%v\n%s", err, debug.Stack())
			monitoring.NoticeError(txn, err)
			nackMessage(ctx, msg, err)
		}
	}()

	msg.AuthContext.SetInContext(ctx)

	if err := c.fn(ctx, msg); err != nil {
		logging.Error(ctx).Err(err).Msgf(couldNotProcessMsg, msg.ID)
		nackMessage(ctx, msg, err)
		monitoring.NoticeError(txn, err)
		return
	}

	ackCtx, cancel := context.WithTimeout(ctx, ackTimeout())
	defer cancel()

	if err := msg.Ack(ackCtx); err != nil {
		logging.Error(ctx).Err(err).Msgf("error sending ack for message %s", msg.ID)
	}

	logging.Debug(ctx).Msgf("message %s processed", msg.ID)
}

func nackMessage(ctx context.Context, msg *ProviderMessage, cause error) {
	nackCtx, cancel := context.WithTimeout(ctx, ackTimeout())
	defer cancel()

	if err := msg.Nack(nackCtx, false, cause); err != nil {
		logging.Error(ctx).Err(err).Msgf("error sending nack for message %s", msg.ID)
	}
}

// signalStop tells the consumer to stop pulling messages. It is idempotent because the
// module shutdown, a failed creation and a direct close may all reach it for the same
// consumer.
func (c *consumer) signalStop() {
	c.stopOnce.Do(func() {
		logging.Info(context.Background()).Msgf(closingQueueConsumer, c.queue)
		close(c.done)
	})
}

// abandon releases a consumer that never started listening, so a provider goroutine that
// already began does not outlive the failure.
func (c *consumer) abandon() {
	c.signalStop()
	unregisterConsumer(c)
}

// close stops the consumer and waits for the message being processed to finish.
func (c *consumer) close() {
	c.signalStop()
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
