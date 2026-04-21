package test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/google/uuid"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	postgresDockerImage = "postgres:16-alpine"
	testDbHost          = "localhost"
	testDbName          = "test_db"
	testDbUser          = "test_user"
	testDbPassword      = "test_password"
	testPostgresSvcPort = "5432"
)

var postgresContainerInstance *PostgresContainer

type PostgresContainer struct {
	pgContainerRequest *testcontainers.ContainerRequest
	pgContainer        testcontainers.Container
	pgDB               *sql.DB
}

// UsePostgresContainer initialize postgres container for integration tests.
func UsePostgresContainer(ctx context.Context) *PostgresContainer {
	if postgresContainerInstance == nil {
		postgresContainerInstance = newPostgresContainer()
		postgresContainerInstance.start(ctx)
	}
	return postgresContainerInstance
}

func newPostgresContainer() *PostgresContainer {
	req := &testcontainers.ContainerRequest{
		Image:        postgresDockerImage,
		ExposedPorts: []string{testPostgresSvcPort},
		Name:         fmt.Sprintf("colibri-project-test-postgres-%s", uuid.New().String()),
		Env: map[string]string{
			"POSTGRES_DB":       testDbName,
			"POSTGRES_USER":     testDbUser,
			"POSTGRES_PASSWORD": testDbPassword,
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(testPostgresSvcPort),
			wait.ForSQL(testPostgresSvcPort, "postgres", func(host string, port nat.Port) string {
				return fmt.Sprintf(
					"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
					host,
					port.Port(),
					testDbUser,
					testDbPassword,
					testDbName,
				)
			}),
		),
	}

	return &PostgresContainer{pgContainerRequest: req}
}

func (c *PostgresContainer) start(ctx context.Context) {
	var err error
	c.pgContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: *c.pgContainerRequest,
		Started:          true,
	})
	if err != nil {
		logging.Fatal(ctx).Err(err)
	}

	testDbPort, err := c.pgContainer.MappedPort(ctx, testPostgresSvcPort)
	if err != nil {
		logging.Fatal(ctx).Err(err)
	}

	c.setDatabaseEnv(ctx, testDbPort)
	databaseURL := fmt.Sprintf(config.SQL_DB_CONNECTION_URI_DEFAULT,
		os.Getenv(config.ENV_SQL_DB_HOST),
		testDbPort.Port(),
		os.Getenv(config.ENV_SQL_DB_USER),
		os.Getenv(config.ENV_SQL_DB_PASSWORD),
		os.Getenv(config.ENV_SQL_DB_NAME),
		"test-app",
		os.Getenv(config.ENV_SQL_DB_SSL_MODE))
	if c.pgDB, err = sql.Open("postgres", databaseURL); err != nil {
		logging.Fatal(ctx).Err(err)
	}

	logging.Info(ctx).Msgf("Test postgres started at port: %s", testDbPort)
}

func (c *PostgresContainer) Dataset(basePath string, scripts ...string) error {
	for _, s := range scripts {
		script, err := c.loadScript(basePath, s)
		if err != nil {
			return err
		}

		if err = c.execScript(script); err != nil {
			return err
		}
	}

	return nil
}

func (c *PostgresContainer) loadScript(basePath, fileName string) (string, error) {
	if !strings.HasSuffix(basePath, "/") {
		basePath += "/"
	}

	filePath := fmt.Sprintf("%s%s", basePath, fileName)
	script, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("could not read script file: %v", err)
	}

	return string(script), nil
}

func (c *PostgresContainer) execScript(script string) error {
	if _, err := c.pgDB.Exec(script); err != nil {
		return fmt.Errorf("could not execute script: %v", err)
	}

	return nil
}

func (c *PostgresContainer) setDatabaseEnv(ctx context.Context, testDbPort nat.Port) {
	c.setEnv(ctx, config.ENV_SQL_DB_HOST, testDbHost)
	c.setEnv(ctx, config.ENV_SQL_DB_PORT, testDbPort.Port())
	c.setEnv(ctx, config.ENV_SQL_DB_NAME, testDbName)
	c.setEnv(ctx, config.ENV_SQL_DB_USER, testDbUser)
	c.setEnv(ctx, config.ENV_SQL_DB_PASSWORD, testDbPassword)
	c.setEnv(ctx, config.ENV_SQL_DB_SSL_MODE, "disable")
	c.setEnv(ctx, config.ENV_SQL_DB_MIGRATION, "true")
}

func (c *PostgresContainer) setEnv(ctx context.Context, env, value string) {
	if err := os.Setenv(env, value); err != nil {
		logging.Fatal(ctx).Msgf("could not set env[%s] value[%s]: %v", env, value, err)
	}
}
