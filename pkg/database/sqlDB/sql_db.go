package sqlDB

import (
	"context"
	"database/sql"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
	_ "github.com/newrelic/go-agent/v3/integrations/nrpq"
)

const (
	db_default_name          string = "SQL"
	db_connection_success    string = "%s database connected"
	db_already_connected     string = "SQL database already connected"
	db_connection_error      string = "an error occurred while trying to connect to the %s database"
	db_migration_error       string = "an error occurred when validate database migrations"
	db_waiting_safe_close    string = "waiting to safely close the %s database connection"
	db_waiting_force_close   string = "waiting timed out, forcing to close the %s database connection"
	db_close_error           string = "error on closing the %s database connection"
	db_close_success         string = "%s database closed"
	db_not_initialized_error string = "database not initialized"
	query_is_empty_error     string = "query is empty"
	page_is_empty_error      string = "page is empty"
)

// sqlDBInstance is a pointer to sql.DB
var sqlDBInstance *sql.DB

// Initialize start connection with sql database and execute migration.
//
// No parameters.
// No return values.
func Initialize() {
	if sqlDBInstance != nil {
		logging.Info(context.Background()).Msg(db_already_connected)
		return
	}

	sqlDB := NewSQLDatabaseInstance(db_default_name, config.SQL_DB_CONNECTION_URI)
	sqlDB.SetMaxOpenConns(config.SQL_DB_MAX_OPEN_CONNS)
	sqlDB.SetMaxIdleConns(config.SQL_DB_MAX_IDLE_CONNS)

	if err := executeDatabaseMigration(sqlDB); err != nil {
		logging.Fatal(context.Background()).Err(err).Msg(db_migration_error)
	}

	sqlDBInstance = sqlDB
}

// NewSQLDatabaseInstance creates a new SQL database instance.
//
// Parameters:
// - name: a string representing the name of the database.
// - databaseURL: a string representing the URL of the database.
// Returns a pointer to sql.DB.
func NewSQLDatabaseInstance(name, databaseURL string) *sql.DB {
	sqlDB, err := sql.Open(monitoring.GetSQLDBDriverName(), databaseURL)
	if err != nil {
		logging.Fatal(context.Background()).Err(err).Msgf(db_connection_error, name)
	}

	if err = sqlDB.Ping(); err != nil {
		logging.Fatal(context.Background()).Err(err).Msgf(db_connection_error, name)
	}

	observer.Attach(sqlDBObserver{name, sqlDB})
	logging.Info(context.Background()).Msgf(db_connection_success, name)

	return sqlDB
}
