package test

import (
	"context"
	"fmt"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	wiremockDockerImage = "wiremock/wiremock:3.3.1-alpine"
	wiremockSvcPort     = "8080"
)

var wiremockContainerInstance *WiremockContainer

type WiremockContainer struct {
	wContainerRequest *testcontainers.ContainerRequest
	wContainer        testcontainers.Container
	configPath        string
	instancePort      int
	ctx               context.Context
}

func UseWiremockContainer(ctx context.Context, configPath string) *WiremockContainer {
	if wiremockContainerInstance == nil {
		wiremockContainerInstance = newWiremockContainer(configPath)
		wiremockContainerInstance.ctx = ctx
		wiremockContainerInstance.start()
	}
	return wiremockContainerInstance
}

func newWiremockContainer(configPath string) *WiremockContainer {
	req := testcontainers.ContainerRequest{
		Image:         wiremockDockerImage,
		ImagePlatform: "linux/amd64",
		Name:          fmt.Sprintf("colibri-project-test-wiremock-%s", uuid.New().String()),
		ExposedPorts:  []string{wiremockSvcPort},
		Env:           map[string]string{},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: configPath,
				Target: "/home/wiremock",
			})
		},
		Cmd:        []string{"--local-response-templating"},
		WaitingFor: wait.ForListeningPort(wiremockSvcPort),
	}

	return &WiremockContainer{wContainerRequest: &req, configPath: configPath}
}

func (c *WiremockContainer) start() {
	var err error
	c.wContainer, err = testcontainers.GenericContainer(c.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: *c.wContainerRequest,
		Started:          true,
	})
	if err != nil {
		logging.Fatal(c.ctx).Err(err)
	}

	runningPort, _ := c.wContainer.MappedPort(c.ctx, wiremockSvcPort)
	c.instancePort = runningPort.Int()

	logging.Info(c.ctx).Msgf("Test wiremock started at port: %s", runningPort.Port())
}

func (c *WiremockContainer) Port() int {
	return c.instancePort
}
