package test

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/cloud"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/validator"
	"github.com/google/uuid"
)

type key string

const (
	localstackID  key = "localstack-id"
	gcpEmulatorID key = "gcpEmulator-id"
	rabbitmqID    key = "rabbitmq-id"

	DEVELOPMENT_ENVIRONMENT_PATH  string = "../../../development-environment"
	DATABASE_ENVIRONMENT_PATH     string = DEVELOPMENT_ENVIRONMENT_PATH + "/database/"
	REST_ENVIRONMENT_PATH         string = DEVELOPMENT_ENVIRONMENT_PATH + "/rest/"
	LOCALSTACK_ENVIRONMENT_PATH   string = DEVELOPMENT_ENVIRONMENT_PATH + "/localstack/"
	GCP_EMULATOR_ENVIRONMENT_PATH string = DEVELOPMENT_ENVIRONMENT_PATH + "/gcp-emulator/"
	WIREMOCK_ENVIRONMENT_PATH     string = DEVELOPMENT_ENVIRONMENT_PATH + "/wiremock/"
)

var m sync.Mutex

func InitializeBaseTest() {
	loadConfig()
}

func InitializeCacheDBTest() {
	UseRedisContainer(context.Background())
	loadConfig()
}

func InitializeSqlDBTest() {
	UsePostgresContainer(context.Background())
	loadConfig()
}

func InitializeTestLocalstack(path ...string) {
	m.Lock()
	ctx := context.WithValue(context.Background(), localstackID, uuid.New().String())
	_ = UseLocalstackContainer(ctx, getLocalstackBasePath(path...))
	loadConfig()
	cloud.Initialize()
	m.Unlock()
}

func getLocalstackBasePath(path ...string) string {
	if len(path) == 0 {
		return MountAbsolutPath(LOCALSTACK_ENVIRONMENT_PATH)
	}
	return path[0]
}

func InitializeGcpEmulator(path ...string) {
	m.Lock()
	ctx := context.WithValue(context.Background(), gcpEmulatorID, uuid.New().String())
	_ = UseGcpEmulatorContainer(ctx, getGcpEmulatorBasePath(path...))
	loadConfig()
	_ = os.Setenv(config.ENV_CLOUD, config.CLOUD_GCP)
	config.CLOUD = config.CLOUD_GCP
	cloud.Initialize()
	m.Unlock()
}

func getGcpEmulatorBasePath(path ...string) string {
	if len(path) == 0 {
		return MountAbsolutPath(GCP_EMULATOR_ENVIRONMENT_PATH)
	}
	return path[0]
}

func InitializeRabbitmq() {
	m.Lock()
	ctx := context.WithValue(context.Background(), rabbitmqID, uuid.New().String())
	_ = UseRabbitmqContainer(ctx)
	loadConfig()

	_ = os.Setenv(config.ENV_COLIBRI_MESSAGING, config.MESSAGING_RABBITMQ)
	config.COLIBRI_MESSAGING = config.MESSAGING_RABBITMQ
	_ = os.Setenv(config.ENV_CLOUD, config.CLOUD_NONE)
	cloud.Initialize()
	m.Unlock()
}

func loadConfig() {
	_ = os.Setenv(config.ENV_ENVIRONMENT, config.ENVIRONMENT_TEST)
	_ = os.Setenv(config.ENV_APP_NAME, "colibri-project-test")
	_ = os.Setenv(config.ENV_APP_TYPE, config.APP_TYPE_SERVICE)
	_ = os.Setenv(config.ENV_CLOUD, config.CLOUD_AWS)

	if err := config.Load(); err != nil {
		log.Fatalf("could not start configuration: %v", err)
		return
	}
	validator.Initialize()
	observer.Initialize()
	monitoring.Initialize()
}
