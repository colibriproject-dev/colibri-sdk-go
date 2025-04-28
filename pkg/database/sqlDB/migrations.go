package sqlDB

import (
	"context"
	"database/sql"
	"errors"
	"os"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	migrationSourceURLEnv       string = "MIGRATION_SOURCE_URL"
	migrationWithPwdDefaultPath string = "${PWD}/migrations"
	migrationDefaultPath        string = "./migrations"

	migrationIgnoringMsg              string = "Ignoring migration because env variable SQL_DB_MIGRATION is set to false"
	migrationEnvNotSetUsingDefaultMsg string = "Migration env variable %s is not set, using default value %s"
	migrationStartingMsg              string = "Starting migration execution"
	migrationCouldNotConnectDBMsg     string = "Could not connect to database for migration"
	migrationExecutingInfoMsg         string = "Executing migration on path: %s"
	migrationExecutionWithErrorMsg    string = "An error when executing database migration"
	migrationFinalizedMsg             string = "Migration finalized successfully"
)

// executeDatabaseMigration performs database migrations based on the provided source URL.
//
// It checks if the SQL_DB_MIGRATION environment variable is set to true before proceeding.
// It uses the MIGRATION_SOURCE_URL environment variable for migration source. If not set, it defaults to "./migrations".
// Returns an error if there is a failure during migration execution.
func executeDatabaseMigration(instance *sql.DB) error {
	if !config.SQL_DB_MIGRATION {
		logging.Info(context.Background()).Msg(migrationIgnoringMsg)
		return nil
	}

	sourceUrl := os.Getenv(migrationSourceURLEnv)
	if sourceUrl == "" {
		sourceUrl = migrationDefaultPath
	}

	logging.Info(context.Background()).Msg(migrationStartingMsg)
	driver, err := postgres.WithInstance(instance, &postgres.Config{})
	if err != nil {
		logging.Error(context.Background()).Err(err).Msg(migrationCouldNotConnectDBMsg)
		return err
	}

	logging.Info(context.Background()).Msgf(migrationExecutingInfoMsg, sourceUrl)
	migrateDatabaseInstance, _ := migrate.NewWithDatabaseInstance("file://"+sourceUrl, config.SQL_DB_NAME, driver)
	if migrateDatabaseInstance != nil {
		if err = migrateDatabaseInstance.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			logging.Error(context.Background()).Err(err).Msg(migrationExecutionWithErrorMsg)
			return err
		}
	}

	logging.Info(context.Background()).Msg(migrationFinalizedMsg)
	return nil
}
