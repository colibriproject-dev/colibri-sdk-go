package logging

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/stretchr/testify/assert"
)

func TestLogging(t *testing.T) {
	var buf bytes.Buffer
	logOutput = &buf

	config.ENVIRONMENT = config.ENVIRONMENT_TEST
	ctx := context.Background()
	ctx = InjectCorrelationIDInContext(ctx, "test-correlation-id")

	t.Run("Should initialize logger with default level when LOG_LEVEL is not set", func(t *testing.T) {
		os.Unsetenv("LOG_LEVEL")

		Initialize()

		assert.NotNil(t, logger)
	})

	t.Run("Should initialize logger with specified level when LOG_LEVEL is set", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "debug")

		Initialize()

		assert.NotNil(t, logger)
	})

	t.Run("Should create logging instance with correlation ID when context has correlation ID", func(t *testing.T) {
		log := Info(ctx)

		assert.Equal(t, "test-correlation-id", log.params[CorrelationIDParam])
	})

	t.Run("Should add parameters to logging instance when AddParam is called", func(t *testing.T) {
		log := Info(ctx).AddParam("testKey", "testValue")

		assert.Equal(t, "testValue", log.params["testKey"])
	})

	t.Run("Should add error to logging instance when Err is called", func(t *testing.T) {
		testErr := errors.New("test error")

		log := Error(ctx).Err(testErr)

		assert.Equal(t, testErr, log.err)
	})

	t.Run("Should format message correctly when Msg is called", func(t *testing.T) {
		Info(ctx).AddParam("test", "value").Msg("Test message 123")

		output := buf.String()
		assert.Contains(t, output, "Test message 123")
		assert.Contains(t, output, "test=value")
	})

	t.Run("Should format message correctly when Msgf is called", func(t *testing.T) {
		Info(ctx).AddParam("test", "value").Msgf("Test %s %d", "message", 456)

		output := buf.String()
		assert.Contains(t, output, "Test message 456")
		assert.Contains(t, output, "test=value")
	})

	t.Run("Should panic when Fatal is called with Msg", func(t *testing.T) {
		assert.Panics(t, func() {
			Fatal(ctx).Msg("fatal message")
		})
	})

	t.Run("Should return correct log level when parseLevel is called", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected slog.Level
		}{
			{"error", slog.LevelError},
			{"warn", slog.LevelWarn},
			{"warning", slog.LevelWarn},
			{"debug", slog.LevelDebug},
			{"info", slog.LevelInfo},
			{"invalid", slog.LevelInfo},
		}

		for _, tc := range testCases {
			t.Run("Should return "+tc.input+" level", func(t *testing.T) {
				result := parseLevel(tc.input)

				assert.EqualValues(t, tc.expected, result)
			})
		}
	})

	t.Run("Should clean function name correctly when cleanFunctionName is called", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging.TestLogging", "logging.TestLogging"},
			{"(*Logging).AddParam", "Logging.AddParam"},
			{"simpleFunction", "simpleFunction"},
		}

		for _, tc := range testCases {
			t.Run("Should clean "+tc.input, func(t *testing.T) {
				result := cleanFunctionName(tc.input)

				assert.EqualValues(t, tc.expected, result)
			})
		}
	})
}
