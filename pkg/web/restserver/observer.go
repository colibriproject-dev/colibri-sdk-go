package restserver

import (
	"context"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

type restObserver struct {
}

// Close shuts the http server down. It runs in the closing phase of the graceful shutdown,
// so the work started by the requests already accepted has been drained; the shutdown itself
// waits for the requests still in flight, bounded by its own timeout.
func (o restObserver) Close() {
	ctx := context.Background()

	logging.Info(ctx).Msg("closing http server")
	if err := srv.shutdown(); err != nil {
		logging.Error(ctx).Err(err).Msg("error when closing http server")
	}

	srv = nil
}
