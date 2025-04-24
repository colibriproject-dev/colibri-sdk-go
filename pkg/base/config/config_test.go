package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	invalid_value              = "XYZ"
	app_name_value             = "TEST APP NAME"
	app_type_value             = "service"
	new_relic_license_value    = "123456-abcdef-7890-ghi"
	cloud_value                = "aws"
	cloud_host_value           = "http://my-cloud-host-fake.com"
	cloud_region_value         = "test-region"
	cloud_secret_value         = "test-secret"
	cloud_token_value          = "test-token"
	cloud_disable_ssl_value    = "true"
	port_value                 = "8081"
	cache_uri_value            = "my-cache-fake:6379"
	cache_password_value       = "my-cache-password"
	sql_db_name_value          = "my-db-name"
	sql_db_host_value          = "my-db-host"
	sql_db_port_value          = "1234"
	sql_db_user_value          = "my-db-user"
	sql_db_password_value      = "my-db-password"
	sql_db_ssl_mode_value      = "disable"
	wait_group_timeout         = 400
	default_wait_group_timeout = 90
)

func TestEnvironmentProfiles(t *testing.T) {
	t.Run("Should return error when enviroment is not configured", func(t *testing.T) {
		assert.EqualError(t, Load(), error_enviroment_not_configured)
	})

	t.Run("Should return error when enviroment contains a invalid value", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, invalid_value))

		err := Load()
		assert.Equal(t, invalid_value, ENVIRONMENT)
		assert.EqualError(t, err, error_enviroment_not_configured)
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

	t.Run("Should configure with develpoment environment", func(t *testing.T) {
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
		assert.EqualError(t, err, error_app_name_not_configured)
	})

	t.Run("Should return app name", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_APP_NAME, app_name_value))

		Load()
		assert.Equal(t, ENVIRONMENT_PRODUCTION, ENVIRONMENT)
		assert.Equal(t, APP_NAME, app_name_value)
	})
}

func TestAppType(t *testing.T) {
	assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, ENVIRONMENT_PRODUCTION))
	assert.NoError(t, os.Setenv(ENV_APP_NAME, app_name_value))

	t.Run("Should return error when enviroment is not configured", func(t *testing.T) {
		err := Load()
		assert.Equal(t, ENVIRONMENT_PRODUCTION, ENVIRONMENT)
		assert.Equal(t, app_name_value, APP_NAME)
		assert.EqualError(t, err, error_app_type_not_configured)
	})

	t.Run("Should return error when app_type contains a invalid value", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_APP_TYPE, invalid_value))

		err := Load()
		assert.Equal(t, ENVIRONMENT_PRODUCTION, ENVIRONMENT)
		assert.Equal(t, app_name_value, APP_NAME)
		assert.Equal(t, invalid_value, APP_TYPE)
		assert.EqualError(t, err, error_app_type_not_configured)
	})

	t.Run("Should return service app type", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_APP_TYPE, APP_TYPE_SERVICE))

		Load()
		assert.Equal(t, ENVIRONMENT_PRODUCTION, ENVIRONMENT)
		assert.Equal(t, app_name_value, APP_NAME)
		assert.Equal(t, APP_TYPE_SERVICE, APP_TYPE)
	})

	t.Run("Should return serverless app type", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_APP_TYPE, APP_TYPE_SERVERLESS))

		Load()
		assert.Equal(t, ENVIRONMENT_PRODUCTION, ENVIRONMENT)
		assert.Equal(t, app_name_value, APP_NAME)
		assert.Equal(t, APP_TYPE_SERVERLESS, APP_TYPE)
	})
}

func TestCloud(t *testing.T) {
	assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, ENVIRONMENT_PRODUCTION))
	assert.NoError(t, os.Setenv(ENV_APP_NAME, app_name_value))
	assert.NoError(t, os.Setenv(ENV_APP_TYPE, APP_TYPE_SERVERLESS))

	t.Run("Should return error when cloud is not configured", func(t *testing.T) {
		assert.EqualError(t, Load(), error_cloud_not_configured)
	})

	t.Run("Should return error when enviroment contains a invalid value", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_CLOUD, invalid_value))

		err := Load()
		assert.Equal(t, invalid_value, CLOUD)
		assert.EqualError(t, err, error_cloud_not_configured)
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
		assert.Equal(t, default_wait_group_timeout, WAIT_GROUP_TIMEOUT_SECONDS)
	})

	t.Run("Should return wait group timeout value", func(t *testing.T) {
		assert.NoError(t, os.Setenv("WAIT_GROUP_TIMEOUT_SECONDS", fmt.Sprintf("%v", wait_group_timeout)))

		Load()
		assert.Equal(t, wait_group_timeout, WAIT_GROUP_TIMEOUT_SECONDS)
	})

}

func TestNewRelicKey(t *testing.T) {
	assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, ENVIRONMENT_PRODUCTION))
	assert.NoError(t, os.Setenv(ENV_APP_NAME, app_name_value))
	assert.NoError(t, os.Setenv(ENV_APP_TYPE, APP_TYPE_SERVERLESS))
	assert.NoError(t, os.Setenv(ENV_CLOUD, CLOUD_FIREBASE))

	t.Run("Should return new relic key is configured in production environment", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_NEW_RELIC_LICENSE, new_relic_license_value))

		_ = Load()
		assert.Equal(t, new_relic_license_value, NEW_RELIC_LICENSE)
	})
}

func TestServerPort(t *testing.T) {
	loadTestEnvs(t)

	t.Run("Should return default server port when environment is empty", func(t *testing.T) {
		_ = Load()
		assert.Equal(t, 8080, PORT)
	})

	t.Run("Should return error when server port is wrong value", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_PORT, invalid_value))
		assert.NotNil(t, Load())
	})

	t.Run("Should return server port when environment is not empty", func(t *testing.T) {
		assert.NoError(t, os.Setenv(ENV_PORT, port_value))

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
		assert.NoError(t, os.Setenv(ENV_SQL_DB_MAX_OPEN_CONNS, invalid_value))
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
		assert.NoError(t, os.Setenv(ENV_SQL_DB_MAX_IDLE_CONNS, invalid_value))
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
		assert.NoError(t, os.Setenv(ENV_SQL_DB_MIGRATION, invalid_value))
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
		assert.NoError(t, os.Setenv(ENV_CLOUD_DISABLE_SSL, invalid_value))
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
		assert.NoError(t, os.Setenv(ENV_CLOUD_HOST, cloud_host_value))
		assert.NoError(t, os.Setenv(ENV_CLOUD_REGION, cloud_region_value))
		assert.NoError(t, os.Setenv(ENV_CLOUD_SECRET, cloud_secret_value))
		assert.NoError(t, os.Setenv(ENV_CLOUD_TOKEN, cloud_token_value))
		assert.NoError(t, os.Setenv(ENV_CACHE_URI, cache_uri_value))
		assert.NoError(t, os.Setenv(ENV_CACHE_PASSWORD, cache_password_value))
		assert.NoError(t, os.Setenv(ENV_SQL_DB_NAME, sql_db_name_value))
		assert.NoError(t, os.Setenv(ENV_SQL_DB_HOST, sql_db_host_value))
		assert.NoError(t, os.Setenv(ENV_SQL_DB_PORT, sql_db_port_value))
		assert.NoError(t, os.Setenv(ENV_SQL_DB_USER, sql_db_user_value))
		assert.NoError(t, os.Setenv(ENV_SQL_DB_PASSWORD, sql_db_password_value))
		assert.NoError(t, os.Setenv(ENV_SQL_DB_SSL_MODE, sql_db_ssl_mode_value))

		dbConnectionUri := fmt.Sprintf(SQL_DB_CONNECTION_URI_DEFAULT,
			sql_db_host_value,
			sql_db_port_value,
			sql_db_user_value,
			sql_db_password_value,
			sql_db_name_value,
			app_name_value,
			sql_db_ssl_mode_value)

		assert.Nil(t, Load())
		assert.Equal(t, cloud_host_value, CLOUD_HOST)
		assert.Equal(t, cloud_region_value, CLOUD_REGION)
		assert.Equal(t, cloud_secret_value, CLOUD_SECRET)
		assert.Equal(t, cloud_token_value, CLOUD_TOKEN)
		assert.Equal(t, cache_uri_value, CACHE_URI)
		assert.Equal(t, cache_password_value, CACHE_PASSWORD)
		assert.Equal(t, sql_db_name_value, SQL_DB_NAME)
		assert.Equal(t, dbConnectionUri, SQL_DB_CONNECTION_URI)
	})
}

func loadTestEnvs(t *testing.T) {
	assert.NoError(t, os.Setenv(ENV_ENVIRONMENT, ENVIRONMENT_TEST))
	assert.NoError(t, os.Setenv(ENV_APP_NAME, app_name_value))
	assert.NoError(t, os.Setenv(ENV_APP_TYPE, APP_TYPE_SERVICE))
	assert.NoError(t, os.Setenv(ENV_CLOUD, CLOUD_GCP))
}
