package test

import (
	"context"
	"fmt"
	"os"

	"github.com/colibri-project-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	gcpEmulatorDockerImage    = "google/cloud-sdk:latest"
	gcpEmulatorPubSubSvcPort  = "8686"
	gcpEmulatorStorageSvcPort = "8080"
)

var (
	gcpEmulatorContainerInstance *gcpEmulatorContainer
)

type gcpEmulatorContainer struct {
	lsContainerRequest *testcontainers.ContainerRequest
	lsContainer        testcontainers.Container
	ctx                context.Context
}

func UseGcpEmulatorContainer(ctx context.Context, configPath string) *gcpEmulatorContainer {
	if gcpEmulatorContainerInstance == nil {
		gcpEmulatorContainerInstance = newGcpEmulatorContainer(ctx, configPath)
		gcpEmulatorContainerInstance.start()
	}
	return gcpEmulatorContainerInstance
}

func newGcpEmulatorContainer(ctx context.Context, configPath string) *gcpEmulatorContainer {
	req := &testcontainers.ContainerRequest{
		Image:        gcpEmulatorDockerImage,
		ExposedPorts: []string{gcpEmulatorPubSubSvcPort, gcpEmulatorStorageSvcPort},
		Name:         fmt.Sprintf("colibri-project-test-gcp-emulator-%s", uuid.New().String()),
		Entrypoint:   []string{"/bin/sh", "-c"},
		Cmd:          []string{"/scripts/start.sh"},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: configPath,
				Target: "/scripts",
			})
			hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: "/var/run/docker.sock",
				Target: "/var/run/docker.sock",
			})
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("GCP emulator started"),
		),
	}

	return &gcpEmulatorContainer{lsContainerRequest: req, ctx: ctx}
}

func (c *gcpEmulatorContainer) start() {
	var err error
	c.lsContainer, err = testcontainers.GenericContainer(c.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: *c.lsContainerRequest,
		Started:          true,
	})
	if err != nil {
		logging.Fatal(c.ctx).Err(err)
	}

	pubSubPort, err := c.lsContainer.MappedPort(c.ctx, gcpEmulatorPubSubSvcPort)
	if err != nil {
		logging.Fatal(c.ctx).Err(err)
	}

	storagePort, err := c.lsContainer.MappedPort(c.ctx, gcpEmulatorStorageSvcPort)
	if err != nil {
		logging.Fatal(c.ctx).Err(err)
	}

	logging.Info(c.ctx).Msgf("Test gcp emulator started. Pub/Sub at %s and Storage at %s", pubSubPort, storagePort)
	c.setEnv(pubSubPort, storagePort)
}

func (c *gcpEmulatorContainer) setEnv(pubSubPort, storagePort nat.Port) {
	os.Setenv("PUBSUB_PROJECT_ID", "test-project")
	os.Setenv("PUBSUB_EMULATOR_HOST", fmt.Sprintf("localhost:%s", pubSubPort.Port()))
	os.Setenv("STORAGE_EMULATOR_HOST", fmt.Sprintf("localhost:%s", storagePort.Port()))
}
