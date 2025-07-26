package test

import (
	"context"
	"fmt"
	"os"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	rabbitmqDockerImage    = "rabbitmq:management-alpine"
	rabbitmqAMQPPort       = "5672"
	rabbitmqManagementPort = "15672"
)

var (
	rabbitmqContainerInstance *RabbitmqContainer
)

type RabbitmqContainer struct {
	rabbitmqContainerRequest *testcontainers.ContainerRequest
	rabbitmqContainer        testcontainers.Container
	ctx                      context.Context
}

func UseRabbitmqContainer(ctx context.Context) *RabbitmqContainer {
	if rabbitmqContainerInstance == nil {
		rabbitmqContainerInstance = newRabbitmqContainer()
		rabbitmqContainerInstance.ctx = ctx
		rabbitmqContainerInstance.start()
	}
	return rabbitmqContainerInstance
}

func newRabbitmqContainer() *RabbitmqContainer {
	req := &testcontainers.ContainerRequest{
		Image:        rabbitmqDockerImage,
		ExposedPorts: []string{rabbitmqAMQPPort, rabbitmqManagementPort},
		Name:         fmt.Sprintf("colibri-project-test-rabbitmq-%s", uuid.New().String()),
		Env: map[string]string{
			"RABBITMQ_DEFAULT_USER": "test",
			"RABBITMQ_DEFAULT_PASS": "test",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(rabbitmqAMQPPort),
			wait.ForListeningPort(rabbitmqManagementPort),
			wait.ForLog("Server startup complete"),
		),
	}

	return &RabbitmqContainer{rabbitmqContainerRequest: req}
}

func (c *RabbitmqContainer) start() {
	var err error
	c.rabbitmqContainer, err = testcontainers.GenericContainer(c.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: *c.rabbitmqContainerRequest,
		Started:          true,
	})
	if err != nil {
		logging.Fatal(c.ctx).Err(err)
	}

	amqpPort, err := c.rabbitmqContainer.MappedPort(c.ctx, rabbitmqAMQPPort)
	if err != nil {
		logging.Fatal(c.ctx).Err(err)
	}

	managementPort, err := c.rabbitmqContainer.MappedPort(c.ctx, rabbitmqManagementPort)
	if err != nil {
		logging.Fatal(c.ctx).Err(err)
	}

	c.setRabbitmqEnv(amqpPort, managementPort)

	logging.Info(c.ctx).Msgf("Test RabbitMQ AMQP started at port: %s", amqpPort)
	logging.Info(c.ctx).Msgf("Test RabbitMQ Management Interface available at: http://localhost:%s", managementPort)
}

func (c *RabbitmqContainer) setRabbitmqEnv(amqpPort, managementPort nat.Port) {
	_ = os.Setenv("RABBITMQ_URL", fmt.Sprintf("amqp://test:test@localhost:%s/", amqpPort.Port()))
	logging.Info(c.ctx).Msgf("RabbitMQ URL: %s", os.Getenv("RABBITMQ_URL"))
	logging.Info(c.ctx).Msgf("RabbitMQ Management URL: http://localhost:%s", managementPort.Port())
}
