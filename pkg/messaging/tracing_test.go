package messaging

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spyTransaction is what spyMonitoring records for each transaction started.
type spyTransaction struct {
	name  string
	kind  colibrimonitoringbase.SpanKind
	attrs map[string]string
}

// spyMonitoring records the transactions and their attributes, delegating everything else
// to the no-op implementation.
type spyMonitoring struct {
	colibrimonitoringbase.Monitoring

	mu   sync.Mutex
	txns []*spyTransaction
}

func installSpyMonitoring(t *testing.T) *spyMonitoring {
	t.Helper()

	spy := &spyMonitoring{Monitoring: colibrimonitoringbase.NewOthers()}
	t.Cleanup(monitoring.ReplaceInstance(spy))

	return spy
}

func (s *spyMonitoring) StartTransaction(ctx context.Context, name string, kind colibrimonitoringbase.SpanKind) (any, context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	txn := &spyTransaction{name: name, kind: kind, attrs: map[string]string{}}
	s.txns = append(s.txns, txn)

	return txn, ctx
}

func (s *spyMonitoring) AddTransactionAttribute(transaction any, key, value string) {
	txn, ok := transaction.(*spyTransaction)
	if !ok {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	txn.attrs[key] = value
}

// transaction returns a copy of the attributes of the only transaction of the given kind.
func (s *spyMonitoring) transaction(t *testing.T, kind colibrimonitoringbase.SpanKind) map[string]string {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	var found []*spyTransaction
	for _, txn := range s.txns {
		if txn.kind == kind {
			found = append(found, txn)
		}
	}
	require.Len(t, found, 1, "expected exactly one %s transaction", kind)

	attrs := make(map[string]string, len(found[0].attrs))
	for k, v := range found[0].attrs {
		attrs[k] = v
	}

	return attrs
}

func useBrokerConfig(t *testing.T, messaging, cloud string) {
	t.Helper()

	previousMessaging, previousCloud := config.COLIBRI_MESSAGING, config.CLOUD
	t.Cleanup(func() {
		config.COLIBRI_MESSAGING, config.CLOUD = previousMessaging, previousCloud
	})
	config.COLIBRI_MESSAGING, config.CLOUD = messaging, cloud
}

func TestBrokerSystems(t *testing.T) {
	tests := []struct {
		name      string
		messaging string
		cloud     string
		publish   string
		consume   string
	}{
		{"rabbitmq wins over the cloud", config.MESSAGING_RABBITMQ, config.CLOUD_AWS, systemRabbitMQ, systemRabbitMQ},
		{"aws publishes to sns and consumes from sqs", config.MESSAGING_CLOUD_DEFAULT, config.CLOUD_AWS, systemAWSSNS, systemAWSSQS},
		{"gcp", config.MESSAGING_CLOUD_DEFAULT, config.CLOUD_GCP, systemGCPPubSub, systemGCPPubSub},
		{"firebase", config.MESSAGING_CLOUD_DEFAULT, config.CLOUD_FIREBASE, systemGCPPubSub, systemGCPPubSub},
		{"no supported broker", config.MESSAGING_CLOUD_DEFAULT, config.CLOUD_NONE, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useBrokerConfig(t, tt.messaging, tt.cloud)

			publish, consume := brokerSystems()

			assert.Equal(t, tt.publish, publish)
			assert.Equal(t, tt.consume, consume)
		})
	}
}

func TestConsumerTransactionCarriesMessagingAttributes(t *testing.T) {
	t.Run("Should name the destination so backends do not report the consumer as unknown", func(t *testing.T) {
		f := setupMessagingTest(t)
		useBrokerConfig(t, config.MESSAGING_RABBITMQ, config.CLOUD_NONE)
		spy := installSpyMonitoring(t)

		processed := make(chan struct{}, 1)
		c := startFakeConsumer(t, "orders-queue", func(_ context.Context, _ *ProviderMessage) error {
			processed <- struct{}{}
			return nil
		})

		msg := NewConsumerMessage("created", nil, nil, nil)
		f.ch <- msg

		select {
		case <-processed:
		case <-time.After(2 * time.Second):
			t.Fatal("the message was not processed")
		}
		closeWithin(t, c, 5*time.Second)

		attrs := spy.transaction(t, colibrimonitoringbase.SpanKindConsumer)
		assert.Equal(t, systemRabbitMQ, attrs[attrMessagingSystem])
		assert.Equal(t, "orders-queue", attrs[attrMessagingDestinationName])
		assert.Equal(t, operationTypeProcess, attrs[attrMessagingOperationType])
		assert.Equal(t, operationNameProcess, attrs[attrMessagingOperationName])
		assert.Equal(t, msg.ID.String(), attrs[attrMessagingMessageID])
	})
}

func TestProducerTransactionCarriesMessagingAttributes(t *testing.T) {
	t.Run("Should name the destination topic of the publish", func(t *testing.T) {
		setupMessagingTest(t)
		useBrokerConfig(t, config.MESSAGING_CLOUD_DEFAULT, config.CLOUD_AWS)
		spy := installSpyMonitoring(t)

		require.NoError(t, NewProducer("orders-topic").Publish(context.Background(), "created", map[string]string{"id": "1"}))

		attrs := spy.transaction(t, colibrimonitoringbase.SpanKindProducer)
		assert.Equal(t, systemAWSSNS, attrs[attrMessagingSystem])
		assert.Equal(t, "orders-topic", attrs[attrMessagingDestinationName])
		assert.Equal(t, operationTypeSend, attrs[attrMessagingOperationType])
		assert.Equal(t, operationNamePublish, attrs[attrMessagingOperationName])
		assert.NotEmpty(t, attrs[attrMessagingMessageID])
	})

	t.Run("Should leave messaging.system out when no broker is configured", func(t *testing.T) {
		setupMessagingTest(t)
		useBrokerConfig(t, config.MESSAGING_CLOUD_DEFAULT, config.CLOUD_NONE)
		spy := installSpyMonitoring(t)

		require.NoError(t, NewProducer("orders-topic").Publish(context.Background(), "created", nil))

		attrs := spy.transaction(t, colibrimonitoringbase.SpanKindProducer)
		assert.NotContains(t, attrs, attrMessagingSystem)
		assert.Equal(t, "orders-topic", attrs[attrMessagingDestinationName])
	})
}
