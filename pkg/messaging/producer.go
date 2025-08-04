package messaging

import (
	"context"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/security"
	"github.com/google/uuid"
)

type Producer struct {
	topic string
}

func NewProducer(topicName string) *Producer {
	return &Producer{topicName}
}

func (p *Producer) Publish(ctx context.Context, action string, message any) error {
	if instance == nil {
		logging.Fatal(context.Background()).Msg(messagingNotInitialized)
	}

	txn := monitoring.GetTransactionInContext(ctx)
	if txn != nil {
		segment := monitoring.StartTransactionSegment(ctx, messagingProducerTransaction, map[string]string{
			"topic": p.topic,
		})
		defer monitoring.EndTransactionSegment(segment)
	}

	msg := &ProviderMessage{
		ID:          uuid.New(),
		Origin:      config.APP_NAME,
		Action:      action,
		Message:     message,
		AuthContext: security.GetAuthenticationContext(ctx),
	}

	if err := instance.producer(ctx, p, msg); err != nil {
		logging.Error(ctx).Err(err).Msgf(couldNotSendMsg, msg.ID, p.topic)
		monitoring.NoticeError(txn, err)
		return err
	}

	return nil
}
