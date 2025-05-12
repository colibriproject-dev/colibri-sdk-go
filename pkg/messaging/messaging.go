package messaging

import (
	"context"

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
	couldNotDeleteMsg            string = "could not delete message with id %s from queue %s"
	couldNotSendMsg              string = "could not send message with id %s to topic %s"
	safelyCloseMsg               string = "waiting to safely close messaging module"
	timeoutCloseMsg              string = "waiting timed out, forcing close the messaging module"
)

type messaging interface {
	producer(ctx context.Context, p *Producer, msg *ProviderMessage) error
	consumer(ctx context.Context, c *consumer) (chan *ProviderMessage, error)
}

var instance messaging

type messagingObserver struct {
	closed bool
}

func (o *messagingObserver) Close() {
	ctx := context.Background()

	logging.Info(ctx).Msg(safelyCloseMsg)
	if observer.WaitRunningTimeout() {
		logging.Warn(ctx).Msg(timeoutCloseMsg)
	}

	o.closed = true
}

func Initialize() {
	if instance != nil {
		logging.Info(context.Background()).Msg(messagingAlreadyConnected)
		return
	}

	switch config.CLOUD {
	case config.CLOUD_AWS:
		instance = newAwsMessaging()
	case config.CLOUD_GCP, config.CLOUD_FIREBASE:
		instance = newGcpMessaging()
	}

	observer.Attach(&messagingObserver{})
	logging.Info(context.Background()).Msg(messagingConnected)
}
