package test

import (
	"context"
	"fmt"
	"os"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/google/uuid"
	"github.com/moby/moby/api/types/network"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	redisDockerImage = "redis:alpine"
	testRedisSvcPort = "6379"
)

var redisContainerInstance *RedisContainer

type RedisContainer struct {
	redisContainerRequest *testcontainers.ContainerRequest
	redisContainer        testcontainers.Container
	redisClient           *redis.Client
}

func UseRedisContainer(ctx context.Context) *RedisContainer {
	if redisContainerInstance == nil {
		redisContainerInstance = newRedisContainer()
		redisContainerInstance.start(ctx)
	}
	return redisContainerInstance
}

func newRedisContainer() *RedisContainer {
	req := &testcontainers.ContainerRequest{
		Image:        redisDockerImage,
		ExposedPorts: []string{testRedisSvcPort},
		Name:         fmt.Sprintf("colibri-project-test-redis-%s", uuid.New().String()),
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(testRedisSvcPort),
		),
	}

	return &RedisContainer{redisContainerRequest: req}
}

func (c *RedisContainer) start(ctx context.Context) {
	var err error
	c.redisContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: *c.redisContainerRequest,
		Started:          true,
	})
	if err != nil {
		logging.Fatal(ctx).Err(err).Msg("could not start the test redis container")
	}

	testDbPort, err := c.redisContainer.MappedPort(ctx, testRedisSvcPort)
	if err != nil {
		logging.Fatal(ctx).Err(err).Msg("could not get the mapped port of the test redis container")
	}

	c.setRedisEnv(testDbPort)
	opts := &redis.Options{Addr: fmt.Sprintf("localhost:%s", testDbPort.Port())}
	c.redisClient = redis.NewClient(opts)

	logging.Info(ctx).Msgf("Test redis started at port: %s", testDbPort)
}

func (c *RedisContainer) Client() *redis.Client {
	return c.redisClient
}

func (c *RedisContainer) setRedisEnv(port network.Port) {
	_ = os.Setenv(config.ENV_CACHE_URI, fmt.Sprintf("localhost:%s", port.Port()))
}
