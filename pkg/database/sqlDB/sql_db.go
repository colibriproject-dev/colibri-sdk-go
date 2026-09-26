package sqlDB

import (
	"context"
	"database/sql"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
	"go.nhat.io/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	dbDefaultName         string = "SQL"
	dbConnectionSuccess   string = "%s database connected"
	dbAlreadyConnected    string = "SQL database already connected"
	dbConnectionError     string = "an error occurred while trying to connect to the %s database"
	dbMigrationError      string = "an error occurred when validate database migrations"
	dbWaitingSafeClose    string = "waiting to safely close the %s database connection"
	dbCloseError          string = "error on closing the %s database connection"
	dbCloseSuccess        string = "%s database closed"
	dbNotInitializedError string = "database not initialized"
	queryIsEmptyError     string = "query is empty"
	pageIsEmptyError      string = "page is empty"
)

// sqlDBInstance is a pointer to sql.DB
var sqlDBInstance *sql.DB

// Initialize starts the connection with the SQL database and executes migrations.
func Initialize() {
	if sqlDBInstance != nil {
		logging.Info(context.Background()).Msg(dbAlreadyConnected)
		return
	}

	sqlDB := NewSQLDatabaseInstance(dbDefaultName, config.SQL_DB_CONNECTION_URI)
	sqlDB.SetMaxOpenConns(config.SQL_DB_MAX_OPEN_CONNS)
	sqlDB.SetMaxIdleConns(config.SQL_DB_MAX_IDLE_CONNS)

	if err := executeDatabaseMigration(sqlDB); err != nil {
		logging.Fatal(context.Background()).Err(err).Msg(dbMigrationError)
	}

	sqlDBInstance = sqlDB
}

// NewSQLDatabaseInstance creates a new SQL database instance with the given name and URL.
func NewSQLDatabaseInstance(name, databaseURL string) *sql.DB {
	sqlDB, err := sql.Open(monitoring.GetSQLDBDriverName(), databaseURL)
	if err != nil {
		logging.Fatal(context.Background()).Err(err).Msgf(dbConnectionError, name)
	}

	if err = sqlDB.Ping(); err != nil {
		logging.Fatal(context.Background()).Err(err).Msgf(dbConnectionError, name)
	}

	recordPoolStats(sqlDB, name)

	observer.Attach(sqlDBObserver{name, sqlDB})
	logging.Info(context.Background()).Msgf(dbConnectionSuccess, name)

	return sqlDB
}

// recordPoolStats reports the connection pool saturation (db.sql.connections.*: open, idle,
// active, waits) when the metric signal is enabled, tagged with the instance name so
// several databases stay apart. A failure costs the metrics only, so it is logged instead
// of aborting the boot.
//
// otelsql offers no way to stop the reporting, so a closed database keeps being observed
// with an empty pool until the process exits.
func recordPoolStats(db *sql.DB, name string) {
	if !monitoring.UseMetrics() {
		return
	}

	err := otelsql.RecordStats(db,
		otelsql.WithInstanceName(name),
		otelsql.WithSystem(semconv.DBSystemNamePostgreSQL),
	)
	if err != nil {
		logging.Warn(context.Background()).Err(err).Msgf("an error occurred while trying to record the %s database pool stats", name)
	}
}
