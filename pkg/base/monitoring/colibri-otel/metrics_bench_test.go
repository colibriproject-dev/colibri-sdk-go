package colibri_otel

import (
	"context"
	"testing"

	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// newBenchMonitoring mirrors newTestMonitoring without the testing.T helper, so the
// benchmarks measure the recording path only.
func newBenchMonitoring(b *testing.B) *MonitoringOpenTelemetry {
	b.Helper()

	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewManualReader()))

	return &MonitoringOpenTelemetry{
		meterProvider: provider,
		meter:         provider.Meter("bench"),
		counters:      make(map[string]*otelCounter),
		histograms:    make(map[string]*otelHistogram),
		gauges:        make(map[string]*otelGauge),
	}
}

// BenchmarkCounterAddAttrs is the reason Attrs exists: the OTEL attribute set and the
// option slice are built once and reused, leaving nothing to allocate per measurement.
func BenchmarkCounterAddAttrs(b *testing.B) {
	m := newBenchMonitoring(b)
	counter := m.Counter("bench.counter", "Bench counter", "1")
	attrs := colibrimonitoringbase.NewAttrs("route", "/api/users", "status", "200")
	ctx := context.Background()

	// Warm the cache so the one-time build does not land in the measured loop.
	counter.AddAttrs(ctx, 1, attrs)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		counter.AddAttrs(ctx, 1, attrs)
	}
}

// BenchmarkCounterAddMap measures the deprecated path for comparison: the map is converted
// to attributes on every call.
func BenchmarkCounterAddMap(b *testing.B) {
	m := newBenchMonitoring(b)
	counter := m.Counter("bench.counter.map", "Bench counter", "1")
	attributes := map[string]string{"route": "/api/users", "status": "200"}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		counter.Add(ctx, 1, attributes)
	}
}

func BenchmarkHistogramRecordAttrs(b *testing.B) {
	m := newBenchMonitoring(b)
	histogram := m.Histogram("bench.histogram", "Bench histogram", "ms")
	attrs := colibrimonitoringbase.NewAttrs("route", "/api/users", "status", "200")
	ctx := context.Background()

	histogram.RecordAttrs(ctx, 1, attrs)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		histogram.RecordAttrs(ctx, 1.5, attrs)
	}
}

func BenchmarkGaugeRecordAttrs(b *testing.B) {
	m := newBenchMonitoring(b)
	gauge := m.Gauge("bench.gauge", "Bench gauge", "1")
	attrs := colibrimonitoringbase.NewAttrs("pool", "main")
	ctx := context.Background()

	gauge.RecordAttrs(ctx, 1, attrs)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		gauge.RecordAttrs(ctx, 1.5, attrs)
	}
}
