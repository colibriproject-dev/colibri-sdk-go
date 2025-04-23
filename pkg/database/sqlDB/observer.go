package sqlDB

import (
	"context"
	"database/sql"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/observer"
)

// sqlDBObserver is a struct for sql database observer.
type sqlDBObserver struct {
	name     string
	instance *sql.DB
}

// Close finalize sql database connection
//
// No parameters.
// No return values.
func (o sqlDBObserver) Close() {
	ctx := context.Background()
	logging.Info(ctx).Msgf(db_waiting_safe_close, o.name)

	if observer.WaitRunningTimeout() {
		logging.Warn(ctx).Msgf(db_waiting_force_close, o.name)
	}

	if err := o.instance.Close(); err != nil {
		logging.Error(ctx).Err(err).Msgf(db_close_error, o.name)
	}

	logging.Info(ctx).Msgf(db_close_success, o.name)
}
