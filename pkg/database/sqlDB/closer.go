package sqlDB

import (
	"context"
	"io"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

// closer closes the provided io.Closer interface and logs an error if closing fails.
//
// o: the io.Closer interface to be closed
// Error: returns any error encountered during closing.
func closer(o io.Closer) {
	if err := o.Close(); err != nil {
		logging.Error(context.Background()).Err(err).Msg("could not close statement")
	}
}
