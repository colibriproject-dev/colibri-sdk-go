package messaging

import (
	"context"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
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
	// read once through the accessor: the module may be swapped or shut down while a handler
	// still draining publishes, and a direct read of the global would race with it
	provider := moduleInstance()
	if provider == nil {
		logging.Fatal(context.Background()).Msg(messagingNotInitialized)
	}

	correlationID := ctx.Value(logging.CorrelationIDParam)
	if correlationID == nil {
		correlationID = uuid.New().String()
	}

	txn, _ := monitoring.StartTransaction(ctx, messagingProducerTransaction, colibrimonitoringbase.SpanKindProducer)
	monitoring.AddTransactionAttribute(txn, "topic", p.topic)
	monitoring.AddTransactionAttribute(txn, "correlationId", correlationID.(string))
	monitoring.AddTransactionAttribute(txn, "action", action)
	defer monitoring.EndTransaction(txn)

	msg := &ProviderMessage{
		ID:            uuid.New(),
		Origin:        config.APP_NAME,
		Action:        action,
		Message:       message,
		AuthContext:   security.GetAuthenticationContext(ctx),
		CorrelationID: correlationID.(string),
	}

	if err := provider.producer(ctx, p, msg); err != nil {
		logging.Error(ctx).Err(err).Msgf(couldNotSendMsg, msg.ID, p.topic)
		monitoring.NoticeError(txn, err)
		return err
	}

	return nil
}
