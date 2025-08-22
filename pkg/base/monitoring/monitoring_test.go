package monitoring

import (
	"context"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/stretchr/testify/assert"
)

func TestProductionMonitoring_OT(t *testing.T) {
	config.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4318/v1/traces"
	config.APP_NAME = "test"
	config.ENVIRONMENT = config.ENVIRONMENT_PRODUCTION
	assert.True(t, config.IsProductionEnvironment())

	Initialize()
	assert.NotNil(t, instance)

	monitoringTest(t)
}

func monitoringTest(t *testing.T) {
	t.Run("Should get transaction in context", func(t *testing.T) {
		txnName := "txn-test"

		_, ctx := StartTransaction(context.Background(), txnName)
		transaction := GetTransactionInContext(ctx)
		EndTransaction(transaction)

		assert.NotNil(t, transaction)
	})
}
