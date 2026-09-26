package sqlDB

import (
	"database/sql"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/monitoringtest"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The pool gauges otelsql reports, all tagged with the instance and the database system.
var poolGauges = []string{
	"db.sql.connections.open",
	"db.sql.connections.idle",
	"db.sql.connections.active",
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
			m, ok := metrics[name]
			require.Truef(t, ok, "%s was not reported", name)
			monitoringtest.AssertShape(t, m, "1", "db.instance", "db.system.name")
		}
		assert.Contains(t, metrics, "db.sql.connections.wait_count")
		assert.Contains(t, metrics, "db.sql.connections.wait_duration")
	})

	t.Run("Should not report the pool when metrics are disabled", func(t *testing.T) {
		previous := config.OTEL_METRICS_ENABLED
		config.OTEL_METRICS_ENABLED = false
		defer func() { config.OTEL_METRICS_ENABLED = previous }()

		recorder := monitoringtest.Install(t)
		db := openMetricsTestDB(t)

		recordPoolStats(db, "disabled-db")

		assert.NotContains(t, recorder.Collect(t), "db.sql.connections.open")
	})
}
