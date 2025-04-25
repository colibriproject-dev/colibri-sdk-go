package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitializeBaseTest(t *testing.T) {
	t.Run("should initialize base test configuration when InitializeBaseTest is called", func(t *testing.T) {
		// Arrange
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)

		// Act
		InitializeBaseTest()

		// Assert
		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "aws", os.Getenv("CLOUD"))
	})
}

func TestInitializeCacheDBTest(t *testing.T) {
	t.Run("should initialize cache database test when InitializeCacheDBTest is called", func(t *testing.T) {
		// Arrange
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)

		// Act
		InitializeCacheDBTest()

		// Assert
		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "aws", os.Getenv("CLOUD"))
	})
}

func TestInitializeSqlDBTest(t *testing.T) {
	t.Run("should initialize SQL database test when InitializeSqlDBTest is called", func(t *testing.T) {
		// Arrange
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)

		// Act
		InitializeSqlDBTest()

		// Assert
		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "aws", os.Getenv("CLOUD"))
	})
}

func TestInitializeTestLocalstack(t *testing.T) {
	t.Run("should initialize localstack test with default path when no path is provided", func(t *testing.T) {
		// Arrange
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)

		// Act
		InitializeTestLocalstack()

		// Assert
		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "aws", os.Getenv("CLOUD"))
	})

	t.Run("should initialize localstack test with custom path when path is provided", func(t *testing.T) {
		// Arrange
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)
		customPath := "/custom/path"

		// Act
		InitializeTestLocalstack(customPath)

		// Assert
		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "aws", os.Getenv("CLOUD"))
	})
}

func TestInitializeGcpEmulator(t *testing.T) {
	t.Run("should initialize GCP emulator test with default path when no path is provided", func(t *testing.T) {
		// Arrange
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)

		// Act
		InitializeGcpEmulator()

		// Assert
		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "gcp", os.Getenv("CLOUD"))
	})

	t.Run("should initialize GCP emulator test with custom path when path is provided", func(t *testing.T) {
		// Arrange
		originalEnv := os.Getenv("ENVIRONMENT")
		defer os.Setenv("ENVIRONMENT", originalEnv)
		customPath := "/custom/path"

		// Act
		InitializeGcpEmulator(customPath)

		// Assert
		assert.Equal(t, "test", os.Getenv("ENVIRONMENT"))
		assert.Equal(t, "colibri-project-test", os.Getenv("APP_NAME"))
		assert.Equal(t, "service", os.Getenv("APP_TYPE"))
		assert.Equal(t, "gcp", os.Getenv("CLOUD"))
	})
}
