package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitializeBaseTest(t *testing.T) {
	t.Run("should initialize base test configuration when InitializeBaseTest is called", func(t *testing.T) {
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)

		InitializeBaseTest()

		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "aws", os.Getenv("CLOUD"))
	})
}

func TestInitializeCacheDBTest(t *testing.T) {
	t.Run("should initialize cache database test when InitializeCacheDBTest is called", func(t *testing.T) {
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)

		InitializeCacheDBTest()

		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "aws", os.Getenv("CLOUD"))
	})
}

func TestInitializeSqlDBTest(t *testing.T) {
	t.Run("should initialize SQL database test when InitializeSqlDBTest is called", func(t *testing.T) {
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)

		InitializeSqlDBTest()

		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "aws", os.Getenv("CLOUD"))
	})
}

func TestInitializeTestLocalstack(t *testing.T) {
	t.Run("should initialize localstack test with default path when no path is provided", func(t *testing.T) {
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)

		InitializeTestLocalstack()

		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "aws", os.Getenv("CLOUD"))
	})

	t.Run("should initialize localstack test with custom path when path is provided", func(t *testing.T) {
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)
		customPath := "/custom/path"

		InitializeTestLocalstack(customPath)

		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "aws", os.Getenv("CLOUD"))
	})
}

func TestInitializeGcpEmulator(t *testing.T) {
	t.Run("should initialize GCP emulator test with default path when no path is provided", func(t *testing.T) {
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)

		InitializeGcpEmulator()

		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "gcp", os.Getenv("CLOUD"))
	})

	t.Run("should initialize GCP emulator test with custom path when path is provided", func(t *testing.T) {
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)
		customPath := "/custom/path"

		InitializeGcpEmulator(customPath)

		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "gcp", os.Getenv("CLOUD"))
	})
}
