package monitoring

import (
	"context"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	colibri_monitoring_base "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductionMonitoring_OT(t *testing.T) {
	observer.Initialize()
	logging.Initialize()
	config.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4318/v1/traces"
	config.OTEL_EXPORTER_OTLP_METRICS_ENDPOINT = ""
	config.APP_NAME = "test"
	config.ENVIRONMENT = config.ENVIRONMENT_PRODUCTION
	assert.True(t, config.IsProductionEnvironment())

	Initialize()
	assert.NotNil(t, instance)

	monitoringTest(t)
	metricsTest(t)
}

func monitoringTest(t *testing.T) {
	t.Run("Should get transaction in context", func(t *testing.T) {
		txnName := "txn-test"

		_, ctx := StartTransaction(context.Background(), txnName, colibri_monitoring_base.SpanKindInternal)
		transaction := GetTransactionInContext(ctx)
		EndTransaction(transaction)

		assert.NotNil(t, transaction)
	})
}

func metricsTest(t *testing.T) {
	t.Run("Counter should return non-nil and record without panic", func(t *testing.T) {
		c := Counter("monitoring.test.counter", "Test counter", "1")
		require.NotNil(t, c)
		assert.NotPanics(t, func() {
			c.Add(context.Background(), 1, map[string]string{"test": "true"})
		})
	})

	t.Run("Histogram should return non-nil and record without panic", func(t *testing.T) {
		h := Histogram("monitoring.test.histogram", "Test histogram", "ms")
		require.NotNil(t, h)
		assert.NotPanics(t, func() {
			h.Record(context.Background(), 99.9, map[string]string{"test": "true"})
		})
	})

	t.Run("Gauge should return non-nil and record without panic", func(t *testing.T) {
		g := Gauge("monitoring.test.gauge", "Test gauge", "1")
		require.NotNil(t, g)
		assert.NotPanics(t, func() {
			g.Record(context.Background(), 1.0, nil)
		})
	})
}
