package sqlDB

import (
	"context"
	"database/sql"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
)

const (
	dbDefaultName         string = "SQL"
	dbConnectionSuccess   string = "%s database connected"
	dbAlreadyConnected    string = "SQL database already connected"
	dbConnectionError     string = "an error occurred while trying to connect to the %s database"
	dbMigrationError      string = "an error occurred when validate database migrations"
	dbWaitingSafeClose    string = "waiting to safely close the %s database connection"
	dbWaitingForceClose   string = "waiting timed out, forcing to close the %s database connection"
	dbCloseError          string = "error on closing the %s database connection"
	dbCloseSuccess        string = "%s database closed"
	dbNotInitializedError string = "database not initialized"
	queryIsEmptyError     string = "query is empty"
	pageIsEmptyError      string = "page is empty"
)

// sqlDBInstance is a pointer to sql.DB
var sqlDBInstance *sql.DB

// Initialize start connection with sql database and execute migration.
//
// No parameters.
// No return values.
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

// NewSQLDatabaseInstance creates a new SQL database instance.
//
// Parameters:
// - name: a string representing the name of the database.
// - databaseURL: a string representing the URL of the database.
// Returns a pointer to sql.DB.
func NewSQLDatabaseInstance(name, databaseURL string) *sql.DB {
	sqlDB, err := sql.Open(monitoring.GetSQLDBDriverName(), databaseURL)
	if err != nil {
		logging.Fatal(context.Background()).Err(err).Msgf(dbConnectionError, name)
	}

	if err = sqlDB.Ping(); err != nil {
		logging.Fatal(context.Background()).Err(err).Msgf(dbConnectionError, name)
	}

	observer.Attach(sqlDBObserver{name, sqlDB})
	logging.Info(context.Background()).Msgf(dbConnectionSuccess, name)

	return sqlDB
}
