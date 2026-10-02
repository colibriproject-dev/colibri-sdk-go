package monitoring_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/monitoringtest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
)

// The third-party instrumentation is exercised directly, the way the SDK wires it, so a
// dependency bump that renames or reshapes one of its metrics fails here rather than in a
// dashboard. sqlDB and cacheDB check theirs against a real database.

func TestCatalogMatchesRuntimeMetrics(t *testing.T) {
	t.Run("Should catalog every Go runtime metric", func(t *testing.T) {
		recorder := monitoringtest.Install(t)
		require.NoError(t, otelruntime.Start(otelruntime.WithMeterProvider(otel.GetMeterProvider())))

		metrics := recorder.Collect(t)
		assert.Contains(t, metrics, monitoring.MetricGoMemoryUsed)
		assert.Contains(t, metrics, monitoring.MetricGoGoroutineCount)
		for _, m := range metrics {
			monitoringtest.AssertCataloged(t, m)
		}
	})

	t.Run("Should catalog the legacy runtime metrics as deprecated", func(t *testing.T) {
		t.Setenv("OTEL_GO_X_DEPRECATED_RUNTIME_METRICS", "true")
		recorder := monitoringtest.Install(t)
		require.NoError(t, otelruntime.Start(otelruntime.WithMeterProvider(otel.GetMeterProvider())))

		metrics := recorder.Collect(t)
		require.Contains(t, metrics, "process.runtime.go.goroutines")
		for name, m := range metrics {
			monitoringtest.AssertCataloged(t, m)
			if strings.HasPrefix(name, "process.runtime.") || name == "runtime.uptime" {
				definition, _ := monitoring.LookupMetric(name)
				assert.Truef(t, definition.Deprecated, "%s is not marked deprecated", name)
			}
		}
	})
}

func TestCatalogMatchesHTTPClientMetrics(t *testing.T) {
	t.Run("Should catalog every HTTP client metric", func(t *testing.T) {
		recorder := monitoringtest.Install(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
		resp, err := client.Post(server.URL, "text/plain", strings.NewReader("payload"))
		require.NoError(t, err)
		_ = resp.Body.Close()

		metrics := recorder.Collect(t)
		require.Contains(t, metrics, monitoring.MetricHTTPClientRequestDuration)
		require.Contains(t, metrics, monitoring.MetricHTTPClientRequestBodySize)
		for _, m := range metrics {
			monitoringtest.AssertCataloged(t, m)
		}
	})
}

func TestCatalogMatchesPrometheusClientCollectors(t *testing.T) {
	t.Run("Should catalog every collector /metrics serves next to the OpenTelemetry metrics", func(t *testing.T) {
		// the same collectors the client registers on prometheus.DefaultRegisterer, in a
		// registry of their own so the OpenTelemetry metrics of other tests stay out
		registry := prometheus.NewRegistry()
		registry.MustRegister(
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
			collectors.NewGoCollector(),
		)

		families, err := registry.Gather()
		require.NoError(t, err)
		require.NotEmpty(t, families)

		for _, family := range families {
			definition, ok := monitoring.LookupMetric(family.GetName())
			if !assert.Truef(t, ok, "%s is not in the catalog", family.GetName()) {
				continue
			}

			assert.Equalf(t, monitoring.OriginPromClient, definition.Origin, "origin of %s", family.GetName())
			assert.Equalf(t, definition.Description, family.GetHelp(), "description of %s", family.GetName())
			assert.Equalf(t, promClientKinds[family.GetType()], definition.Kind, "kind of %s", family.GetName())
			for _, metric := range family.GetMetric() {
				for _, label := range metric.GetLabel() {
					assert.Containsf(t, definition.Attributes, label.GetName(),
						"%s carries the uncataloged label %s", family.GetName(), label.GetName())
				}
			}
		}
	})
}

var promClientKinds = map[dto.MetricType]monitoring.MetricKind{
	dto.MetricType_COUNTER: monitoring.KindCounter,
	dto.MetricType_GAUGE:   monitoring.KindGauge,
	dto.MetricType_SUMMARY: monitoring.KindSummary,
}
