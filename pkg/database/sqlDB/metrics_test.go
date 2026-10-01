package sqlDB

import (
	"database/sql"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/monitoringtest"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The pool gauges otelsql reports, all tagged with the instance and the database system.
var poolGauges = []string{
	monitoring.MetricDBSQLConnectionsOpen,
	monitoring.MetricDBSQLConnectionsIdle,
	monitoring.MetricDBSQLConnectionsActive,
}

func openMetricsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("postgres", config.SQL_DB_CONNECTION_URI)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.Ping())

	return db
}

func TestRecordPoolStats(t *testing.T) {
	test.InitializeSqlDBTest()

	t.Run("Should report the connection pool saturation", func(t *testing.T) {
		recorder := monitoringtest.Install(t)
		db := openMetricsTestDB(t)

		recordPoolStats(db, "metrics-db")

		metrics := recorder.Collect(t)
		for _, name := range poolGauges {
			require.Containsf(t, metrics, name, "%s was not reported", name)
		}
		assert.Contains(t, metrics, monitoring.MetricDBSQLConnectionsWaitCount)
		assert.Contains(t, metrics, monitoring.MetricDBSQLConnectionsWaitDuration)
		for _, m := range metrics {
			monitoringtest.AssertCataloged(t, m)
		}
	})

	t.Run("Should report the calls made through the instrumented driver as cataloged", func(t *testing.T) {
		recorder := monitoringtest.Install(t)
		db, err := sql.Open(monitoring.GetSQLDBDriverName(), config.SQL_DB_CONNECTION_URI)
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })

		_, err = db.Exec("SELECT 1")
		require.NoError(t, err)

		metrics := recorder.Collect(t)
		require.Contains(t, metrics, monitoring.MetricDBSQLClientCalls)
		require.Contains(t, metrics, monitoring.MetricDBSQLClientLatency)
		for _, m := range metrics {
			monitoringtest.AssertCataloged(t, m)
		}
	})

	t.Run("Should not report the pool when metrics are disabled", func(t *testing.T) {
		previous := config.OTEL_METRICS_ENABLED
		config.OTEL_METRICS_ENABLED = false
		defer func() { config.OTEL_METRICS_ENABLED = previous }()

		recorder := monitoringtest.Install(t)
		db := openMetricsTestDB(t)

		recordPoolStats(db, "disabled-db")

		assert.NotContains(t, recorder.Collect(t), monitoring.MetricDBSQLConnectionsOpen)
	})
}
