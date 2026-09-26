package monitoring

import (
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	colibriotel "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// restoreMonitoringState snapshots the package and config state the gates read, so a test
// that flips a signal cannot decide whether the next one passes.
func restoreMonitoringState(t *testing.T) {
	t.Helper()

	previousInstance := instance
	endpoint := config.OTEL_EXPORTER_OTLP_ENDPOINT
	metricsEndpoint := config.OTEL_EXPORTER_OTLP_METRICS_ENDPOINT
	traces := config.OTEL_TRACES_ENABLED
	metrics := config.OTEL_METRICS_ENABLED
	prom := config.OTEL_METRICS_PROMETHEUS_ENABLED

	t.Cleanup(func() {
		instance = previousInstance
		config.OTEL_EXPORTER_OTLP_ENDPOINT = endpoint
		config.OTEL_EXPORTER_OTLP_METRICS_ENDPOINT = metricsEndpoint
		config.OTEL_TRACES_ENABLED = traces
		config.OTEL_METRICS_ENABLED = metrics
		config.OTEL_METRICS_PROMETHEUS_ENABLED = prom
	})
}

func TestSignalGates(t *testing.T) {
	restoreMonitoringState(t)

	cases := []struct {
		name       string
		endpoint   string
		traces     bool
		metrics    bool
		prometheus bool

		wantTracing    bool
		wantOTLP       bool
		wantPrometheus bool
		wantMetrics    bool
		wantAny        bool
	}{
		{
			name: "defaults without a collector expose Prometheus metrics only", endpoint: "",
			traces: true, metrics: true, prometheus: true,
			wantTracing: false, wantOTLP: false, wantPrometheus: true, wantMetrics: true, wantAny: true,
		},
		{
			name: "a collector turns on traces and the OTLP reader", endpoint: "http://localhost:4318",
			traces: true, metrics: true, prometheus: true,
			wantTracing: true, wantOTLP: true, wantPrometheus: true, wantMetrics: true, wantAny: true,
		},
		{
			name: "OTLP only when Prometheus is disabled", endpoint: "http://localhost:4318",
			traces: true, metrics: true, prometheus: false,
			wantTracing: true, wantOTLP: true, wantPrometheus: false, wantMetrics: true, wantAny: true,
		},
		{
			name: "Prometheus only when traces are disabled", endpoint: "http://localhost:4318",
			traces: false, metrics: true, prometheus: true,
			wantTracing: false, wantOTLP: true, wantPrometheus: true, wantMetrics: true, wantAny: true,
		},
		{
			name: "traces only when metrics are disabled", endpoint: "http://localhost:4318",
			traces: true, metrics: false, prometheus: true,
			wantTracing: true, wantOTLP: false, wantPrometheus: false, wantMetrics: false, wantAny: true,
		},
		{
			name: "nothing when every signal is disabled", endpoint: "http://localhost:4318",
			traces: false, metrics: false, prometheus: false,
			wantTracing: false, wantOTLP: false, wantPrometheus: false, wantMetrics: false, wantAny: false,
		},
		{
			name: "nothing when metrics are disabled and no collector is set", endpoint: "",
			traces: true, metrics: false, prometheus: true,
			wantTracing: false, wantOTLP: false, wantPrometheus: false, wantMetrics: false, wantAny: false,
		},
	}

	for _, tc := range cases {
		t.Run("Should report "+tc.name, func(t *testing.T) {
			config.OTEL_EXPORTER_OTLP_ENDPOINT = tc.endpoint
			config.OTEL_EXPORTER_OTLP_METRICS_ENDPOINT = ""
			config.OTEL_TRACES_ENABLED = tc.traces
			config.OTEL_METRICS_ENABLED = tc.metrics
			config.OTEL_METRICS_PROMETHEUS_ENABLED = tc.prometheus

			assert.Equal(t, tc.wantTracing, UseTracing(), "UseTracing")
			assert.Equal(t, tc.wantOTLP, UseOTLPMetrics(), "UseOTLPMetrics")
			assert.Equal(t, tc.wantPrometheus, UsePrometheusMetrics(), "UsePrometheusMetrics")
			assert.Equal(t, tc.wantMetrics, UseMetrics(), "UseMetrics")
			assert.Equal(t, tc.wantAny, UseOTELMonitoring(), "UseOTELMonitoring")
		})
	}

	t.Run("Should enable the OTLP reader from the metrics endpoint alone", func(t *testing.T) {
		config.OTEL_EXPORTER_OTLP_ENDPOINT = ""
		config.OTEL_EXPORTER_OTLP_METRICS_ENDPOINT = "http://localhost:4318"
		config.OTEL_TRACES_ENABLED = true
		config.OTEL_METRICS_ENABLED = true

		assert.True(t, UseOTLPMetrics())
		assert.False(t, UseTracing(), "a metrics-only endpoint must not turn on traces")
	})
}

func TestInitialize(t *testing.T) {
	t.Run("Should use the noop monitoring when every signal is disabled", func(t *testing.T) {
		restoreMonitoringState(t)
		instance = nil

		config.OTEL_EXPORTER_OTLP_ENDPOINT = ""
		config.OTEL_METRICS_ENABLED = false
		config.OTEL_METRICS_PROMETHEUS_ENABLED = false

		Initialize()

		require.NotNil(t, instance)
		assert.IsType(t, colibrimonitoringbase.NewOthers(), instance)
	})

	t.Run("Should keep the first instance when called again", func(t *testing.T) {
		restoreMonitoringState(t)
		instance = nil

		// The Prometheus reader stays off: a second registration on the default registry
		// would outlive this test and break any later scrape in this binary.
		config.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4318"
		config.OTEL_TRACES_ENABLED = true
		config.OTEL_METRICS_ENABLED = true
		config.OTEL_METRICS_PROMETHEUS_ENABLED = false

		Initialize()
		first := instance
		require.NotNil(t, first)

		assert.NotPanics(t, Initialize)
		assert.Same(t, first, instance)
	})
}

func TestSignals(t *testing.T) {
	t.Run("Should report no enabled signal on the zero value", func(t *testing.T) {
		var signals colibriotel.Signals

		assert.False(t, signals.AnyEnabled())
		assert.False(t, signals.AnyMetrics())
	})

	t.Run("Should not report metrics when only tracing is enabled", func(t *testing.T) {
		signals := colibriotel.Signals{Tracing: true}

		assert.True(t, signals.AnyEnabled())
		assert.False(t, signals.AnyMetrics())
	})

	t.Run("Should report metrics from either reader", func(t *testing.T) {
		assert.True(t, colibriotel.Signals{OTLPMetrics: true}.AnyMetrics())
		assert.True(t, colibriotel.Signals{PrometheusMetrics: true}.AnyMetrics())
	})
}
