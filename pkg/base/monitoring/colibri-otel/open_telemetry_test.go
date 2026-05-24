package colibri_otel

import (
	"context"
	"testing"
	"time"

	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

// newTestMonitoring creates a MonitoringOpenTelemetry backed by in-memory exporters
// so that tests do not require a real OTLP collector.
func newTestMonitoring(t *testing.T) (*MonitoringOpenTelemetry, *tracetest.SpanRecorder, *sdkmetric.ManualReader) {
	t.Helper()

	spanRecorder := tracetest.NewSpanRecorder()
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("test-service"),
		semconv.ServiceVersionKey.String("v0.0.0"),
		semconv.ServiceInstanceIDKey.String("test-instance"),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)
	otel.SetMeterProvider(mp)

	tracer := tp.Tracer("test")
	meter := mp.Meter("test")

	m := &MonitoringOpenTelemetry{
		tracerProvider: tp,
		meterProvider:  mp,
		tracer:         tracer,
		meter:          meter,
		counters:       make(map[string]metric.Int64Counter),
		histograms:     make(map[string]metric.Float64Histogram),
		gauges:         make(map[string]metric.Float64Gauge),
	}
	return m, spanRecorder, reader
}

func TestOpenTelemetry(t *testing.T) {
	t.Run("splitAndTrim", func(t *testing.T) {
		t.Run("Should return empty slice when headers is empty", func(t *testing.T) {
			result := splitAndTrim("", ",")
			assert.Empty(t, result)
		})

		t.Run("Should return slice with trimmed values", func(t *testing.T) {
			result := splitAndTrim("  a, b,  c  ", ",")
			assert.Len(t, result, 3)
			assert.Equal(t, []string{"a", "b", "c"}, result)
		})
	})

	t.Run("normalizeEndpoint", func(t *testing.T) {
		cases := map[string]string{
			"localhost:4318":                   "localhost:4318",
			"http://localhost:4318":            "localhost:4318",
			"https://collector.example.com":    "collector.example.com",
			"http://localhost:4318/v1/traces":  "localhost:4318",
			"http://localhost:4318/v1/metrics": "localhost:4318",
		}
		for input, expected := range cases {
			assert.Equal(t, expected, normalizeEndpoint(input), "input: %s", input)
		}
	})

	t.Run("parseHeaders", func(t *testing.T) {
		t.Run("Should return empty map when input is empty", func(t *testing.T) {
			result := parseHeaders("")
			assert.Empty(t, result)
		})

		t.Run("Should parse key=value pairs", func(t *testing.T) {
			result := parseHeaders("api-key=secret, x-env=prod")
			assert.Equal(t, map[string]string{"api-key": "secret", "x-env": "prod"}, result)
		})
	})

	t.Run("KindToOpenTelemetry", func(t *testing.T) {
		validations := map[colibrimonitoringbase.SpanKind]trace.SpanKind{
			colibrimonitoringbase.SpanKindInternal: trace.SpanKindInternal,
			colibrimonitoringbase.SpanKindClient:   trace.SpanKindClient,
			colibrimonitoringbase.SpanKindServer:   trace.SpanKindServer,
			colibrimonitoringbase.SpanKindProducer: trace.SpanKindProducer,
			colibrimonitoringbase.SpanKindConsumer: trace.SpanKindConsumer,
		}
		for key, value := range validations {
			assert.Equal(t, value, kindToOpenTelemetry(key))
		}
	})

	t.Run("Counter", func(t *testing.T) {
		m, _, _ := newTestMonitoring(t)

		t.Run("Should return non-nil counter", func(t *testing.T) {
			c := m.Counter("test.counter", "A test counter", "1")
			require.NotNil(t, c)
		})

		t.Run("Should record without panic", func(t *testing.T) {
			c := m.Counter("test.counter.record", "A test counter", "1")
			assert.NotPanics(t, func() {
				c.Add(context.Background(), 5, map[string]string{"env": "test"})
			})
		})

		t.Run("Should return same instrument on repeated calls", func(t *testing.T) {
			c1 := m.Counter("test.counter.cache", "A test counter", "1")
			c2 := m.Counter("test.counter.cache", "A test counter", "1")
			assert.Same(t, c1.(*otelCounter).instrument.(metric.Int64Counter), c2.(*otelCounter).instrument.(metric.Int64Counter))
		})
	})

	t.Run("Histogram", func(t *testing.T) {
		m, _, _ := newTestMonitoring(t)

		t.Run("Should return non-nil histogram", func(t *testing.T) {
			h := m.Histogram("test.histogram", "A test histogram", "ms")
			require.NotNil(t, h)
		})

		t.Run("Should record without panic", func(t *testing.T) {
			h := m.Histogram("test.histogram.record", "A test histogram", "ms")
			assert.NotPanics(t, func() {
				h.Record(context.Background(), 42.5, map[string]string{"route": "/api"})
			})
		})
	})

	t.Run("Gauge", func(t *testing.T) {
		m, _, _ := newTestMonitoring(t)

		t.Run("Should return non-nil gauge", func(t *testing.T) {
			g := m.Gauge("test.gauge", "A test gauge", "1")
			require.NotNil(t, g)
		})

		t.Run("Should record without panic", func(t *testing.T) {
			g := m.Gauge("test.gauge.record", "A test gauge", "1")
			assert.NotPanics(t, func() {
				g.Record(context.Background(), 3.14, nil)
			})
		})
	})

	t.Run("Close", func(t *testing.T) {
		m, _, _ := newTestMonitoring(t)

		t.Run("Should complete within timeout", func(t *testing.T) {
			done := make(chan struct{})
			go func() {
				m.Close()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("Close() did not complete within 10 seconds")
			}
		})
	})

	t.Run("StartTransaction", func(t *testing.T) {
		m, recorder, _ := newTestMonitoring(t)
		ctx := context.Background()

		tx, newCtx := m.StartTransaction(ctx, "test-span", colibrimonitoringbase.SpanKindInternal)
		require.NotNil(t, tx)
		require.NotNil(t, newCtx)

		m.EndTransaction(tx)

		spans := recorder.Ended()
		require.Len(t, spans, 1)
		assert.Equal(t, "test-span", spans[0].Name())
	})
}
