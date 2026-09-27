package cacheDB

import (
	"context"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/monitoringtest"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/test"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMetricsTestClient(t *testing.T) *redis.Client {
	t.Helper()

	client := redis.NewClient(&redis.Options{Addr: config.CACHE_URI, Password: config.CACHE_PASSWORD})
	t.Cleanup(func() { _ = client.Close() })

	return client
}

func TestInstrumentMetrics(t *testing.T) {
	test.InitializeCacheDBTest()

	t.Run("Should report the connection pool metrics", func(t *testing.T) {
		recorder := monitoringtest.Install(t)
		client := newMetricsTestClient(t)

		stop := instrumentMetrics(client)
		require.NotNil(t, stop)
		defer close(stop)

		require.NoError(t, client.Ping(context.Background()).Err())

		metrics := recorder.Collect(t)
		require.Contains(t, metrics, monitoring.MetricDBClientConnectionsUsage, "pool usage was not reported")
		assert.Contains(t, metrics, monitoring.MetricDBClientConnectionsMax)
		assert.Contains(t, metrics, monitoring.MetricDBClientConnectionsUseTime)
		for _, m := range metrics {
			monitoringtest.AssertCataloged(t, m)
		}
	})

	t.Run("Should stop reporting the pool once the stop channel is closed", func(t *testing.T) {
		recorder := monitoringtest.Install(t)
		client := newMetricsTestClient(t)

		stop := instrumentMetrics(client)
		require.NotNil(t, stop)
		require.Contains(t, recorder.Collect(t), monitoring.MetricDBClientConnectionsUsage)

		close(stop)

		assert.Eventually(t, func() bool {
			_, ok := recorder.Collect(t)[monitoring.MetricDBClientConnectionsUsage]
			return !ok
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("Should not instrument when metrics are disabled", func(t *testing.T) {
		previous := config.OTEL_METRICS_ENABLED
		config.OTEL_METRICS_ENABLED = false
		defer func() { config.OTEL_METRICS_ENABLED = previous }()

		recorder := monitoringtest.Install(t)
		client := newMetricsTestClient(t)

		assert.Nil(t, instrumentMetrics(client))
		require.NoError(t, client.Ping(context.Background()).Err())
		assert.NotContains(t, recorder.Collect(t), monitoring.MetricDBClientConnectionsUsage)
	})
}
