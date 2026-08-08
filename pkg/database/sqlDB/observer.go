package sqlDB

import (
	"context"
	"database/sql"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

// sqlDBObserver is a struct for SQL database observer.
type sqlDBObserver struct {
	name     string
	instance *sql.DB
}

// Close finalize SQL database connection. It runs in the closing phase of the graceful
// shutdown, so the work that queries the database has already been drained.
//
// No parameters.
// No return values.
func (o sqlDBObserver) Close() {
	ctx := context.Background()
	logging.Info(ctx).Msgf(dbWaitingSafeClose, o.name)

	if err := o.instance.Close(); err != nil {
		logging.Error(ctx).Err(err).Msgf(dbCloseError, o.name)
	}

	logging.Info(ctx).Msgf(dbCloseSuccess, o.name)
}
