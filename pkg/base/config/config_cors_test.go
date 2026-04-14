package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetCORSDefaults() {
	CORS_ALLOW_ORIGINS = "*"
	CORS_ALLOW_METHODS = "OPTIONS, GET, POST, PUT, PATCH, DELETE"
	CORS_ALLOW_HEADERS = "Origin, Content-Type, Authorization, X-User-Id, X-Tenant-Id"
	CORS_EXPOSE_HEADERS = ""
	CORS_ALLOW_CREDENTIALS = false
	CORS_MAX_AGE = 0
}

func TestLoad_CORSConfigurations(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		assertCORS  func(t *testing.T)
	}{
		{
			name: "should use default CORS values when env vars not set",
			envVars: map[string]string{
				ENV_ENVIRONMENT:            ENVIRONMENT_TEST,
				ENV_APP_NAME:               "test-app",
				ENV_APP_TYPE:               APP_TYPE_CLI,
				ENV_CLOUD:                  CLOUD_NONE,
				ENV_CORS_ALLOW_ORIGINS:     "",
				ENV_CORS_ALLOW_METHODS:     "",
				ENV_CORS_ALLOW_HEADERS:     "",
				ENV_CORS_EXPOSE_HEADERS:    "",
				ENV_CORS_ALLOW_CREDENTIALS: "",
				ENV_CORS_MAX_AGE:           "",
			},
			expectError: false,
			assertCORS: func(t *testing.T) {
				assert.Equal(t, "*", CORS_ALLOW_ORIGINS)
				assert.Equal(t, "OPTIONS, GET, POST, PUT, PATCH, DELETE", CORS_ALLOW_METHODS)
				assert.Equal(t, "Origin, Content-Type, Authorization, X-User-Id, X-Tenant-Id", CORS_ALLOW_HEADERS)
				assert.Equal(t, "", CORS_EXPOSE_HEADERS)
				assert.False(t, CORS_ALLOW_CREDENTIALS)
				assert.Equal(t, 0, CORS_MAX_AGE)
			},
		},
		{
			name: "should set CORS_ALLOW_ORIGINS from env var",
			envVars: map[string]string{
				ENV_ENVIRONMENT:            ENVIRONMENT_TEST,
				ENV_APP_NAME:               "test-app",
				ENV_APP_TYPE:               APP_TYPE_CLI,
				ENV_CLOUD:                  CLOUD_NONE,
				ENV_CORS_ALLOW_ORIGINS:     "https://example.com,https://another.com",
				ENV_CORS_ALLOW_METHODS:     "",
				ENV_CORS_ALLOW_HEADERS:     "",
				ENV_CORS_EXPOSE_HEADERS:    "",
				ENV_CORS_ALLOW_CREDENTIALS: "",
				ENV_CORS_MAX_AGE:           "",
			},
			expectError: false,
			assertCORS: func(t *testing.T) {
				assert.Equal(t, "https://example.com,https://another.com", CORS_ALLOW_ORIGINS)
			},
		},
		{
			name: "should set CORS_ALLOW_METHODS from env var",
			envVars: map[string]string{
				ENV_ENVIRONMENT:            ENVIRONMENT_TEST,
				ENV_APP_NAME:               "test-app",
				ENV_APP_TYPE:               APP_TYPE_CLI,
				ENV_CLOUD:                  CLOUD_NONE,
				ENV_CORS_ALLOW_ORIGINS:     "",
				ENV_CORS_ALLOW_METHODS:     "GET, POST",
				ENV_CORS_ALLOW_HEADERS:     "",
				ENV_CORS_EXPOSE_HEADERS:    "",
				ENV_CORS_ALLOW_CREDENTIALS: "",
				ENV_CORS_MAX_AGE:           "",
			},
			expectError: false,
			assertCORS: func(t *testing.T) {
				assert.Equal(t, "GET, POST", CORS_ALLOW_METHODS)
			},
		},
		{
			name: "should set CORS_ALLOW_HEADERS from env var",
			envVars: map[string]string{
				ENV_ENVIRONMENT:            ENVIRONMENT_TEST,
				ENV_APP_NAME:               "test-app",
				ENV_APP_TYPE:               APP_TYPE_CLI,
				ENV_CLOUD:                  CLOUD_NONE,
				ENV_CORS_ALLOW_ORIGINS:     "",
				ENV_CORS_ALLOW_METHODS:     "",
				ENV_CORS_ALLOW_HEADERS:     "Authorization, X-Custom-Header",
				ENV_CORS_EXPOSE_HEADERS:    "",
				ENV_CORS_ALLOW_CREDENTIALS: "",
				ENV_CORS_MAX_AGE:           "",
			},
			expectError: false,
			assertCORS: func(t *testing.T) {
				assert.Equal(t, "Authorization, X-Custom-Header", CORS_ALLOW_HEADERS)
			},
		},
		{
			name: "should set CORS_EXPOSE_HEADERS from env var",
			envVars: map[string]string{
				ENV_ENVIRONMENT:            ENVIRONMENT_TEST,
				ENV_APP_NAME:               "test-app",
				ENV_APP_TYPE:               APP_TYPE_CLI,
				ENV_CLOUD:                  CLOUD_NONE,
				ENV_CORS_ALLOW_ORIGINS:     "",
				ENV_CORS_ALLOW_METHODS:     "",
				ENV_CORS_ALLOW_HEADERS:     "",
				ENV_CORS_EXPOSE_HEADERS:    "X-Custom-Header, X-Request-Id",
				ENV_CORS_ALLOW_CREDENTIALS: "",
				ENV_CORS_MAX_AGE:           "",
			},
			expectError: false,
			assertCORS: func(t *testing.T) {
				assert.Equal(t, "X-Custom-Header, X-Request-Id", CORS_EXPOSE_HEADERS)
			},
		},
		{
			name: "should set CORS_ALLOW_CREDENTIALS to true",
			envVars: map[string]string{
				ENV_ENVIRONMENT:            ENVIRONMENT_TEST,
				ENV_APP_NAME:               "test-app",
				ENV_APP_TYPE:               APP_TYPE_CLI,
				ENV_CLOUD:                  CLOUD_NONE,
				ENV_CORS_ALLOW_ORIGINS:     "",
				ENV_CORS_ALLOW_METHODS:     "",
				ENV_CORS_ALLOW_HEADERS:     "",
				ENV_CORS_EXPOSE_HEADERS:    "",
				ENV_CORS_ALLOW_CREDENTIALS: "true",
				ENV_CORS_MAX_AGE:           "",
			},
			expectError: false,
			assertCORS: func(t *testing.T) {
				assert.True(t, CORS_ALLOW_CREDENTIALS)
			},
		},
		{
			name: "should set CORS_ALLOW_CREDENTIALS to false",
			envVars: map[string]string{
				ENV_ENVIRONMENT:            ENVIRONMENT_TEST,
				ENV_APP_NAME:               "test-app",
				ENV_APP_TYPE:               APP_TYPE_CLI,
				ENV_CLOUD:                  CLOUD_NONE,
				ENV_CORS_ALLOW_ORIGINS:     "",
				ENV_CORS_ALLOW_METHODS:     "",
				ENV_CORS_ALLOW_HEADERS:     "",
				ENV_CORS_EXPOSE_HEADERS:    "",
				ENV_CORS_ALLOW_CREDENTIALS: "false",
				ENV_CORS_MAX_AGE:           "",
			},
			expectError: false,
			assertCORS: func(t *testing.T) {
				assert.False(t, CORS_ALLOW_CREDENTIALS)
			},
		},
		{
			name: "should return error for invalid CORS_ALLOW_CREDENTIALS value",
			envVars: map[string]string{
				ENV_ENVIRONMENT:            ENVIRONMENT_TEST,
				ENV_APP_NAME:               "test-app",
				ENV_APP_TYPE:               APP_TYPE_CLI,
				ENV_CLOUD:                  CLOUD_NONE,
				ENV_CORS_ALLOW_ORIGINS:     "",
				ENV_CORS_ALLOW_METHODS:     "",
				ENV_CORS_ALLOW_HEADERS:     "",
				ENV_CORS_EXPOSE_HEADERS:    "",
				ENV_CORS_ALLOW_CREDENTIALS: "invalid",
				ENV_CORS_MAX_AGE:           "",
			},
			expectError: true,
			assertCORS:  nil,
		},
		{
			name: "should set CORS_MAX_AGE from env var",
			envVars: map[string]string{
				ENV_ENVIRONMENT:            ENVIRONMENT_TEST,
				ENV_APP_NAME:               "test-app",
				ENV_APP_TYPE:               APP_TYPE_CLI,
				ENV_CLOUD:                  CLOUD_NONE,
				ENV_CORS_ALLOW_ORIGINS:     "",
				ENV_CORS_ALLOW_METHODS:     "",
				ENV_CORS_ALLOW_HEADERS:     "",
				ENV_CORS_EXPOSE_HEADERS:    "",
				ENV_CORS_ALLOW_CREDENTIALS: "",
				ENV_CORS_MAX_AGE:           "3600",
			},
			expectError: false,
			assertCORS: func(t *testing.T) {
				assert.Equal(t, 3600, CORS_MAX_AGE)
			},
		},
		{
			name: "should return error for invalid CORS_MAX_AGE value",
			envVars: map[string]string{
				ENV_ENVIRONMENT:            ENVIRONMENT_TEST,
				ENV_APP_NAME:               "test-app",
				ENV_APP_TYPE:               APP_TYPE_CLI,
				ENV_CLOUD:                  CLOUD_NONE,
				ENV_CORS_ALLOW_ORIGINS:     "",
				ENV_CORS_ALLOW_METHODS:     "",
				ENV_CORS_ALLOW_HEADERS:     "",
				ENV_CORS_EXPOSE_HEADERS:    "",
				ENV_CORS_ALLOW_CREDENTIALS: "",
				ENV_CORS_MAX_AGE:           "invalid",
			},
			expectError: true,
			assertCORS:  nil,
		},
		{
			name: "should set all CORS values at once",
			envVars: map[string]string{
				ENV_ENVIRONMENT:            ENVIRONMENT_TEST,
				ENV_APP_NAME:               "test-app",
				ENV_APP_TYPE:               APP_TYPE_CLI,
				ENV_CLOUD:                  CLOUD_NONE,
				ENV_CORS_ALLOW_ORIGINS:     "https://app.example.com",
				ENV_CORS_ALLOW_METHODS:     "GET, POST, PUT, DELETE",
				ENV_CORS_ALLOW_HEADERS:     "Authorization, Content-Type",
				ENV_CORS_EXPOSE_HEADERS:    "X-Request-Id",
				ENV_CORS_ALLOW_CREDENTIALS: "true",
				ENV_CORS_MAX_AGE:           "7200",
			},
			expectError: false,
			assertCORS: func(t *testing.T) {
				assert.Equal(t, "https://app.example.com", CORS_ALLOW_ORIGINS)
				assert.Equal(t, "GET, POST, PUT, DELETE", CORS_ALLOW_METHODS)
				assert.Equal(t, "Authorization, Content-Type", CORS_ALLOW_HEADERS)
				assert.Equal(t, "X-Request-Id", CORS_EXPOSE_HEADERS)
				assert.True(t, CORS_ALLOW_CREDENTIALS)
				assert.Equal(t, 7200, CORS_MAX_AGE)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetCORSDefaults()
			// Clear env
			os.Clearenv()
			t.Cleanup(func() {
				os.Clearenv()
				resetCORSDefaults()
			})

			// Set test env vars
			for key, value := range tt.envVars {
				if value != "" {
					err := os.Setenv(key, value)
					require.NoError(t, err)
				}
			}

			err := Load()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.assertCORS != nil {
					tt.assertCORS(t)
				}
			}
		})
	}
}
