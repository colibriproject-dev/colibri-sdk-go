package messaging

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConfirmation implements the piece of *amqp.DeferredConfirmation the producer
// waits on. The concrete amqp type cannot be built outside the library, which is
// why the provider narrows the publish behind rabbitChannel.confirmPublish.
type fakeConfirmation struct {
	acked bool
	err   error
}

func (f fakeConfirmation) WaitContext(context.Context) (bool, error) { return f.acked, f.err }

// fakeChannel is a rabbitChannel double: it records what the provider asked the
// broker for and hands back canned answers instead of reaching a live server.
type fakeChannel struct {
	mu sync.Mutex

	qosCalls int

	// deliveries feeds Consume: the fake returns the channel queued for the name,
	// or a closed one so processMessages follows the connection-lost branch
	deliveries map[string]<-chan amqp.Delivery
	// openDeliveries are the unconsumed delivery channels the test created, closed
	// by the cleanup once the consumers have stopped
	openDeliveries []chan amqp.Delivery
	consumed       []string

	// returnsChans are closed by the test cleanup so the watchReturns goroutines end
	returnsChans []chan amqp.Return

	published     []amqp.Publishing
	mandatory     []bool
	confirmations []*fakeConfirmation
	publishErr    error
	// skipConfirm makes confirmPublish return a nil confirmation, the branch the
	// real channel only takes when it is not in confirm mode
	skipConfirm bool

	closed bool
}

func (f *fakeChannel) Qos(int, int, bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.qosCalls++
	return nil
}

func (f *fakeChannel) Consume(queue, _ string, _, _, _, _ bool, _ amqp.Table) (<-chan amqp.Delivery, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.consumed = append(f.consumed, queue)
	if msgs, ok := f.deliveries[queue]; ok {
		return msgs, nil
	}
	closed := make(chan amqp.Delivery)
	close(closed)
	return closed, nil
}

func (f *fakeChannel) NotifyReturn(c chan amqp.Return) chan amqp.Return {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.returnsChans = append(f.returnsChans, c)
	return c
}

func (f *fakeChannel) confirmPublish(_ context.Context, _, _ string, mandatory, _ bool, msg amqp.Publishing) (*rabbitDeferredConfirmation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.publishErr != nil {
		return nil, f.publishErr
	}
	f.published = append(f.published, msg)
	f.mandatory = append(f.mandatory, mandatory)
	if f.skipConfirm {
		return nil, nil
	}
	// hand out the confirmation the test queued for this publish; an empty
	// default would ack nothing and every publish would look nacked
	if len(f.confirmations) == 0 {
		f.confirmations = append(f.confirmations, &fakeConfirmation{})
	}
	return &rabbitDeferredConfirmation{deferredConfirmation: f.confirmations[0]}, nil
}

func (f *fakeChannel) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

// closeDeliveries ends every delivery stream the fake handed out, so the
// processMessages goroutines blocked on them return
func (f *fakeChannel) closeDeliveries() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.openDeliveries {
		close(c)
	}
	f.openDeliveries = nil
}

func (f *fakeChannel) consumedQueues() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.consumed...)
}

// fakeConnection is a rabbitConnection double; the test cleanup closes the
// channels it handed to NotifyClose so the watchClose goroutines end.
type fakeConnection struct {
	mu            sync.Mutex
	closeNotifies []chan *amqp.Error
	closed        bool
}

func (f *fakeConnection) NotifyClose(c chan *amqp.Error) chan *amqp.Error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closeNotifies = append(f.closeNotifies, c)
	return c
}

func (f *fakeConnection) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

// newFakeChannelMessaging builds a provider wired to fakes. Nothing dials, so the
// whole file runs without Docker.
func newFakeChannelMessaging(t *testing.T, ch *fakeChannel) *rabbitMQMessaging {
	logging.Initialize()

	t.Cleanup(func() {
		ch.mu.Lock()
		defer ch.mu.Unlock()
		for _, c := range ch.returnsChans {
			close(c)
		}
	})

	return &rabbitMQMessaging{ch: ch, stop: make(chan struct{})}
}

func TestRabbitMQProducerConfirmation(t *testing.T) {
	t.Run("Should return nil when the broker confirms the publish", func(t *testing.T) {
		ch := &fakeChannel{confirmations: []*fakeConfirmation{{acked: true}}}
		m := newFakeChannelMessaging(t, ch)

		err := m.producer(context.Background(), NewProducer("topic"), NewProviderMessage(context.Background(), "test", nil))

		assert.NoError(t, err)
		assert.Len(t, ch.published, 1)
		// mandatory publishes are what makes an unroutable message come back
		// through NotifyReturn instead of vanishing
		assert.Equal(t, []bool{true}, ch.mandatory)
	})

	t.Run("Should return an error when the broker nacks the publish", func(t *testing.T) {
		ch := &fakeChannel{confirmations: []*fakeConfirmation{{acked: false}}}
		m := newFakeChannelMessaging(t, ch)
		msg := NewProviderMessage(context.Background(), "test", nil)

		err := m.producer(context.Background(), NewProducer("topic"), msg)

		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), msg.ID.String())
			assert.Contains(t, err.Error(), "topic")
		}
	})

	t.Run("Should return an error when the confirmation wait times out", func(t *testing.T) {
		ch := &fakeChannel{confirmations: []*fakeConfirmation{{err: context.DeadlineExceeded}}}
		m := newFakeChannelMessaging(t, ch)

		err := m.producer(context.Background(), NewProducer("topic"), NewProviderMessage(context.Background(), "test", nil))

		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("Should return the publish error without waiting for a confirmation", func(t *testing.T) {
		publishErr := errors.New("channel is closed")
		ch := &fakeChannel{publishErr: publishErr}
		m := newFakeChannelMessaging(t, ch)

		err := m.producer(context.Background(), NewProducer("topic"), NewProviderMessage(context.Background(), "test", nil))

		assert.ErrorIs(t, err, publishErr)
		assert.Empty(t, ch.published)
	})

	t.Run("Should not fail when the channel is not in confirm mode", func(t *testing.T) {
		ch := &fakeChannel{skipConfirm: true}
		m := newFakeChannelMessaging(t, ch)

		err := m.producer(context.Background(), NewProducer("topic"), NewProviderMessage(context.Background(), "test", nil))

		assert.NoError(t, err)
		assert.Len(t, ch.published, 1)
	})
}

func TestRabbitMQReconnectResubscribesLiveSubs(t *testing.T) {
	t.Run("Should re-subscribe only the live subscriptions and drop canceled ones", func(t *testing.T) {
		logging.Initialize()

		live := &consumer{queue: "live-queue", done: make(chan any)}
		stopped := &consumer{queue: "stopped-queue", done: make(chan any)}
		liveMsgs := make(chan *ProviderMessage, 1)
		// kept open so the re-subscribed processMessages stays alive until the test
		// stops the consumer; a closed stream would make it return through the
		// connection-lost branch instead of removeSub
		liveDeliveries := make(chan amqp.Delivery)

		oldConn := &fakeConnection{}
		oldCh := &fakeChannel{}
		newConn := &fakeConnection{}
		newCh := &fakeChannel{deliveries: map[string]<-chan amqp.Delivery{"live-queue": liveDeliveries}}

		m := &rabbitMQMessaging{
			url:  "amqp://fake",
			conn: oldConn,
			ch:   oldCh,
			stop: make(chan struct{}),
			subs: []*rabbitSubscription{
				{c: stopped, ctx: context.Background(), providerMsgs: make(chan *ProviderMessage, 1)},
				{c: live, ctx: context.Background(), providerMsgs: liveMsgs},
			},
		}
		m.dial = func(string) (rabbitConnection, rabbitChannel, error) {
			return newConn, newCh, nil
		}
		m.watchReturns(oldCh)
		m.watchClose(oldConn)
		t.Cleanup(func() {
			oldCh.mu.Lock()
			newCh.mu.Lock()
			oldConn.mu.Lock()
			newConn.mu.Lock()
			defer func() {
				newConn.mu.Unlock()
				oldConn.mu.Unlock()
				newCh.mu.Unlock()
				oldCh.mu.Unlock()
			}()
			for _, c := range append(oldCh.returnsChans, newCh.returnsChans...) {
				close(c)
			}
			for _, c := range append(oldConn.closeNotifies, newConn.closeNotifies...) {
				close(c)
			}
		})

		// the DeleteFunc race: the consumer was already canceled, but its
		// processMessages has not run removeSub yet, so it is still in subs when
		// the reconnect starts
		close(stopped.done)

		m.reconnect()

		assert.Equal(t, newCh, m.ch)
		assert.Equal(t, newConn, m.conn)
		// only the live queue is re-subscribed on the replacement channel, and it
		// keeps the same providerMsgs its listener is blocked on
		assert.Equal(t, []string{"live-queue"}, newCh.consumedQueues())
		assert.Equal(t, 1, newCh.qosCalls)

		m.mu.RLock()
		subs := append([]*rabbitSubscription(nil), m.subs...)
		m.mu.RUnlock()
		require.Len(t, subs, 1)
		assert.Same(t, live, subs[0].c)
		assert.Equal(t, liveMsgs, subs[0].providerMsgs)

		// once the surviving processMessages observes done it drops the last sub
		close(live.done)
		assert.Eventually(t, func() bool {
			m.mu.RLock()
			defer m.mu.RUnlock()
			return len(m.subs) == 0
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("Should keep a subscription whose delivery channel closed with the connection", func(t *testing.T) {
		logging.Initialize()

		c := &consumer{queue: "test-queue", done: make(chan any)}
		deliveries := make(chan amqp.Delivery)
		close(deliveries)

		oldConn := &fakeConnection{}
		oldCh := &fakeChannel{deliveries: map[string]<-chan amqp.Delivery{"test-queue": deliveries}}
		newConn := &fakeConnection{}
		newCh := &fakeChannel{}

		m := &rabbitMQMessaging{
			url:  "amqp://fake",
			conn: oldConn,
			ch:   oldCh,
			stop: make(chan struct{}),
		}
		m.dial = func(string) (rabbitConnection, rabbitChannel, error) {
			return newConn, newCh, nil
		}
		t.Cleanup(func() {
			close(c.done)

			oldCh.mu.Lock()
			newCh.mu.Lock()
			oldConn.mu.Lock()
			newConn.mu.Lock()
			defer func() {
				newConn.mu.Unlock()
				oldConn.mu.Unlock()
				newCh.mu.Unlock()
				oldCh.mu.Unlock()
			}()
			for _, rc := range append(oldCh.returnsChans, newCh.returnsChans...) {
				close(rc)
			}
			for _, cn := range append(oldConn.closeNotifies, newConn.closeNotifies...) {
				close(cn)
			}
		})

		// the consumer is live: its processMessages returned when the delivery
		// channel closed, but the subscription stayed for the reconnect
		_, err := m.consumer(context.Background(), c)
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			m.mu.RLock()
			defer m.mu.RUnlock()
			return len(m.subs) == 1
		}, time.Second, 10*time.Millisecond)

		m.reconnect()

		assert.Equal(t, []string{"test-queue"}, newCh.consumedQueues())
		m.mu.RLock()
		defer m.mu.RUnlock()
		require.Len(t, m.subs, 1)
		assert.Same(t, c, m.subs[0].c)
	})

	t.Run("Should retry with a delay after a failed dial and install the replacement", func(t *testing.T) {
		logging.Initialize()

		newConn := &fakeConnection{}
		newCh := &fakeChannel{}

		var mu sync.Mutex
		dials := 0

		m := &rabbitMQMessaging{
			url:  "amqp://fake",
			stop: make(chan struct{}),
		}
		m.dial = func(string) (rabbitConnection, rabbitChannel, error) {
			mu.Lock()
			defer mu.Unlock()
			dials++
			if dials == 1 {
				return nil, nil, errors.New("connection refused")
			}
			return newConn, newCh, nil
		}
		t.Cleanup(func() {
			for _, c := range newCh.returnsChans {
				close(c)
			}
			for _, c := range newConn.closeNotifies {
				close(c)
			}
		})

		done := make(chan struct{})
		go func() {
			defer close(done)
			m.reconnect()
		}()

		select {
		case <-done:
		case <-time.After(2*reconnectInitialDelay + time.Second):
			t.Fatal("reconnect did not retry after the backoff delay")
		}

		mu.Lock()
		assert.Equal(t, 2, dials)
		mu.Unlock()
		assert.Equal(t, newCh, m.ch)
	})

	t.Run("Should give up and close the replacement when closed mid-reconnect", func(t *testing.T) {
		logging.Initialize()

		newConn := &fakeConnection{}
		newCh := &fakeChannel{}
		dialed := make(chan struct{})

		m := &rabbitMQMessaging{
			url:  "amqp://fake",
			stop: make(chan struct{}),
		}
		m.dial = func(string) (rabbitConnection, rabbitChannel, error) {
			select {
			case <-m.stop:
				return nil, nil, errors.New("connection refused")
			default:
			}
			// the first attempt fails so the loop parks in the backoff wait,
			// which is where close() has to interrupt it
			close(dialed)
			return nil, nil, errors.New("connection refused")
		}

		go m.reconnect()
		<-dialed

		require.NoError(t, m.close())

		// close() flipped closing before the loop redialed, so the reconnect is
		// over and nothing was installed
		assert.True(t, m.closing)
		assert.Nil(t, m.ch)
		assert.False(t, newCh.closed)
		assert.False(t, newConn.closed)
	})
}
