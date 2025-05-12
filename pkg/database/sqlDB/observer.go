package sqlDB

import (
	"context"
	"database/sql"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
)

// sqlDBObserver is a struct for SQL database observer.
type sqlDBObserver struct {
	name     string
	instance *sql.DB
}

// Close finalize SQL database connection
//
// No parameters.
// No return values.
func (o sqlDBObserver) Close() {
	ctx := context.Background()
	logging.Info(ctx).Msgf(dbWaitingSafeClose, o.name)

	if observer.WaitRunningTimeout() {
		logging.Warn(ctx).Msgf(dbWaitingForceClose, o.name)
	}

	if err := o.instance.Close(); err != nil {
		logging.Error(ctx).Err(err).Msgf(dbCloseError, o.name)
	}

	logging.Info(ctx).Msgf(dbCloseSuccess, o.name)
}
