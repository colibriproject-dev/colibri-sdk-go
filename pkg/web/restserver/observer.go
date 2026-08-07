package restserver

import (
	"context"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
)

type restObserver struct {
}

// Stop makes the server refuse new connections. The fiber shutdown also waits for the
// requests still in flight, so it runs on the running counter and is left to the drain phase
// instead of being waited on here: the handlers finishing keep every resource they use open,
// and no per-request tracking is needed to cover them.
func (o restObserver) Stop() {
	ctx := context.Background()
	server := srv
	if server == nil {
		return
	}

	logging.Info(ctx).Msg("closing http server")

	observer.AddRunning()
	go func() {
		defer observer.DoneRunning()

		if err := server.shutdown(); err != nil {
			logging.Error(ctx).Err(err).Msg("error when closing http server")
		}
	}()
}

// Close releases the server. There is nothing left to wait for: the shutdown started by Stop
// was drained before this phase began.
func (o restObserver) Close() {
	srv = nil
}
