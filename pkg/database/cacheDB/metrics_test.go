package cacheDB

import (
	"context"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
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
		usage, ok := metrics["db.client.connections.usage"]
		require.True(t, ok, "pool usage was not reported")
		monitoringtest.AssertShape(t, usage, "", "db.system", "pool.name", "state")
		assert.Contains(t, metrics, "db.client.connections.max")
		assert.Contains(t, metrics, "db.client.connections.use_time")
	})

	t.Run("Should stop reporting the pool once the stop channel is closed", func(t *testing.T) {
		recorder := monitoringtest.Install(t)
		client := newMetricsTestClient(t)

		stop := instrumentMetrics(client)
		require.NotNil(t, stop)
		require.Contains(t, recorder.Collect(t), "db.client.connections.usage")

		close(stop)

		assert.Eventually(t, func() bool {
			_, ok := recorder.Collect(t)["db.client.connections.usage"]
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
		assert.NotContains(t, recorder.Collect(t), "db.client.connections.usage")
	})
}
