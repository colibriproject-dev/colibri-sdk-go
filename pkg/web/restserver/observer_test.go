package restserver

import (
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
	"github.com/gofiber/fiber/v3"

	"github.com/stretchr/testify/assert"
)

func TestCloseServer(t *testing.T) {
	logging.Initialize()

	srv = &fiberWebServer{srv: &fiber.App{}}

	restObserver{}.Close()
	assert.Nil(t, srv)
}

func TestStopServer(t *testing.T) {
	t.Run("Should return without waiting and leave the shutdown to the drain phase", func(t *testing.T) {
		logging.Initialize()

		previousTimeout := config.WAIT_GROUP_TIMEOUT_SECONDS
		config.WAIT_GROUP_TIMEOUT_SECONDS = 5
		t.Cleanup(func() { config.WAIT_GROUP_TIMEOUT_SECONDS = previousTimeout })

		srv = &fiberWebServer{srv: &fiber.App{}}
		t.Cleanup(func() { srv = nil })

		returned := make(chan struct{})
		go func() {
			defer close(returned)
			restObserver{}.Stop()
		}()

		select {
		case <-returned:
		case <-time.After(time.Second):
			t.Fatal("Stop waited for the server shutdown instead of leaving it to the drain phase")
		}

		// the shutdown Stop started is registered on the running counter, so the drain phase
		// covers it: a wait that returns without a timeout means it was both registered and
		// released
		assert.False(t, observer.WaitRunningTimeout(), "the server shutdown was not drained")
	})
}

func TestCloseServerDoesNotWaitForRunningWork(t *testing.T) {
	t.Run("Should release the server without draining, which the pipeline already did", func(t *testing.T) {
		logging.Initialize()

		previousTimeout := config.WAIT_GROUP_TIMEOUT_SECONDS
		config.WAIT_GROUP_TIMEOUT_SECONDS = 30
		t.Cleanup(func() { config.WAIT_GROUP_TIMEOUT_SECONDS = previousTimeout })

		// work that outlives the close, the way a shutdown reaching the closing phase after
		// the drain timed out leaves it
		observer.AddRunning()
		t.Cleanup(observer.DoneRunning)

		srv = &fiberWebServer{srv: &fiber.App{}}

		returned := make(chan struct{})
		go func() {
			defer close(returned)
			restObserver{}.Close()
		}()

		select {
		case <-returned:
		case <-time.After(time.Second):
			t.Fatal("Close waited for the running work, which belongs to the drain phase")
		}

		assert.Nil(t, srv)
	})
}
