package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	invalidValue            = "XYZ"
	appNameValue            = "TEST APP NAME"
	cloudHostValue          = "http://my-cloud-host-fake.com"
	cloudRegionValue        = "test-region"
	cloudSecretValue        = "test-secret"
	cloudTokenValue         = "test-token"
	portValue               = "8081"
	cacheUriValue           = "my-cache-fake:6379"
	cachePasswordValue      = "my-cache-password"
	sqlDbNameValue          = "my-db-name"
	sqlDbHostValue          = "my-db-host"
	sqlDbPortValue          = "1234"
	sqlDbUserValue          = "my-db-user"
	sqlDbPasswordValue      = "my-db-password"
	sqlDbSslModeValue       = "disable"
	waitGroupTimeout        = 400
	defaultWaitGroupTimeout = 90
)

func TestEnvironmentProfiles(t *testing.T) {
	t.Run("Should return error when environment is not configured", func(t *testing.T) {
		assert.EqualError(t, Load(), errorEnvironmentNotConfiguredMsg)
	})

	t.Run("Should return error when environment contains a invalid value", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, invalidValue))

		err := Load()
		assert.Equal(t, invalidValue, ENVIRONMENT)
		assert.EqualError(t, err, errorEnvironmentNotConfiguredMsg)
	})

	t.Run("Should configure with production environment", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, ENVIRONMENT_PRODUCTION))

		Load()
		assert.Equal(t, ENVIRONMENT_PRODUCTION, ENVIRONMENT)
		assert.True(t, IsProductionEnvironment())
		assert.False(t, IsSandboxEnvironment())
		assert.False(t, IsTestEnvironment())
		assert.False(t, IsDevelopmentEnvironment())
		assert.True(t, IsCloudEnvironment())
		assert.False(t, IsLocalEnvironment())
	})

	t.Run("Should configure with sandbox environment", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, ENVIRONMENT_SANDBOX))

		Load()
		assert.Equal(t, ENVIRONMENT_SANDBOX, ENVIRONMENT)
		assert.False(t, IsProductionEnvironment())
		assert.True(t, IsSandboxEnvironment())
		assert.False(t, IsTestEnvironment())
		assert.False(t, IsDevelopmentEnvironment())
		assert.True(t, IsCloudEnvironment())
		assert.False(t, IsLocalEnvironment())
	})

	t.Run("Should configure with test environment", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, ENVIRONMENT_TEST))

		Load()
		assert.Equal(t, ENVIRONMENT_TEST, ENVIRONMENT)
		assert.False(t, IsProductionEnvironment())
		assert.False(t, IsSandboxEnvironment())
		assert.True(t, IsTestEnvironment())
		assert.False(t, IsDevelopmentEnvironment())
		assert.False(t, IsCloudEnvironment())
		assert.True(t, IsLocalEnvironment())
	})

	t.Run("Should configure with development environment", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, ENVIRONMENT_DEVELOPMENT))

		Load()
		assert.Equal(t, ENVIRONMENT_DEVELOPMENT, ENVIRONMENT)
		assert.False(t, IsProductionEnvironment())
		assert.False(t, IsSandboxEnvironment())
		assert.False(t, IsTestEnvironment())
		assert.True(t, IsDevelopmentEnvironment())
		assert.False(t, IsCloudEnvironment())
		assert.True(t, IsLocalEnvironment())
	})
}

func TestAppName(t *testing.T) {
	assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, ENVIRONMENT_PRODUCTION))

	t.Run("Should return error when app name is not configured", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_APP_NAME, ""))

		err := Load()
		assert.Equal(t, ENVIRONMENT_PRODUCTION, ENVIRONMENT)
		assert.EqualError(t, err, errorAppNameNotConfiguredMsg)
	})

	t.Run("Should return app name", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_APP_NAME, appNameValue))

		Load()
		assert.Equal(t, ENVIRONMENT_PRODUCTION, ENVIRONMENT)
		assert.Equal(t, APP_NAME, appNameValue)
	})
}

func TestAppType(t *testing.T) {
	assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, ENVIRONMENT_PRODUCTION))
	assert.NoError(t, os.Setenv(ENV_APP_NAME, appNameValue))

	t.Run("Should return error when enviroment is not configured", func(t *testing.T) {
		err := Load()
		assert.Equal(t, ENVIRONMENT_PRODUCTION, ENVIRONMENT)
		assert.Equal(t, appNameValue, APP_NAME)
		assert.EqualError(t, err, errorAppTypeNotConfiguredMsg)
	})

	t.Run("Should return error when app_type contains a invalid value", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_APP_TYPE, invalidValue))

		err := Load()
		assert.Equal(t, ENVIRONMENT_PRODUCTION, ENVIRONMENT)
		assert.Equal(t, appNameValue, APP_NAME)
		assert.Equal(t, invalidValue, APP_TYPE)
		assert.EqualError(t, err, errorAppTypeNotConfiguredMsg)
	})

	t.Run("Should return service app type", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_APP_TYPE, APP_TYPE_SERVICE))

		Load()
		assert.Equal(t, ENVIRONMENT_PRODUCTION, ENVIRONMENT)
		assert.Equal(t, appNameValue, APP_NAME)
		assert.Equal(t, APP_TYPE_SERVICE, APP_TYPE)
	})

	t.Run("Should return serverless app type", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_APP_TYPE, APP_TYPE_SERVERLESS))

		Load()
		assert.Equal(t, ENVIRONMENT_PRODUCTION, ENVIRONMENT)
		assert.Equal(t, appNameValue, APP_NAME)
		assert.Equal(t, APP_TYPE_SERVERLESS, APP_TYPE)
	})
}

func TestCloud(t *testing.T) {
	assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, ENVIRONMENT_PRODUCTION))
	assert.NoError(t, os.Setenv(ENV_APP_NAME, appNameValue))
	assert.NoError(t, os.Setenv(ENV_APP_TYPE, APP_TYPE_SERVERLESS))

	t.Run("Should return error when cloud is not configured", func(t *testing.T) {
		assert.EqualError(t, Load(), errorCloudNotConfiguredMsg)
	})

	t.Run("Should return error when enviroment contains a invalid value", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_CLOUD, invalidValue))

		err := Load()
		assert.Equal(t, invalidValue, CLOUD)
		assert.EqualError(t, err, errorCloudNotConfiguredMsg)
	})

	t.Run("Should configure with aws environment", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_CLOUD, CLOUD_AWS))

		Load()
		assert.Equal(t, CLOUD_AWS, CLOUD)
	})

	t.Run("Should configure with gcp environment", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_CLOUD, CLOUD_GCP))

		Load()
		assert.Equal(t, CLOUD_GCP, CLOUD)
	})

	t.Run("Should configure with firebase environment", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_CLOUD, CLOUD_FIREBASE))

		Load()
		assert.Equal(t, CLOUD_FIREBASE, CLOUD)
	})
}

func TestWaitGroupTimeout(t *testing.T) {
	loadTestEnvs(t)
	t.Run("Should return default value when enviroment is not configured", func(t *testing.T) {
		assert.Equal(t, defaultWaitGroupTimeout, WAIT_GROUP_TIMEOUT_SECONDS)
	})

	t.Run("Should return wait group timeout value", func(t *testing.T) {
		assert.NoError(t, os.Setenv("WAIT_GROUP_TIMEOUT_SECONDS", fmt.Sprintf("%v", waitGroupTimeout)))

		Load()
		assert.Equal(t, waitGroupTimeout, WAIT_GROUP_TIMEOUT_SECONDS)
	})

}

func TestServerPort(t *testing.T) {
	loadTestEnvs(t)

	t.Run("Should return default server port when environment is empty", func(t *testing.T) {
		_ = Load()
		assert.Equal(t, 8080, PORT)
	})

	t.Run("Should return error when server port is wrong value", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_PORT, invalidValue))
		assert.NotNil(t, Load())
	})

	t.Run("Should return server port when environment is not empty", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_PORT, portValue))

		Load()
		assert.Equal(t, 8081, PORT)
	})
}

func TestSqlDBMaxOpenConns(t *testing.T) {
	loadTestEnvs(t)

	t.Run("Should return default sqldb max open conns when environment is empty", func(t *testing.T) {
		Load()
		assert.Equal(t, 10, SQL_DB_MAX_OPEN_CONNS)
	})

	t.Run("Should return error when sqldb max open conns is wrong value", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_SQL_DB_MAX_OPEN_CONNS, invalidValue))
		assert.NotNil(t, Load())
	})

	t.Run("Should return sqldb max open conns when environment is not empty", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_SQL_DB_MAX_OPEN_CONNS, "20"))

		Load()
		assert.Equal(t, 20, SQL_DB_MAX_OPEN_CONNS)
	})
}

func TestSqlDBMaxIdleConns(t *testing.T) {
	loadTestEnvs(t)

	t.Run("Should return default sqldb max idle conns when environment is empty", func(t *testing.T) {
		Load()
		assert.Equal(t, 3, SQL_DB_MAX_IDLE_CONNS)
	})

	t.Run("Should return error when sqldb max idle conns is wrong value", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_SQL_DB_MAX_IDLE_CONNS, invalidValue))
		assert.NotNil(t, Load())
	})

	t.Run("Should return sqldb max idle conns when environment is not empty", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_SQL_DB_MAX_IDLE_CONNS, "10"))

		Load()
		assert.Equal(t, 10, SQL_DB_MAX_IDLE_CONNS)
	})
}

func TestSqlDBMigration(t *testing.T) {
	loadTestEnvs(t)

	t.Run("Should return default migration when environment is empty", func(t *testing.T) {
		Load()
		assert.False(t, SQL_DB_MIGRATION)
	})

	t.Run("Should return error when exec migration is wrong value", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_SQL_DB_MIGRATION, invalidValue))
		assert.NotNil(t, Load())
	})

	t.Run("Should return exec migration when environment is not empty", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_SQL_DB_MIGRATION, "true"))

		Load()
		assert.True(t, SQL_DB_MIGRATION)
	})
}

func TestCloudDisableSsl(t *testing.T) {
	loadTestEnvs(t)

	t.Run("Should return default debug when environment is empty", func(t *testing.T) {
		Load()
		assert.True(t, CLOUD_DISABLE_SSL)
	})

	t.Run("Should return error when debug is wrong value", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_CLOUD_DISABLE_SSL, invalidValue))
		assert.NotNil(t, Load())
	})

	t.Run("Should return debug when environment is not empty", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_CLOUD_DISABLE_SSL, "false"))

		Load()
		assert.False(t, CLOUD_DISABLE_SSL)
	})
}

func TestGeneralEnvs(t *testing.T) {
	loadTestEnvs(t)

	t.Run("Should load configurations with success and return nil error", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_CLOUD_HOST, cloudHostValue))
		assert.NoError(t, os.Setenv(ENV_CLOUD_REGION, cloudRegionValue))
		assert.NoError(t, os.Setenv(ENV_CLOUD_SECRET, cloudSecretValue))
		assert.NoError(t, os.Setenv(ENV_CLOUD_TOKEN, cloudTokenValue))
		assert.NoError(t, os.Setenv(ENV_CACHE_URI, cacheUriValue))
		assert.NoError(t, os.Setenv(ENV_CACHE_PASSWORD, cachePasswordValue))
		assert.NoError(t, os.Setenv(ENV_SQL_DB_NAME, sqlDbNameValue))
		assert.NoError(t, os.Setenv(ENV_SQL_DB_HOST, sqlDbHostValue))
		assert.NoError(t, os.Setenv(ENV_SQL_DB_PORT, sqlDbPortValue))
		assert.NoError(t, os.Setenv(ENV_SQL_DB_USER, sqlDbUserValue))
		assert.NoError(t, os.Setenv(ENV_SQL_DB_PASSWORD, sqlDbPasswordValue))
		assert.NoError(t, os.Setenv(ENV_SQL_DB_SSL_MODE, sqlDbSslModeValue))

		dbConnectionUri := fmt.Sprintf(SQL_DB_CONNECTION_URI_DEFAULT,
			sqlDbHostValue,
			sqlDbPortValue,
			sqlDbUserValue,
			sqlDbPasswordValue,
			sqlDbNameValue,
			appNameValue,
			sqlDbSslModeValue)

		assert.Nil(t, Load())
		assert.Equal(t, cloudHostValue, CLOUD_HOST)
		assert.Equal(t, cloudRegionValue, CLOUD_REGION)
		assert.Equal(t, cloudSecretValue, CLOUD_SECRET)
		assert.Equal(t, cloudTokenValue, CLOUD_TOKEN)
		assert.Equal(t, cacheUriValue, CACHE_URI)
		assert.Equal(t, cachePasswordValue, CACHE_PASSWORD)
		assert.Equal(t, sqlDbNameValue, SQL_DB_NAME)
		assert.Equal(t, dbConnectionUri, SQL_DB_CONNECTION_URI)
	})
}

func loadTestEnvs(t *testing.T) {
	assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, ENVIRONMENT_TEST))
	assert.NoError(t, os.Setenv(ENV_APP_NAME, appNameValue))
	assert.NoError(t, os.Setenv(ENV_APP_TYPE, APP_TYPE_SERVICE))
	assert.NoError(t, os.Setenv(ENV_CLOUD, CLOUD_GCP))
}
