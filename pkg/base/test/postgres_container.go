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
	_ "github.com/newrelic/go-agent/v3/integrations/nrpq"
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
	ctx                context.Context
}

// UsePostgresContainer initialize postgres container for integration tests.
func UsePostgresContainer(ctx context.Context) *PostgresContainer {
	if postgresContainerInstance == nil {
		postgresContainerInstance = newPostgresContainer()
		postgresContainerInstance.ctx = ctx
		postgresContainerInstance.start()
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

func (c *PostgresContainer) start() {
	var err error
	c.pgContainer, err = testcontainers.GenericContainer(c.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: *c.pgContainerRequest,
		Started:          true,
	})
	if err != nil {
		logging.Fatal(c.ctx).Err(err)
	}

	testDbPort, err := c.pgContainer.MappedPort(c.ctx, testPostgresSvcPort)
	if err != nil {
		logging.Fatal(c.ctx).Err(err)
	}

	c.setDatabaseEnv(testDbPort)
	databaseURL := fmt.Sprintf(config.SQL_DB_CONNECTION_URI_DEFAULT,
		os.Getenv(config.ENV_SQL_DB_HOST),
		testDbPort.Port(),
		os.Getenv(config.ENV_SQL_DB_USER),
		os.Getenv(config.ENV_SQL_DB_PASSWORD),
		os.Getenv(config.ENV_SQL_DB_NAME),
		"test-app",
		os.Getenv(config.ENV_SQL_DB_SSL_MODE))
	if c.pgDB, err = sql.Open("postgres", databaseURL); err != nil {
		logging.Fatal(c.ctx).Err(err)
	}

	logging.Info(c.ctx).Msgf("Test postgres started at port: %s", testDbPort)
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

func (c *PostgresContainer) loadScript(basePath string, fileName string) (string, error) {
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

func (c *PostgresContainer) setDatabaseEnv(testDbPort nat.Port) {
	c.setEnv(config.ENV_SQL_DB_HOST, testDbHost)
	c.setEnv(config.ENV_SQL_DB_PORT, testDbPort.Port())
	c.setEnv(config.ENV_SQL_DB_NAME, testDbName)
	c.setEnv(config.ENV_SQL_DB_USER, testDbUser)
	c.setEnv(config.ENV_SQL_DB_PASSWORD, testDbPassword)
	c.setEnv(config.ENV_SQL_DB_SSL_MODE, "disable")
	c.setEnv(config.ENV_SQL_DB_MIGRATION, "true")
}

func (c *PostgresContainer) setEnv(env string, value string) {
	if err := os.Setenv(env, value); err != nil {
		logging.Fatal(c.ctx).Msgf("could not set env[%s] value[%s]: %v", env, value, err)
	}
}
