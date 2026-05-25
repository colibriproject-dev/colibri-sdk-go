package restserver

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/security"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	authorizationHeader = "Authorization"
	userIDHeader        = "X-User-Id"
	tenantIDHeader      = "X-Tenant-Id"
)

type MiddlewareError struct {
	Err        error `json:"error"`
	StatusCode int   `json:"statusCode"`
}

func (e MiddlewareError) Error() string {
	return e.Err.Error()
}

func NewMiddlewareError(statusCode int, err error) *MiddlewareError {
	return &MiddlewareError{StatusCode: statusCode, Err: err}
}

type CustomMiddleware interface {
	Apply(ctx WebContext) *MiddlewareError
}

type CustomAuthenticationMiddleware interface {
	Apply(ctx WebContext) (*security.AuthenticationContext, error)
}

func authenticationContextFiberMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !strings.Contains(c.Request().URI().String(), string(AuthenticatedApi)) {
			return c.Next()
		}

		tenantID := string(c.Request().Header.Peek(tenantIDHeader))
		userID := string(c.Request().Header.Peek(userIDHeader))
		authCtx := security.NewAuthenticationContext(tenantID, userID)
		if authCtx.Valid() {
			newCtx := authCtx.SetInContext(c.UserContext())
			c.SetUserContext(newCtx)
			return c.Next()
		}

		c.Status(http.StatusUnauthorized)
		c.Request()
		return c.JSON(&Error{Error: "user not authenticated"})
	}
}

func customAuthenticationContextFiberMiddleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		webCtx := &fiberWebContext{ctx: ctx}
		authCtx, err := customAuth.Apply(webCtx)
		if err != nil {
			ctx.Status(http.StatusUnauthorized)
			return ctx.JSON(err)
		}

		newCtx := authCtx.SetInContext(ctx.UserContext())
		ctx.SetUserContext(newCtx)
		return ctx.Next()
	}
}

func newOpenTelemetryFiberMiddleware() fiber.Handler {
	return otelfiber.Middleware(
		otelfiber.WithoutMetrics(true),
		otelfiber.WithSpanNameFormatter(func(ctx *fiber.Ctx) string {
			// utils.CopyString: GetRespHeader returns an UnsafeString backed by fasthttp's
			// buffer, which is reused for the next request on the same keep-alive connection.
			// Without copying, the span/metric attribute value mutates after the request ends
			// — e.g. "/public/v1/subscription-plans" becomes "/health/v1/subscription-plans"
			// when "/health" overwrites the first bytes of the buffer.
			route := utils.CopyString(ctx.GetRespHeader(parameterizedURLHeaderKey))
			if route == "" {
				route = ctx.Route().Path
			}
			trace.SpanFromContext(ctx.UserContext()).SetAttributes(attribute.String("http.route", route))
			return fmt.Sprintf("%s %s", ctx.Method(), route)
		}),
	)
}

func httpMetricsFiberMiddleware() fiber.Handler {
	meter := otel.GetMeterProvider().Meter("github.com/colibriproject-dev/colibri-sdk-go")

	// HTTP semconv stable v1 (semconv >= 1.23): `http.server.request.duration` in seconds.
	// New Relic APM derives the Transactions view from this metric name + `http.route`
	// attribute; the older `http.server.duration` (milliseconds) is shown as "unknown".
	httpDuration, _ := meter.Float64Histogram("http.server.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of HTTP server requests"),
	)
	activeRequests, _ := meter.Int64UpDownCounter("http.server.active_requests",
		metric.WithUnit("{request}"),
		metric.WithDescription("Number of active HTTP server requests"),
	)
	requestSize, _ := meter.Int64Histogram("http.server.request.body.size",
		metric.WithUnit("By"),
		metric.WithDescription("Size of HTTP server request bodies"),
	)
	responseSize, _ := meter.Int64Histogram("http.server.response.body.size",
		metric.WithUnit("By"),
		metric.WithDescription("Size of HTTP server response bodies"),
	)

	return func(c *fiber.Ctx) error {
		start := time.Now()
		ctx := c.UserContext()

		reqAttrs := []attribute.KeyValue{
			attribute.String("url.scheme", c.Protocol()),
			attribute.String("server.address", c.Hostname()),
			attribute.String("http.request.method", c.Method()),
		}
		reqBodySize := int64(len(c.Request().Body()))

		activeRequests.Add(ctx, 1, metric.WithAttributes(reqAttrs...))
		defer func() {
			// utils.CopyString — see comment in newOpenTelemetryFiberMiddleware: without
			// copying, the metric attribute value mutates when fasthttp reuses the response
			// header buffer for the next request on the same connection.
			route := utils.CopyString(c.GetRespHeader(parameterizedURLHeaderKey))
			if route == "" {
				route = c.Route().Path
			}

			respAttrs := append(reqAttrs,
				attribute.Int("http.response.status_code", c.Response().StatusCode()),
				attribute.String("http.route", route),
			)

			activeRequests.Add(ctx, -1, metric.WithAttributes(reqAttrs...))
			httpDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(respAttrs...))
			requestSize.Record(ctx, reqBodySize, metric.WithAttributes(respAttrs...))
			responseSize.Record(ctx, int64(len(c.Response().Body())), metric.WithAttributes(respAttrs...))
		}()

		return c.Next()
	}
}

func accessControlFiberMiddleware() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     config.CORS_ALLOW_ORIGINS,
		AllowMethods:     config.CORS_ALLOW_METHODS,
		AllowHeaders:     config.CORS_ALLOW_HEADERS,
		ExposeHeaders:    config.CORS_EXPOSE_HEADERS,
		AllowCredentials: config.CORS_ALLOW_CREDENTIALS,
		MaxAge:           config.CORS_MAX_AGE,
	})
}

func panicRecoverMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logging.Error(c.UserContext()).
					Err(fmt.Errorf("%v", r)).
					AddParam("path", c.Path()).
					AddParam("method", c.Method()).
					Msg("panic recovered")

				c.Status(fiber.StatusInternalServerError)
				err = c.JSON(Error{Error: "internal server error occurred"})
			}
		}()

		return c.Next()
	}
}

func correlationIdMiddleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		correlationID := ctx.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		ctx.SetUserContext(logging.InjectCorrelationIDInContext(ctx.UserContext(), correlationID))
		if monitoring.UseOTELMonitoring() {
			txn := monitoring.GetTransactionInContext(ctx.UserContext())
			monitoring.AddTransactionAttribute(txn, logging.CorrelationIDParam, correlationID)
		}
		return ctx.Next()
	}
}
