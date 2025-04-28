package restserver

import (
	"context"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
)

type restObserver struct {
}

func (o restObserver) Close() {
	ctx := context.Background()

	logging.Info(ctx).Msg("waiting to safely close the http server")
	if observer.WaitRunningTimeout() {
		logging.Warn(ctx).Msg("WaitGroup timed out, forcing close http server")
	}

	logging.Info(ctx).Msg("closing http server")
	if err := srv.shutdown(); err != nil {
		logging.Error(ctx).Err(err).Msg("error when closing http server")
	}

	srv = nil
}
