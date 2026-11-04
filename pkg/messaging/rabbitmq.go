package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	rabbitMQDefaultURL = "amqp://guest:guest@localhost:5672/"
	rabbitMQURLEnvVar  = "RABBITMQ_URL"

	messageRejectedToDLQ = "message %s from queue %s rejected without requeue due to: %s"

	// publishTimeout bounds a single publish together with the broker confirmation
	// it now waits for.
	publishTimeout = 5 * time.Second
	// reconnectInitialDelay and reconnectMaxDelay bound the exponential backoff
	// between the reconnection attempts once the broker connection has been lost.
	reconnectInitialDelay = 1 * time.Second
	reconnectMaxDelay     = 30 * time.Second

	rabbitMQClosing        = "rabbitmq provider is closing"
	rabbitMQConnLost       = "rabbitmq connection lost; reconnecting"
	rabbitMQReconnectFail  = "rabbitmq reconnection failed; retrying in %s"
	rabbitMQReconnected    = "rabbitmq reconnected; re-subscribed %d queue(s)"
	rabbitMQResubscribeErr = "could not re-subscribe queue %s after reconnect"
	rabbitMQNotConfirmed   = "broker did not confirm (nack) message %s published to %s"
	rabbitMQNoConfirmMode  = "channel is not in confirm mode; message %s published without confirmation"
	rabbitMQReturned       = "message returned by the broker as unroutable: exchange=%s routingKey=%s reply=%s messageId=%s"
)

type rabbitMQMessaging struct {
	mu   sync.RWMutex
	url  string
	conn rabbitConnection
	ch   rabbitChannel

	// dial is the seam the unit tests swap to fake the broker; nil means the
	// real dialRabbitMQ. It must only be set while no reconnect is in flight.
	dial func(url string) (rabbitConnection, rabbitChannel, error)

	// subs are the live consumers, re-subscribed into their own providerMsgs channel
	// after a reconnect so the listener blocked on it never has to be rebuilt.
	subs    []*rabbitSubscription
	closing bool
	stop    chan struct{}
}

// rabbitChannel and rabbitConnection narrow the concrete amqp types to what the
// provider calls, so the unit tests can fake the broker: amqp091 gives no way to
// build a usable *amqp.Channel or a *DeferredConfirmation without a live server.
type (
	rabbitChannel interface {
		Qos(prefetchCount, prefetchSize int, global bool) error
		Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
		NotifyReturn(c chan amqp.Return) chan amqp.Return
		// confirmPublish publishes and returns the confirmation this exact publish
		// waits on; a nil confirmation with a nil error means the channel is not in
		// confirm mode
		confirmPublish(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) (*rabbitDeferredConfirmation, error)
		Close() error
	}

	rabbitConnection interface {
		NotifyClose(c chan *amqp.Error) chan *amqp.Error
		Close() error
	}
)

// rabbitDeferredConfirmation wraps *amqp.DeferredConfirmation behind the wait the
// producer performs, for the same reason as the interfaces above.
type rabbitDeferredConfirmation struct {
	deferredConfirmation interface {
		WaitContext(ctx context.Context) (bool, error)
	}
}

func (d *rabbitDeferredConfirmation) WaitContext(ctx context.Context) (bool, error) {
	return d.deferredConfirmation.WaitContext(ctx)
}

// amqpChannel and amqpConnection adapt the concrete amqp types to the seams above.
type (
	amqpChannel    struct{ ch *amqp.Channel }
	amqpConnection struct{ conn *amqp.Connection }
)

func (a amqpChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	return a.ch.Qos(prefetchCount, prefetchSize, global)
}

func (a amqpChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	return a.ch.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
}

func (a amqpChannel) NotifyReturn(c chan amqp.Return) chan amqp.Return { return a.ch.NotifyReturn(c) }

func (a amqpChannel) confirmPublish(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) (*rabbitDeferredConfirmation, error) {
	deferred, err := a.ch.PublishWithDeferredConfirmWithContext(ctx, exchange, key, mandatory, immediate, msg)
	if err != nil || deferred == nil {
		return nil, err
	}
	return &rabbitDeferredConfirmation{deferredConfirmation: deferred}, nil
}

func (a amqpChannel) Close() error { return a.ch.Close() }

func (a amqpConnection) NotifyClose(c chan *amqp.Error) chan *amqp.Error {
	return a.conn.NotifyClose(c)
}

func (a amqpConnection) Close() error { return a.conn.Close() }

// rabbitSubscription ties a running consumer to the outbound channel its messages
// are delivered on. The channel is created once and reused across reconnects, so
// consumer.listen keeps blocking on the same channel while the provider
// re-subscribes the queue underneath it.
type rabbitSubscription struct {
	c            *consumer
	ctx          context.Context
	providerMsgs chan *ProviderMessage
}

type rabbitMQOriginalMessage struct {
	d amqp.Delivery
}

func (r rabbitMQOriginalMessage) Ack(_ context.Context) error {
	return r.d.Ack(false)
}

func (r rabbitMQOriginalMessage) Nack(_ context.Context, requeue bool, _ error) error {
	return r.d.Reject(requeue)
}

func newRabbitMQMessaging() *rabbitMQMessaging {
	url := os.Getenv(rabbitMQURLEnvVar)
	if url == "" {
		url = rabbitMQDefaultURL
	}

	conn, ch, err := dialRabbitMQ(url)
	if err != nil {
		logging.Fatal(context.Background()).Err(err).Msg(connectionError)
	}

	m := &rabbitMQMessaging{
		url:  url,
		conn: conn,
		ch:   ch,
		dial: dialRabbitMQ,
		stop: make(chan struct{}),
	}
	m.watchReturns(ch)
	m.watchClose(conn)

	return m
}

// dialRabbitMQ opens the connection and a channel already in confirm mode, so
// every publish can wait for the broker to acknowledge it. Both values are
// non-nil when the error is nil; the reconnect loop relies on that.
func dialRabbitMQ(url string) (rabbitConnection, rabbitChannel, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}

	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, err
	}

	return amqpConnection{conn}, amqpChannel{ch}, nil
}

// watchReturns logs the messages the broker hands back because mandatory was set
// and no queue matched the routing key. The listener channel is closed when the
// AMQP channel closes, which ends the goroutine; a reconnect registers a fresh one
// on the replacement channel.
func (m *rabbitMQMessaging) watchReturns(ch rabbitChannel) {
	returns := ch.NotifyReturn(make(chan amqp.Return, 16))
	go func() {
		for r := range returns {
			logging.Error(context.Background()).Msgf(rabbitMQReturned, r.Exchange, r.RoutingKey, r.ReplyText, r.MessageId)
		}
	}()
}

// watchClose waits for the connection to drop and triggers the reconnection. It
// returns as soon as the provider is closing, so an intentional shutdown does not
// dial the broker again.
func (m *rabbitMQMessaging) watchClose(conn rabbitConnection) {
	notify := conn.NotifyClose(make(chan *amqp.Error, 1))
	go func() {
		select {
		case err, ok := <-notify:
			if !ok {
				return
			}
			m.mu.RLock()
			closing := m.closing
			m.mu.RUnlock()
			if closing {
				return
			}
			logging.Error(context.Background()).Err(err).Msg(rabbitMQConnLost)
			m.reconnect()
		case <-m.stop:
			return
		}
	}()
}

// reconnect redials the broker with an exponential backoff and, once it is back,
// swaps in the new connection and channel and re-subscribes every live consumer
// into the channel its listener is already blocked on. It gives up silently when
// the provider is closed while it is retrying.
func (m *rabbitMQMessaging) reconnect() {
	delay := reconnectInitialDelay
	for {
		select {
		case <-m.stop:
			return
		default:
		}

		dial := m.dial
		if dial == nil {
			dial = dialRabbitMQ
		}

		conn, ch, err := dial(m.url)
		if err != nil {
			logging.Warn(context.Background()).Err(err).Msgf(rabbitMQReconnectFail, delay)
			select {
			case <-time.After(delay):
			case <-m.stop:
				return
			}
			if delay < reconnectMaxDelay {
				delay = min(delay*2, reconnectMaxDelay)
			}
			continue
		}

		m.mu.Lock()
		if m.closing {
			m.mu.Unlock()
			_ = ch.Close()
			_ = conn.Close()
			return
		}

		m.conn = conn
		m.ch = ch
		// registered under the lock so no publish slips onto the new channel before
		// its return listener is in place, and no second close event is missed while
		// the consumers are being re-subscribed
		m.watchReturns(ch)
		m.watchClose(conn)

		// a consumer stopped while the broker was down is dropped instead of
		// re-subscribed; its listener has already returned on c.done
		m.subs = slices.DeleteFunc(m.subs, func(s *rabbitSubscription) bool {
			return s.c.isCanceled()
		})
		for _, s := range m.subs {
			msgs, err := startConsumingOn(ch, s.c.queue)
			if err != nil {
				logging.Error(context.Background()).Err(err).Msgf(rabbitMQResubscribeErr, s.c.queue)
				continue
			}
			go m.processMessages(s.ctx, s.c, msgs, s.providerMsgs)
		}
		count := len(m.subs)
		m.mu.Unlock()

		logging.Info(context.Background()).Msgf(rabbitMQReconnected, count)
		return
	}
}

// removeSub forgets a consumer that has been stopped, so a later reconnect does
// not re-subscribe a queue nobody is listening on anymore.
func (m *rabbitMQMessaging) removeSub(c *consumer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.subs = slices.DeleteFunc(m.subs, func(s *rabbitSubscription) bool {
		return s.c == c
	})
}

// close releases the channel and the connection during the graceful shutdown. Without it
// the deliveries stay in flight on the broker until the TCP connection eventually drops.
// It also stops the reconnection, so a broker that is down while the process exits is not
// dialed again, and it is idempotent because the observer may notify more than once.
func (m *rabbitMQMessaging) close() error {
	m.mu.Lock()
	if m.closing {
		m.mu.Unlock()
		return nil
	}
	m.closing = true
	close(m.stop)
	ch := m.ch
	conn := m.conn
	m.mu.Unlock()

	// both are nil while the broker is down and the reconnect has not installed
	// the replacement yet; there is nothing to close in that window
	if ch == nil || conn == nil {
		return nil
	}

	if err := ch.Close(); err != nil {
		logging.Error(context.Background()).Err(err).Msg("could not close the rabbitmq channel")
	}

	return conn.Close()
}

// producer publishes the message and waits for the broker to confirm it. The
// publish is mandatory, so a message matching no queue is handed to the
// NotifyReturn listener and logged instead of vanishing. The confirmation is
// awaited through the DeferredConfirmation, which correlates it to this exact
// publish even with concurrent producers; a nack, a wait that fails, or a
// connection lost mid-publish becomes an error the caller surfaces.
func (m *rabbitMQMessaging) producer(ctx context.Context, p *Producer, msg *ProviderMessage) error {
	body := []byte(msg.String())
	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	m.mu.RLock()
	ch := m.ch
	m.mu.RUnlock()

	deferred, err := ch.confirmPublish(
		ctx,
		p.topic,
		p.topic,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			MessageId:    msg.ID.String(),
			// read back by the consumer as the start of the consume lag
			Timestamp: time.Now(),
		},
	)
	if err != nil {
		return err
	}

	if deferred == nil {
		// the channel is always opened in confirm mode, so this only happens if that
		// regressed: the message went out but was not confirmed
		logging.Error(ctx).Msgf(rabbitMQNoConfirmMode, msg.ID)
		return nil
	}

	acked, err := deferred.WaitContext(ctx)
	if err != nil {
		return err
	}
	if !acked {
		return fmt.Errorf(rabbitMQNotConfirmed, msg.ID, p.topic)
	}

	return nil
}

func (m *rabbitMQMessaging) consumer(ctx context.Context, c *consumer) (chan *ProviderMessage, error) {
	providerMsgs := make(chan *ProviderMessage, 1)

	// the subscription and the Consume happen under the same lock so a reconnect
	// cannot swap the channel in between and re-subscribe this queue a second time
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closing {
		return nil, errors.New(rabbitMQClosing)
	}

	msgs, err := startConsumingOn(m.ch, c.queue)
	if err != nil {
		return nil, err
	}

	m.subs = append(m.subs, &rabbitSubscription{c: c, ctx: ctx, providerMsgs: providerMsgs})
	go m.processMessages(ctx, c, msgs, providerMsgs)

	return providerMsgs, nil
}

// startConsumingOn sets QoS and starts consuming from the queue on the given
// channel, which after a reconnect is the replacement rather than m.ch
func startConsumingOn(ch rabbitChannel, queueName string) (<-chan amqp.Delivery, error) {
	if err := ch.Qos(
		1,
		0,
		false,
	); err != nil {
		return nil, err
	}

	return ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
}

// processMessages handles incoming messages from RabbitMQ until the delivery
// channel closes or the consumer is canceled
func (m *rabbitMQMessaging) processMessages(ctx context.Context, c *consumer, msgs <-chan amqp.Delivery, providerMsgs chan<- *ProviderMessage) {
	for {
		select {
		case <-c.done:
			// the consumer was stopped: forget it so a reconnect does not revive it
			m.removeSub(c)
			return
		case d, ok := <-msgs:
			if !ok {
				// the delivery channel closed because the connection was lost: keep the
				// subscription so the reconnect re-subscribes into the same providerMsgs
				return
			}
			m.handleMessage(ctx, c, d, providerMsgs)
		}
	}
}

// handleMessage processes with a single message
func (m *rabbitMQMessaging) handleMessage(ctx context.Context, c *consumer, d amqp.Delivery, providerMsgs chan<- *ProviderMessage) {
	var pm ProviderMessage

	if err := json.Unmarshal(d.Body, &pm); err != nil {
		m.handleUnmarshalError(ctx, c, d, err)
		return
	}

	pm.addOriginBrokerNotification(rabbitMQOriginalMessage{d: d})
	pm.setReceiptMetadata(rabbitMQAttributes(d), rabbitMQDeliveryAttempt(d))
	pm.setPublishTime(d.Timestamp)

	select {
	case providerMsgs <- &pm:
	case <-c.done:
		// nobody is listening anymore; requeue so the message is not lost when the
		// connection closes. The reject makes it available again immediately, the same as
		// the Pub/Sub nack and unlike SQS, where it only reappears after the visibility
		// timeout
		if err := d.Reject(true); err != nil {
			logging.Error(ctx).Err(err).Msgf("could not requeue message %s", d.MessageId)
		}
	}
}

// rabbitMQAttributes stringifies the AMQP delivery headers into a uniform string map.
func rabbitMQAttributes(d amqp.Delivery) map[string]string {
	if len(d.Headers) == 0 {
		return nil
	}
	attrs := make(map[string]string, len(d.Headers))
	for k, v := range d.Headers {
		attrs[k] = fmt.Sprintf("%v", v)
	}
	return attrs
}

// rabbitMQDeliveryAttempt derives the delivery count from the x-death header the broker
// adds when a message is dead-lettered, when present.
func rabbitMQDeliveryAttempt(d amqp.Delivery) *int {
	death, ok := d.Headers["x-death"]
	if !ok {
		return nil
	}
	entries, ok := death.([]any)
	if !ok || len(entries) == 0 {
		return nil
	}
	first, ok := entries[0].(amqp.Table)
	if !ok {
		return nil
	}
	if count, ok := first["count"].(int64); ok {
		n := int(count)
		return &n
	}
	return nil
}

// handleUnmarshalError rejects a malformed message without requeue so the broker
// routes it to the configured DLX instead of redelivering or dropping it silently
func (m *rabbitMQMessaging) handleUnmarshalError(ctx context.Context, c *consumer, d amqp.Delivery, err error) {
	logging.Error(ctx).Err(err).Msgf(couldNotReadMsgBody, d.MessageId, c.queue)

	logging.Debug(ctx).Msgf(messageRejectedToDLQ, d.MessageId, c.queue, err.Error())
	if rejectErr := d.Reject(false); rejectErr != nil {
		logging.Error(ctx).Err(rejectErr).Msgf("could not reject message %s", d.MessageId)
	}
}
