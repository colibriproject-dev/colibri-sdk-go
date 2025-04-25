package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
)

type correlationID string

const (
	correlationIDParam correlationID = "correlationID"
	callerParam        string        = "caller"
	errorParam         string        = "error"
	cIDParam           string        = "correlationId"
)

var (
	logger    *slog.Logger
	logLevel  string    = ""
	logOutput io.Writer = os.Stdout
)

// Logging is a struct that contains the parameters for the logging.
type Logging struct {
	params        map[string]any
	err           error
	level         string
	correlationID string
}

func init() {
	Initialize()
}

// Initialize initializes the logging.
func Initialize() {
	logger = slog.New(createLogHandler())
}

// createLogHandler creates and returns the appropriate log handler based on the environment and log level.
// If the application is running in a development environment,
// it returns a text-based log handler. Otherwise, it returns a JSON handler.
func createLogHandler() slog.Handler {
	logLevel = os.Getenv("LOG_LEVEL")
	if !slices.Contains([]string{"debug", "info", "warn", "warning", "error"}, logLevel) {
		logLevel = "info"
	}

	opts := &slog.HandlerOptions{Level: parseLevel(logLevel)}
	if config.IsLocalEnvironment() {
		return slog.NewTextHandler(logOutput, opts)
	}

	return slog.NewJSONHandler(logOutput, opts)
}

// parseLevel converts a string representation of the log level into a corresponding slog.Level value.
// Accepted values: "error", "warn", "warning", "debug", "info".
// If an invalid value is provided, it defaults to "info".
func parseLevel(lvl string) slog.Level {
	switch strings.ToLower(lvl) {
	case "error":
		return slog.LevelError
	case "warn", "warning":
		return slog.LevelWarn
	case "debug":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

// Error returns a new Logging instance configured for ERROR level log.
func Error(ctx context.Context) *Logging {
	return startLogging(ctx, "ERROR")
}

// Fatal returns a new Logging instance configured for ERROR level log. After the log is printed, a panic is thrown.
func Fatal(ctx context.Context) *Logging {
	return startLogging(ctx, "FATAL")
}

// Info returns a new Logging instance configured for INFO level log.
func Info(ctx context.Context) *Logging {
	return startLogging(ctx, "INFO")
}

// Warn returns a new Logging instance configured for WARN level log.
func Warn(ctx context.Context) *Logging {
	return startLogging(ctx, "WARN")
}

// Debug returns a new Logging instance configured for DEBUG level log.
func Debug(ctx context.Context) *Logging {
	return startLogging(ctx, "DEBUG")
}

// AddParam adds a key-value pair to the logging parameters.
// Example:
//
//	logging.Info(ctx).AddParam("userID", 123).Msg("User logged in")
func (l *Logging) AddParam(key string, val any) *Logging {
	l.params[key] = val

	return l
}

// Err sets the error field in the Logging instance.
// When an error is added, the log will include a specific field for the error.
// error.
// Example:
//
//	err := errors.New("database connection failed")
//
//	logging.Error(ctx).Err(err).Msg("Database error")
func (l *Logging) Err(err error) *Logging {
	l.err = err
	return l
}

// Msg logs a message with the appropriate log level and parameters.
// Example:
//
//	logging.Info(ctx).AddParam("userID", 123).Msg("User logged in")
func (l *Logging) Msg(msg string) {
	params := buildParams(l)
	switch l.level {
	case "DEBUG":
		logger.Debug(msg, params...)
	case "WARN":
		logger.Warn(msg, params...)
	case "ERROR":
		logger.Error(msg, params...)
	case "FATAL":
		logger.Error(msg, params...)
		panic(msg)
	default:
		logger.Info(msg, params...)
	}
}

// Msgf logs a formatted message with the appropriate log level.
// Example:
//
//	logging.Info(ctx).Msgf("User %s logged in at %s", "John", time.Now().Format(time.RFC3339))
func (l *Logging) Msgf(msg string, args ...any) {
	l.Msg(fmt.Sprintf(msg, args...))
}

// buildParams transforms the log params received in a slice. This is necessary because of the slog, the params needs
// to be a slice, where the key is the first param, and the value the second (or the next).
func buildParams(l *Logging) []any {
	if l.err != nil {
		l.params[errorParam] = l.err.Error()
	}

	if l.correlationID != "" {
		l.params[cIDParam] = l.correlationID
	}

	vals := make([]any, 0, len(l.params)*2)
	for k, v := range l.params {
		vals = append(vals, k)
		vals = append(vals, v)
	}

	return vals
}

// startLogging initializes a new Logging instance with a given log level and correlation ID from context (if exists).
func startLogging(ctx context.Context, level string) *Logging {
	l := createLogging()
	l.level = level
	l.correlationID = getCorrelationIDFromContext(ctx)

	return l
}

// InjectCorrelationIDInContext injects the correlation ID into the provided context, that allow to log with this ID
// making easy to identify all logs related to the same execution.
func InjectCorrelationIDInContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDParam, id)
}

// getCorrelationIDFromContext retrieves the correlation ID from the context.
func getCorrelationIDFromContext(ctx context.Context) string {
	val := ctx.Value(correlationIDParam)
	if val == nil {
		return ""
	}

	cID := val.(string)
	return cID
}

// createLogging creates and initializes a new Logging instance.
func createLogging() *Logging {
	l := &Logging{}
	l.params = make(map[string]any)
	l.params[callerParam] = getCallerFunctionName()

	return l
}

// getCallerFunctionName returns the name of the function that called the logging.
func getCallerFunctionName() string {
	skipCallerNumber := 4

	pc, _, _, ok := runtime.Caller(skipCallerNumber)
	if !ok {
		return "unknown"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}

	return cleanFunctionName(fn.Name())
}

// cleanFunctionName removes unwanted characters from a function name.
func cleanFunctionName(name string) string {
	name = strings.ReplaceAll(name, "*", "")
	name = strings.ReplaceAll(name, "(", "")
	name = strings.ReplaceAll(name, ")", "")

	funcNameParts := strings.Split(name, "/")
	if len(funcNameParts) < 2 {
		return name
	}

	return funcNameParts[len(funcNameParts)-1]
}
