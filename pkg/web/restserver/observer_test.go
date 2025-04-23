package restserver

import (
	"testing"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/observer"
	"github.com/gofiber/fiber/v2"

	"github.com/stretchr/testify/assert"
)

func TestCloseServer(t *testing.T) {
	srv = &fiberWebServer{srv: &fiber.App{}}

	restObserver{}.Close()
	assert.Nil(t, srv)
}

func TestCloseServerWithTimeout(t *testing.T) {
	config.WAIT_GROUP_TIMEOUT_SECONDS = 1
	observer.GetWaitGroup().Add(1)
	defer observer.GetWaitGroup().Done()

	srv = &fiberWebServer{srv: &fiber.App{}}

	restObserver{}.Close()
	assert.Nil(t, srv)
}
