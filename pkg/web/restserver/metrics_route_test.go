package restserver

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withSignals sets the gate configuration for one test and restores it afterwards.
func withSignals(t *testing.T, endpoint string, traces, metrics, prometheusMetrics bool) {
	t.Helper()

	previous := [3]bool{config.OTEL_TRACES_ENABLED, config.OTEL_METRICS_ENABLED, config.OTEL_METRICS_PROMETHEUS_ENABLED}
	previousEndpoint := config.OTEL_EXPORTER_OTLP_ENDPOINT

	t.Cleanup(func() {
		config.OTEL_TRACES_ENABLED = previous[0]
		config.OTEL_METRICS_ENABLED = previous[1]
		config.OTEL_METRICS_PROMETHEUS_ENABLED = previous[2]
		config.OTEL_EXPORTER_OTLP_ENDPOINT = previousEndpoint
	})

	config.OTEL_EXPORTER_OTLP_ENDPOINT = endpoint
	config.OTEL_TRACES_ENABLED = traces
	config.OTEL_METRICS_ENABLED = metrics
	config.OTEL_METRICS_PROMETHEUS_ENABLED = prometheusMetrics
}

func TestInjectMiddlewares(t *testing.T) {
	// Fiber does not expose its middleware stack, so each case asserts the server still
	// serves a request under that gate combination — the point being that the trace and
	// metric middlewares are now registered independently.
	cases := []struct {
		name              string
		endpoint          string
		traces            bool
		metrics           bool
		prometheusMetrics bool
	}{
		{"Prometheus metrics only, no collector", "", true, true, true},
		{"traces and metrics with a collector", "http://localhost:4318", true, true, true},
		{"traces only", "http://localhost:4318", true, false, false},
		{"no observability signal", "", false, false, false},
	}

	for _, tc := range cases {
		t.Run("Should serve a request with "+tc.name, func(t *testing.T) {
			withSignals(t, tc.endpoint, tc.traces, tc.metrics, tc.prometheusMetrics)

			server := &fiberWebServer{}
			server.initialize()
			server.injectMiddlewares()
			server.srv.Get("/ping", func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

			resp, err := server.srv.Test(newTestRequest(t, http.MethodGet, "/ping"))
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

// TestMetricsRoute covers the gap the issue opens with: /metrics served the client_golang
// default registry, so OTEL metrics never reached it and the route carried only the Go and
// process collectors. The Prometheus reader registers on that same registry, so a metric
// recorded through the SDK now shows up on a scrape.
func TestMetricsRoute(t *testing.T) {
	require.True(t, monitoring.UsePrometheusMetrics(), "the Prometheus reader is expected on by default")

	monitoring.Counter("metrics_route_requests", "Requests served", "1").
		AddAttrs(context.Background(), 1, colibrimonitoringbase.NewAttrs("route", "/metrics"))

	server := &fiberWebServer{}
	server.initialize()
	server.addMetricsRoute()

	resp, err := server.srv.Test(newTestRequest(t, http.MethodGet, "/metrics"))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	exposition := string(body)

	t.Run("Should keep exposing the Go collector metrics", func(t *testing.T) {
		assert.Contains(t, exposition, "go_goroutines")
	})

	t.Run("Should expose the metrics recorded through the OTEL meter provider", func(t *testing.T) {
		assert.Contains(t, exposition, "metrics_route_requests_total")
	})
}
