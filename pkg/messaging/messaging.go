package messaging

import (
	"context"
	"slices"
	"sync"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
)

const (
	messagingProducerTransaction string = "producer"
	messagingConsumerTransaction string = "consumer-%s"
	messagingNotInitialized      string = "messaging has not been initialized. add in main.go `messaging.Initialize()`"
	messagingAlreadyConnected    string = "message broker already connected"
	messagingConnected           string = "message broker connected"
	queueNotFound                string = "queue %s not found"
	connectionError              string = "an error occurred when trying to connect to the message broker"
	createQueueError             string = "an error occurred when trying to create a consumer to queue %s"
	closingQueueConsumer         string = "closing queue consumer %s"
	couldNotConnectQueue         string = "could not connect to queue %s"
	couldNotReceiveMsg           string = "error on receive message from queue %s"
	couldNotProcessMsg           string = "could not process message %s"
	couldNotReadMsgBody          string = "could not read message body with id %s from queue %s"
	messagingNotSupported        string = "messaging is not supported for cloud %s"
	couldNotSendMsg              string = "could not send message with id %s to topic %s"
	couldNotCloseBroker          string = "an error occurred when trying to close the message broker connection"
	safelyCloseMsg               string = "waiting to safely close messaging module"
)

type messaging interface {
	producer(ctx context.Context, p *Producer, msg *ProviderMessage) error
	consumer(ctx context.Context, c *consumer) (chan *ProviderMessage, error)
}

// brokerCloser is implemented by the providers that hold a persistent connection to the
// broker. SQS opens no connection between calls, so the AWS provider does not implement it.
type brokerCloser interface {
	close() error
}

// Module state shared with the consumers, guarded by moduleMu. moduleCtx is handed to every
// provider so the shutdown can end long polls and receives that would otherwise keep
// running while the process is going down.
var (
	moduleMu      sync.RWMutex
	instance      messaging
	moduleCtx     context.Context
	moduleCancel  context.CancelFunc
	moduleClosing bool
	moduleClosed  bool
	consumers     []*consumer
)

type messagingObserver struct{}

// Stop blocks new work from entering the module: the consumers stop pulling messages and any
// consumer created from now on is refused. It does not wait for the messages being processed,
// the drain phase does. It is idempotent through the stopOnce of each consumer.
func (o *messagingObserver) Stop() {
	logging.Info(context.Background()).Msg(safelyCloseMsg)

	for _, c := range beginShutdown() {
		c.signalStop()
	}
}

// Close releases the broker. It runs in the closing phase, after the drain, because
// publishing is a side effect: a handler still finishing has to be able to publish, and so
// does anything else being closed. A message left unacknowledged when the drain timed out is
// redelivered by the broker.
func (o *messagingObserver) Close() {
	// Initialize attaches a fresh observer on every call, so the same module may be notified
	// more than once. A second run would close an already closed connection.
	if !beginClose() {
		return
	}

	ctx := context.Background()

	// end the long polls and receives still open
	if cancel := moduleCancelFunc(); cancel != nil {
		cancel()
	}

	// close the broker connection, which releases anything still buffered for redelivery
	if closer, ok := moduleInstance().(brokerCloser); ok {
		if err := closer.close(); err != nil {
			logging.Error(ctx).Err(err).Msg(couldNotCloseBroker)
		}
	}
}

// Initialize starts the messaging module and connects to the message broker.
//
// It reconnects a module that has been shut down. Looking only at the instance would make a
// second Initialize a no-op that leaves the closed state in place, and every consumer created
// from then on would be refused with ErrMessagingClosed.
func Initialize() {
	if alreadyConnected() {
		logging.Info(context.Background()).Msg(messagingAlreadyConnected)
		return
	}

	provider := newProvider()
	if provider == nil {
		logging.Fatal(context.Background()).Msgf(messagingNotSupported, config.CLOUD)
		return
	}

	openModule(provider)
	logging.Info(context.Background()).Msg(messagingConnected)
}

// alreadyConnected reports whether the module is up and serving. A module that has been
// through a shutdown keeps its instance, so the closed state is what tells the two apart.
func alreadyConnected() bool {
	moduleMu.RLock()
	defer moduleMu.RUnlock()

	return instance != nil && !moduleClosing && !moduleClosed
}

// newProvider builds the provider for the configured broker, or nil when the configuration
// names none this module supports.
func newProvider() messaging {
	if config.COLIBRI_MESSAGING == config.MESSAGING_RABBITMQ {
		return newRabbitMQMessaging()
	}

	switch config.CLOUD {
	case config.CLOUD_AWS:
		return newAwsMessaging()
	case config.CLOUD_GCP, config.CLOUD_FIREBASE:
		return newGcpMessaging()
	}

	return nil
}

// openModule publishes the provider and opens the module for consumers, whether it is coming
// up for the first time or back from a shutdown.
func openModule(provider messaging) {
	// the state is reset before the instance is published: a consumer created in between
	// would otherwise reach a provider through a nil moduleCtx and never be canceled
	resetModuleState()
	setInstance(provider)

	// the consumers stop in the first phase and the broker connection is released in the
	// closing one, so a single observer covers both ends of the shutdown
	observer.Attach(&messagingObserver{})
}

func resetModuleState() {
	moduleMu.Lock()
	defer moduleMu.Unlock()

	// releases the providers still bound to the previous context when the module is
	// initialized again without a shutdown in between
	if moduleCancel != nil {
		moduleCancel()
	}

	moduleClosing = false
	moduleClosed = false
	consumers = nil
	moduleCtx, moduleCancel = context.WithCancel(context.Background())
}

func setInstance(m messaging) {
	moduleMu.Lock()
	defer moduleMu.Unlock()

	instance = m
}

func moduleInstance() messaging {
	moduleMu.RLock()
	defer moduleMu.RUnlock()

	return instance
}

// registerConsumer records the consumer so the shutdown can signal it. It rejects
// registration once the shutdown has begun, otherwise the consumer would keep running with
// nobody left to stop it.
func registerConsumer(c *consumer) error {
	moduleMu.Lock()
	defer moduleMu.Unlock()

	if moduleClosing {
		return ErrMessagingClosed
	}

	consumers = append(consumers, c)

	return nil
}

// activateConsumer registers a consumer that is about to start listening as a module task.
// It shares the critical section with the moduleClosing check because the broker may have
// taken long enough to hand back its channel for the shutdown to have begun and drained
// already, and a counter raised after that drain is never waited for.
//
// A listener lives as long as the process, so it is a module task rather than application
// work: counting it as the latter would keep GetWaitGroup().Wait from ever returning.
func activateConsumer() error {
	moduleMu.Lock()
	defer moduleMu.Unlock()

	if moduleClosing {
		return ErrMessagingClosed
	}

	observer.AddModuleTask()

	return nil
}

// beginClose reports whether this call owns the release of the broker, so it happens only
// once per module lifecycle. Stopping needs no such guard: it is idempotent on its own.
func beginClose() bool {
	moduleMu.Lock()
	defer moduleMu.Unlock()

	if moduleClosed {
		return false
	}

	moduleClosed = true

	return true
}

func unregisterConsumer(c *consumer) {
	moduleMu.Lock()
	defer moduleMu.Unlock()

	consumers = slices.DeleteFunc(consumers, func(registered *consumer) bool {
		return registered == c
	})
}

// beginShutdown closes the module for new consumers and returns the ones to signal.
func beginShutdown() []*consumer {
	moduleMu.Lock()
	defer moduleMu.Unlock()

	moduleClosing = true

	return slices.Clone(consumers)
}

func moduleContext() context.Context {
	moduleMu.RLock()
	defer moduleMu.RUnlock()

	if moduleCtx == nil {
		return context.Background()
	}

	return moduleCtx
}

func moduleCancelFunc() context.CancelFunc {
	moduleMu.RLock()
	defer moduleMu.RUnlock()

	return moduleCancel
}
