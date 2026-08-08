package messaging

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
	"github.com/stretchr/testify/assert"
)

// initMessagingTestDeps guards the package level singletons the consumer reads from its
// listener goroutine. The observer is deliberately left out: it is reinitialized per test.
var initMessagingTestDeps sync.Once

// fakeMessaging stands in for a broker so the consumer lifecycle can be exercised without a
// live connection.
type fakeMessaging struct {
	ch          chan *ProviderMessage
	consumerErr error
	nilChannel  bool
	closed      atomic.Bool
	// seq orders the shutdown steps. Handlers and observers stamp themselves with next() so
	// the tests can assert what happened before what, without sleeping on it.
	seq      atomic.Int32
	closedAt atomic.Int32
}

// next returns the position of the next event in the shutdown sequence.
func (f *fakeMessaging) next() int32 {
	return f.seq.Add(1)
}

// errBrokerClosed stands in for what a real broker answers once its connection is gone.
var errBrokerClosed = errors.New("broker connection is closed")

func (f *fakeMessaging) producer(_ context.Context, _ *Producer, _ *ProviderMessage) error {
	if f.closed.Load() {
		return errBrokerClosed
	}

	return nil
}

func (f *fakeMessaging) consumer(_ context.Context, _ *consumer) (chan *ProviderMessage, error) {
	if f.consumerErr != nil {
		return nil, f.consumerErr
	}

	if f.nilChannel {
		return nil, nil
	}

	return f.ch, nil
}

// close makes the fake a brokerCloser, so the shutdown order can be asserted.
func (f *fakeMessaging) close() error {
	f.closed.Store(true)
	f.closedAt.Store(f.next())
	return nil
}

// dbLikeObserver stands in for the sql and cache database observers: it closes in the same
// phase they do, without waiting on its own, and records when it actually closed.
type dbLikeObserver struct {
	f        *fakeMessaging
	closedAt atomic.Int32
}

func (o *dbLikeObserver) Close() {
	o.closedAt.Store(o.f.next())
}

// recordingOriginalMessage records whether the broker was told to ack or nack.
type recordingOriginalMessage struct {
	acked  atomic.Bool
	nacked atomic.Bool
}

func (r *recordingOriginalMessage) Ack(_ context.Context) error {
	r.acked.Store(true)
	return nil
}

func (r *recordingOriginalMessage) Nack(_ context.Context, _ bool, _ error) error {
	r.nacked.Store(true)
	return nil
}

// setupMessagingTest installs a fake broker and guarantees every consumer it creates is
// stopped and drained, so no leftover goroutine keeps the module WaitGroup above zero for
// the next test.
func setupMessagingTest(t *testing.T) *fakeMessaging {
	t.Helper()

	initMessagingTestDeps.Do(func() {
		logging.Initialize()
		monitoring.Initialize()
	})

	// observers are never detached, so a shared subject would accumulate consumers of every
	// previous test and notify them all
	observer.Initialize()

	previousInstance := moduleInstance()
	f := &fakeMessaging{ch: make(chan *ProviderMessage, 1)}
	setInstance(f)
	resetModuleState()

	t.Cleanup(func() {
		for _, c := range beginShutdown() {
			c.close()
		}
		setInstance(previousInstance)
	})

	return f
}

func startFakeConsumer(t *testing.T, queue string, fn func(ctx context.Context, msg *ProviderMessage) error) *consumer {
	t.Helper()

	c, err := newConsumer(&testProducerConsumer{fn: fn, queueName: queue})
	if err != nil {
		t.Fatalf("could not start consumer: %v", err)
	}

	return c
}

// closeWithin runs close in a goroutine and reports how long it took, so a close that never
// returns fails the test instead of hanging the suite.
func closeWithin(t *testing.T, c *consumer, timeout time.Duration) time.Duration {
	t.Helper()

	start := time.Now()
	done := make(chan time.Duration, 1)
	go func() {
		c.close()
		done <- time.Since(start)
	}()

	select {
	case elapsed := <-done:
		return elapsed
	case <-time.After(timeout):
		t.Fatalf("close did not return within %s", timeout)
		return 0
	}
}

func TestConsumerCloseWaitsForInFlightMessage(t *testing.T) {
	t.Run("Should wait for the message being processed before returning", func(t *testing.T) {
		const processing = 200 * time.Millisecond

		f := setupMessagingTest(t)
		started := make(chan struct{})
		var finished atomic.Bool

		c := startFakeConsumer(t, "in-flight-queue", func(_ context.Context, _ *ProviderMessage) error {
			close(started)
			time.Sleep(processing)
			finished.Store(true)
			return nil
		})

		f.ch <- NewConsumerMessage("test", nil, nil, nil)

		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("message was never processed")
		}

		elapsed := closeWithin(t, c, 5*time.Second)

		assert.True(t, finished.Load(), "close returned while the message was still being processed")
		assert.GreaterOrEqual(t, elapsed, processing/2, "close returned too early to have waited for the in-flight message")
	})
}

func TestConsumerStopsAfterClose(t *testing.T) {
	t.Run("Should not process messages delivered after close", func(t *testing.T) {
		f := setupMessagingTest(t)
		var processed atomic.Int32

		c := startFakeConsumer(t, "stops-after-close-queue", func(_ context.Context, _ *ProviderMessage) error {
			processed.Add(1)
			return nil
		})

		closeWithin(t, c, 5*time.Second)

		f.ch <- NewConsumerMessage("test", nil, nil, nil)
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, int32(0), processed.Load(), "a message delivered after close was still processed")
	})
}

func TestConsumerHandlesClosedChannel(t *testing.T) {
	t.Run("Should stop the listener when the broker closes the channel", func(t *testing.T) {
		f := setupMessagingTest(t)
		var processed atomic.Int32

		c := startFakeConsumer(t, "closed-channel-queue", func(_ context.Context, _ *ProviderMessage) error {
			processed.Add(1)
			return nil
		})

		close(f.ch)
		time.Sleep(50 * time.Millisecond)

		// the listener already returned, so close has nothing left to wait for
		elapsed := closeWithin(t, c, time.Second)

		assert.Less(t, elapsed, 100*time.Millisecond, "listener did not stop when the channel closed")
		assert.Equal(t, int32(0), processed.Load())
	})
}

func TestConsumerIgnoresNilMessage(t *testing.T) {
	t.Run("Should skip a nil message and keep processing the next one", func(t *testing.T) {
		f := setupMessagingTest(t)
		processed := make(chan string, 2)

		c := startFakeConsumer(t, "nil-message-queue", func(_ context.Context, msg *ProviderMessage) error {
			processed <- msg.Action
			return nil
		})

		f.ch <- nil
		f.ch <- NewConsumerMessage("valid", nil, nil, nil)

		select {
		case action := <-processed:
			assert.Equal(t, "valid", action)
		case <-time.After(time.Second):
			t.Fatal("the nil message stopped the listener")
		}

		closeWithin(t, c, 5*time.Second)
		assert.Empty(t, processed, "the nil message should not have been processed")
	})
}

func TestConsumerProcessesSequentially(t *testing.T) {
	t.Run("Should never overlap the processing of two messages", func(t *testing.T) {
		f := setupMessagingTest(t)

		var inFlight atomic.Int32
		var overlapped atomic.Bool
		processed := make(chan struct{}, 2)

		c := startFakeConsumer(t, "sequential-queue", func(_ context.Context, _ *ProviderMessage) error {
			if inFlight.Add(1) > 1 {
				overlapped.Store(true)
			}
			time.Sleep(50 * time.Millisecond)
			inFlight.Add(-1)
			processed <- struct{}{}
			return nil
		})

		f.ch <- NewConsumerMessage("test", nil, nil, nil)
		f.ch <- NewConsumerMessage("test", nil, nil, nil)

		for i := 0; i < 2; i++ {
			select {
			case <-processed:
			case <-time.After(time.Second):
				t.Fatalf("only %d of 2 messages were processed", i)
			}
		}

		closeWithin(t, c, 5*time.Second)

		assert.False(t, overlapped.Load(), "two messages were processed at the same time")
	})
}

func TestProcessMessageRecoversFromPanic(t *testing.T) {
	t.Run("Should nack the message and keep the listener alive", func(t *testing.T) {
		f := setupMessagingTest(t)
		processed := make(chan string, 2)

		c := startFakeConsumer(t, "panic-queue", func(_ context.Context, msg *ProviderMessage) error {
			if msg.Action == "boom" {
				panic("consumer exploded")
			}
			processed <- msg.Action
			return nil
		})

		origin := &recordingOriginalMessage{}
		exploding := NewConsumerMessage("boom", nil, nil, nil)
		exploding.addOriginBrokerNotification(origin)

		f.ch <- exploding
		f.ch <- NewConsumerMessage("ok", nil, nil, nil)

		select {
		case action := <-processed:
			assert.Equal(t, "ok", action, "the listener died with the panic")
		case <-time.After(2 * time.Second):
			t.Fatal("the listener did not survive the panic")
		}

		closeWithin(t, c, 5*time.Second)

		assert.True(t, origin.nacked.Load(), "the message that panicked was not nacked")
		assert.False(t, origin.acked.Load(), "the message that panicked must not be acked")
	})
}

func TestNewConsumerWithError(t *testing.T) {
	t.Run("Should return an error when messaging was not initialized", func(t *testing.T) {
		setupMessagingTest(t)
		setInstance(nil)

		err := NewConsumerWithError(&testProducerConsumer{queueName: "any-queue"})

		assert.ErrorIs(t, err, ErrMessagingNotInitialized)
	})

	t.Run("Should return an error when the broker fails to create the consumer", func(t *testing.T) {
		f := setupMessagingTest(t)
		brokerErr := errors.New("queue does not exist")
		f.consumerErr = brokerErr

		err := NewConsumerWithError(&testProducerConsumer{queueName: "missing-queue"})

		assert.ErrorIs(t, err, brokerErr)
		assert.Empty(t, beginShutdown(), "a failed consumer must not stay registered")
	})

	t.Run("Should return an error when the broker hands back a nil channel", func(t *testing.T) {
		f := setupMessagingTest(t)
		f.nilChannel = true

		err := NewConsumerWithError(&testProducerConsumer{queueName: "nil-channel-queue"})

		assert.ErrorIs(t, err, ErrNilConsumerChannel)
		assert.Empty(t, beginShutdown(), "a failed consumer must not stay registered")
	})

	t.Run("Should refuse to create a consumer after the shutdown started", func(t *testing.T) {
		setupMessagingTest(t)
		beginShutdown()

		err := NewConsumerWithError(&testProducerConsumer{queueName: "late-queue"})

		assert.ErrorIs(t, err, ErrMessagingClosed)
	})
}

func TestNewConsumer(t *testing.T) {
	t.Run("Should abort the process when the broker refuses the consumer", func(t *testing.T) {
		f := setupMessagingTest(t)
		f.consumerErr = errors.New("queue does not exist")

		assert.Panics(t, func() { NewConsumer(&testProducerConsumer{queueName: "missing-queue"}) })
		assert.Empty(t, beginShutdown(), "a failed consumer must not stay registered")
	})

	t.Run("Should register the consumer when the broker accepts it", func(t *testing.T) {
		setupMessagingTest(t)

		assert.NotPanics(t, func() {
			NewConsumer(&testProducerConsumer{
				queueName: "fatal-variant-queue",
				fn:        func(_ context.Context, _ *ProviderMessage) error { return nil },
			})
		})
		assert.Len(t, beginShutdown(), 1)
	})
}

func TestConsumerSignalStopIsIdempotent(t *testing.T) {
	t.Run("Should tolerate being signaled more than once", func(t *testing.T) {
		setupMessagingTest(t)
		c := startFakeConsumer(t, "idempotent-queue", func(_ context.Context, _ *ProviderMessage) error {
			return nil
		})

		assert.NotPanics(t, func() {
			c.signalStop()
			c.signalStop()

			// the module shutdown reaches every registered consumer, and the test cleanup
			// closes them once more afterwards
			for _, registered := range beginShutdown() {
				registered.signalStop()
			}

			c.close()
		})
	})
}

func TestModuleShutdownDrainsInFlightMessage(t *testing.T) {
	t.Run("Should stop the consumers, drain the work and close the broker", func(t *testing.T) {
		const processing = 200 * time.Millisecond

		f := setupMessagingTest(t)
		started := make(chan struct{})
		var finishedAt atomic.Int32

		startFakeConsumer(t, "module-drain-queue", func(_ context.Context, _ *ProviderMessage) error {
			close(started)
			time.Sleep(processing)
			finishedAt.Store(f.next())
			return nil
		})

		f.ch <- NewConsumerMessage("test", nil, nil, nil)

		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("message was never processed")
		}

		shutdownWithin(t, 5*time.Second)

		assert.NotZero(t, finishedAt.Load(), "the shutdown did not wait for the in-flight message")
		assert.True(t, f.closed.Load(), "the broker connection was not closed")
		assert.Less(t, finishedAt.Load(), f.closedAt.Load(),
			"the broker connection was closed before the in-flight message was drained")
		assert.NotNil(t, moduleContext().Err(), "the module context was not canceled")
	})
}

func TestModuleShutdownKeepsTheBrokerOpenWhileDraining(t *testing.T) {
	t.Run("Should let a handler still draining publish before the broker is closed", func(t *testing.T) {
		f := setupMessagingTest(t)

		var c *consumer
		var publishErr error
		var publishedAt atomic.Int32
		started := make(chan struct{})
		// released after the stopping phase has returned, so the publish lands in the drain
		// phase with no room for it to have happened before the broker could be closed
		released := make(chan struct{})

		c = startFakeConsumer(t, "publish-while-draining-queue", func(ctx context.Context, _ *ProviderMessage) error {
			close(started)
			<-c.done
			<-released

			publishErr = NewProducer("drain-topic").Publish(ctx, "test", nil)
			publishedAt.Store(f.next())

			return nil
		})

		f.ch <- NewConsumerMessage("test", nil, nil, nil)

		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("message was never processed")
		}

		previousTimeout := config.WAIT_GROUP_TIMEOUT_SECONDS
		config.WAIT_GROUP_TIMEOUT_SECONDS = 1
		t.Cleanup(func() { config.WAIT_GROUP_TIMEOUT_SECONDS = previousTimeout })

		o := &messagingObserver{}
		o.Stop()
		close(released)

		// the drain phase, run the way the pipeline does. The consumer is waited on directly
		// as well, so work another test left on the process wide counter can neither hold the
		// drain nor end it before this handler is done
		runWithin(t, 5*time.Second, "drain", func() {
			c.Wait()
			observer.WaitRunningTimeout()
		})
		o.Close()

		assert.NoError(t, publishErr, "a handler draining could not publish: the broker was closed too early")
		assert.True(t, f.closed.Load(), "the broker connection was not closed")
		assert.Less(t, publishedAt.Load(), f.closedAt.Load(),
			"the broker connection was closed before the handler draining published")
	})
}

func TestModuleShutdownTimesOutOnStuckConsumer(t *testing.T) {
	t.Run("Should complete the shutdown instead of hanging on a stuck consumer", func(t *testing.T) {
		f := setupMessagingTest(t)

		release := make(chan struct{})

		previousTimeout := config.WAIT_GROUP_TIMEOUT_SECONDS
		config.WAIT_GROUP_TIMEOUT_SECONDS = 1
		t.Cleanup(func() { config.WAIT_GROUP_TIMEOUT_SECONDS = previousTimeout })

		started := make(chan struct{})
		c := startFakeConsumer(t, "stuck-queue", func(_ context.Context, _ *ProviderMessage) error {
			close(started)
			<-release
			return nil
		})

		f.ch <- NewConsumerMessage("test", nil, nil, nil)

		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("message was never processed")
		}

		elapsed := shutdownWithin(t, 10*time.Second)

		assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "the shutdown did not wait for the timeout")
		assert.Less(t, elapsed, 5*time.Second, "the shutdown waited past the configured timeout")
		assert.True(t, f.closed.Load(), "the broker connection was not closed after the timeout")

		// the abandoned handler still reads the configured timeout when it acknowledges, so
		// let it finish before the cleanup restores it
		close(release)
		c.Wait()
	})
}

func TestModuleShutdownRunsBeforeTheDatabase(t *testing.T) {
	t.Run("Should drain the in-flight message before a database observer attached earlier closes", func(t *testing.T) {
		f := setupMessagingTest(t)

		// short enough that a reverted priority fails the test in seconds instead of
		// holding the suite for the default 90s
		previousTimeout := config.WAIT_GROUP_TIMEOUT_SECONDS
		config.WAIT_GROUP_TIMEOUT_SECONDS = 2
		t.Cleanup(func() { config.WAIT_GROUP_TIMEOUT_SECONDS = previousTimeout })

		// attached in the order a main.go calling sqlDB.Initialize before
		// messaging.Initialize produces, which is the order the phases have to override
		db := &dbLikeObserver{f: f}
		observer.AttachWithPriority(db, observer.PriorityDefault)
		observer.AttachWithPriority(&messagingObserver{}, observer.PriorityDefault)

		// the handler stays in flight until the consumer is told to stop, which only the
		// messaging observer does in the first phase: a database closed before the drain
		// finds the handler still running
		var c *consumer
		var handlerFinishedAt atomic.Int32
		started := make(chan struct{})

		c = startFakeConsumer(t, "db-order-queue", func(_ context.Context, _ *ProviderMessage) error {
			close(started)
			<-c.done
			handlerFinishedAt.Store(f.next())
			return nil
		})

		f.ch <- NewConsumerMessage("test", nil, nil, nil)

		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("message was never processed")
		}

		notifyWithin(t, 10*time.Second)

		assert.NotZero(t, handlerFinishedAt.Load(), "the message handler never finished")
		assert.NotZero(t, db.closedAt.Load(), "the database observer never closed")
		assert.Less(t, handlerFinishedAt.Load(), db.closedAt.Load(),
			"the database was closed while the message handler was still running")
	})
}

// shutdownWithin runs the module through the shutdown pipeline and reports how long it took.
// The observer only covers two of the phases, so the drain between them is run here the same
// way the observer notification does it.
func shutdownWithin(t *testing.T, timeout time.Duration) time.Duration {
	t.Helper()

	return runWithin(t, timeout, "module shutdown", func() {
		o := &messagingObserver{}
		o.Stop()
		observer.WaitRunningTimeout()
		o.Close()
	})
}

// notifyWithin runs the full observer notification, so the shutdown order between modules is
// exercised the same way a termination signal does it.
func notifyWithin(t *testing.T, timeout time.Duration) time.Duration {
	t.Helper()

	return runWithin(t, timeout, "observer notification", observer.Notify)
}

func runWithin(t *testing.T, timeout time.Duration, what string, fn func()) time.Duration {
	t.Helper()

	start := time.Now()
	done := make(chan time.Duration, 1)
	go func() {
		fn()
		done <- time.Since(start)
	}()

	select {
	case elapsed := <-done:
		return elapsed
	case <-time.After(timeout):
		t.Fatalf("%s did not return within %s", what, timeout)
		return 0
	}
}
