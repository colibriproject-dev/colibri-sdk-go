package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	rabbitmqDockerImage    = "rabbitmq:4-management-alpine"
	rabbitmqAMQPPort       = "5672"
	rabbitmqManagementPort = "15672"
)

var (
	rabbitmqContainerInstance *RabbitmqContainer
)

type RabbitmqContainer struct {
	rabbitmqContainerRequest *testcontainers.ContainerRequest
	rabbitmqContainer        testcontainers.Container
}

func UseRabbitmqContainer(ctx context.Context, configPath string) *RabbitmqContainer {
	if rabbitmqContainerInstance == nil {
		rabbitmqContainerInstance = newRabbitmqContainer(configPath)
		rabbitmqContainerInstance.start(ctx)
	}
	return rabbitmqContainerInstance
}

func newRabbitmqContainer(configPath string) *RabbitmqContainer {
	// Get the absolute path to the definitions.json file
	rabbitmqConfPath := filepath.Join(configPath, "rabbitmq.conf")
	definitionsPath := filepath.Join(configPath, "definitions.json")

	req := &testcontainers.ContainerRequest{
		Image:        rabbitmqDockerImage,
		ExposedPorts: []string{rabbitmqAMQPPort, rabbitmqManagementPort},
		Name:         fmt.Sprintf("colibri-project-test-rabbitmq-%s", uuid.New().String()),
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: rabbitmqConfPath,
				Target: "/etc/rabbitmq/rabbitmq.conf",
			})
			hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: definitionsPath,
				Target: "/etc/rabbitmq/definitions.json",
			})
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(rabbitmqAMQPPort),
			wait.ForListeningPort(rabbitmqManagementPort),
			wait.ForLog("Server startup complete"),
		),
	}

	return &RabbitmqContainer{rabbitmqContainerRequest: req}
}

func (c *RabbitmqContainer) start(ctx context.Context) {
	var err error
	c.rabbitmqContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: *c.rabbitmqContainerRequest,
		Started:          true,
	})
	if err != nil {
		logging.Fatal(ctx).Err(err)
	}

	amqpPort, err := c.rabbitmqContainer.MappedPort(ctx, rabbitmqAMQPPort)
	if err != nil {
		logging.Fatal(ctx).Err(err)
	}

	managementPort, err := c.rabbitmqContainer.MappedPort(ctx, rabbitmqManagementPort)
	if err != nil {
		logging.Fatal(ctx).Err(err)
	}

	c.setRabbitmqEnv(ctx, amqpPort, managementPort)

	logging.Info(ctx).Msgf("Test RabbitMQ AMQP started at port: %s", amqpPort)
	logging.Info(ctx).Msgf("Test RabbitMQ Management Interface available at: http://localhost:%s", managementPort)
}

func (c *RabbitmqContainer) setRabbitmqEnv(ctx context.Context, amqpPort, managementPort nat.Port) {
	_ = os.Setenv("RABBITMQ_URL", fmt.Sprintf("amqp://test:test@localhost:%s/", amqpPort.Port()))
	logging.Info(ctx).Msgf("RabbitMQ URL: %s", os.Getenv("RABBITMQ_URL"))
	logging.Info(ctx).Msgf("RabbitMQ Management URL: http://localhost:%s", managementPort.Port())
}
