package messaging

import (
	"context"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/security"
	"github.com/google/uuid"
)

type Producer struct {
	topic string
}

func NewProducer(topicName string) *Producer {
	if instance == nil {
		logging.Fatal(context.Background()).Msg(messagingNotInitialized)
	}

	return &Producer{topicName}
}

func (p *Producer) Publish(ctx context.Context, action string, message any) error {
	txn := monitoring.GetTransactionInContext(ctx)

	if txn != nil {
		segment := monitoring.StartTransactionSegment(ctx, messagingProducerTransaction, map[string]string{
			"topic": p.topic,
		})
		defer monitoring.EndTransactionSegment(segment)
	}

	msg := &ProviderMessage{
		Id:      uuid.New(),
		Origin:  config.APP_NAME,
		Action:  action,
		Message: message,
	}

	authContext := security.GetAuthenticationContext(ctx)
	if authContext != nil {
		msg.TenantId = authContext.GetTenantID()
		msg.UserId = authContext.GetUserID()
	}

	if err := instance.producer(ctx, p, msg); err != nil {
		logging.Error(ctx).Err(err).Msgf(couldNotSendMsg, msg.Id, p.topic)
		monitoring.NoticeError(txn, err)
		return err
	}

	return nil
}
