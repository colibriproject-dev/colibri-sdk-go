package colibri_otel

import (
	"context"
	"errors"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignalsFlags(t *testing.T) {
	cases := []struct {
		name       string
		signals    Signals
		wantAny    bool
		wantMetric bool
	}{
		{"zero value", Signals{}, false, false},
		{"tracing only", Signals{Tracing: true}, true, false},
		{"OTLP metrics only", Signals{OTLPMetrics: true}, true, true},
		{"Prometheus metrics only", Signals{PrometheusMetrics: true}, true, true},
		{"every signal", Signals{Tracing: true, OTLPMetrics: true, PrometheusMetrics: true}, true, true},
	}

	for _, tc := range cases {
		t.Run("Should report the enabled signals with "+tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantAny, tc.signals.AnyEnabled())
			assert.Equal(t, tc.wantMetric, tc.signals.AnyMetrics())
		})
	}
}

// rejectingRegisterer stands in for a registry that refuses the collector.
type rejectingRegisterer struct {
	err error
}

func (r rejectingRegisterer) Register(prometheus.Collector) error { return r.err }

func (r rejectingRegisterer) MustRegister(...prometheus.Collector) {}

func (r rejectingRegisterer) Unregister(prometheus.Collector) bool { return false }

func TestBuildPrometheusReader(t *testing.T) {
	ctx := context.Background()

	t.Run("Should return a reader registered on the given registry", func(t *testing.T) {
		registry := prometheus.NewRegistry()

		reader := buildPrometheusReader(ctx, registry)

		require.NotNil(t, reader)
		assert.NoError(t, reader.Shutdown(ctx))
	})

	t.Run("Should return nil when the registry rejects the collector", func(t *testing.T) {
		registerer := rejectingRegisterer{err: errors.New("registry closed")}

		assert.Nil(t, buildPrometheusReader(ctx, registerer))
	})

	t.Run("Should return nil when the collector is already registered", func(t *testing.T) {
		registerer := rejectingRegisterer{err: prometheus.AlreadyRegisteredError{}}

		assert.Nil(t, buildPrometheusReader(ctx, registerer))
	})
}

// TestPrometheusOnlyMonitoring covers the configuration this issue exists for: no
// collector, metrics scrapable from the process itself, traces inert.
func TestPrometheusOnlyMonitoring(t *testing.T) {
	registry := prometheus.NewRegistry()
	endpoint := config.OTEL_EXPORTER_OTLP_ENDPOINT
	config.OTEL_EXPORTER_OTLP_ENDPOINT = ""
	t.Cleanup(func() { config.OTEL_EXPORTER_OTLP_ENDPOINT = endpoint })

	monitoring := StartOpenTelemetryMonitoring(Signals{
		PrometheusMetrics:    true,
		PrometheusRegisterer: registry,
	})
	require.NotNil(t, monitoring)
	t.Cleanup(monitoring.Close)

	instance, ok := monitoring.(*MonitoringOpenTelemetry)
	require.True(t, ok)

	t.Run("Should build a meter provider and no tracer provider", func(t *testing.T) {
		assert.NotNil(t, instance.meterProvider)
		assert.Nil(t, instance.tracerProvider)
	})

	t.Run("Should expose a recorded metric on the registry", func(t *testing.T) {
		counter := instance.Counter("prometheus_only_requests", "Requests served", "1")
		counter.AddAttrs(context.Background(), 3, colibrimonitoringbase.NewAttrs("route", "/api"))

		gathered, err := registry.Gather()
		require.NoError(t, err)

		names := make([]string, 0, len(gathered))
		for _, family := range gathered {
			names = append(names, family.GetName())
		}
		assert.Contains(t, names, "prometheus_only_requests_total")
	})

	t.Run("Should keep transactions inert without a tracer provider", func(t *testing.T) {
		transaction, ctx := instance.StartTransaction(context.Background(), "noop-span", colibrimonitoringbase.SpanKindInternal)

		require.NotNil(t, transaction)
		assert.NotPanics(t, func() {
			instance.AddTransactionAttribute(transaction, "key", "value")
			instance.EndTransaction(transaction)
			instance.EndTransactionSegment(instance.StartTransactionSegment(ctx, "segment", nil))
		})
	})
}

// TestTracingOnlyMonitoring is the mirror case: a collector for traces, metrics disabled.
func TestTracingOnlyMonitoring(t *testing.T) {
	endpoint := config.OTEL_EXPORTER_OTLP_ENDPOINT
	config.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4318"
	t.Cleanup(func() { config.OTEL_EXPORTER_OTLP_ENDPOINT = endpoint })

	monitoring := StartOpenTelemetryMonitoring(Signals{Tracing: true})
	require.NotNil(t, monitoring)
	t.Cleanup(monitoring.Close)

	instance, ok := monitoring.(*MonitoringOpenTelemetry)
	require.True(t, ok)

	t.Run("Should build a tracer provider and no meter provider", func(t *testing.T) {
		assert.NotNil(t, instance.tracerProvider)
		assert.Nil(t, instance.meterProvider)
	})

	t.Run("Should keep instruments usable without a meter provider", func(t *testing.T) {
		assert.NotPanics(t, func() {
			instance.Counter("tracing.only.counter", "A counter", "1").
				AddAttrs(context.Background(), 1, colibrimonitoringbase.NewAttrs("k", "v"))
			instance.Histogram("tracing.only.histogram", "A histogram", "ms").
				RecordAttrs(context.Background(), 1, colibrimonitoringbase.Attrs{})
			instance.Gauge("tracing.only.gauge", "A gauge", "1").
				RecordAttrs(context.Background(), 1, colibrimonitoringbase.Attrs{})
		})
	})

	t.Run("Should register the observable gauge without emitting", func(t *testing.T) {
		registration := instance.ObservableGauge("tracing.only.observable", "A gauge", "1",
			func(context.Context) []colibrimonitoringbase.Observation { return nil })

		require.NotNil(t, registration)
		assert.NoError(t, registration.Unregister())
	})

	t.Run("Should fall back to a noop meter provider for the SQL driver", func(t *testing.T) {
		assert.NotNil(t, instance.sqlMeterProvider())
	})
}

func TestCloseWithDisabledSignals(t *testing.T) {
	t.Run("Should not panic when no provider was built", func(t *testing.T) {
		instance := &MonitoringOpenTelemetry{}

		assert.NotPanics(t, instance.Close)
	})
}
