package colibri_otel

import (
	"context"
	"testing"

	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// collect reads every metric currently held by the reader, keyed by instrument name.
func collect(t *testing.T, reader *sdkmetric.ManualReader) map[string]metricdata.Metrics {
	t.Helper()

	var collected metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &collected))

	byName := map[string]metricdata.Metrics{}
	for _, scope := range collected.ScopeMetrics {
		for _, m := range scope.Metrics {
			byName[m.Name] = m
		}
	}

	return byName
}

func TestAttrsRecording(t *testing.T) {
	t.Run("Should record a counter under the Attrs attributes", func(t *testing.T) {
		m, _, reader := newTestMonitoring(t)
		attrs := colibrimonitoringbase.NewAttrs("route", "/api/users", "status", "200")

		counter := m.Counter("attrs.counter", "A test counter", "1")
		counter.AddAttrs(context.Background(), 2, attrs)
		counter.AddAttrs(context.Background(), 3, attrs)

		sum, ok := collect(t, reader)["attrs.counter"].Data.(metricdata.Sum[int64])
		require.True(t, ok)
		require.Len(t, sum.DataPoints, 1)
		assert.Equal(t, int64(5), sum.DataPoints[0].Value)
		assert.Equal(t, attribute.NewSet(
			attribute.String("route", "/api/users"),
			attribute.String("status", "200"),
		), sum.DataPoints[0].Attributes)
	})

	t.Run("Should record a histogram under the Attrs attributes", func(t *testing.T) {
		m, _, reader := newTestMonitoring(t)

		histogram := m.Histogram("attrs.histogram", "A test histogram", "ms")
		histogram.RecordAttrs(context.Background(), 42.5, colibrimonitoringbase.NewAttrs("route", "/api"))

		data, ok := collect(t, reader)["attrs.histogram"].Data.(metricdata.Histogram[float64])
		require.True(t, ok)
		require.Len(t, data.DataPoints, 1)
		assert.Equal(t, uint64(1), data.DataPoints[0].Count)
	})

	t.Run("Should record a gauge under the Attrs attributes", func(t *testing.T) {
		m, _, reader := newTestMonitoring(t)

		gauge := m.Gauge("attrs.gauge", "A test gauge", "1")
		gauge.RecordAttrs(context.Background(), 3.14, colibrimonitoringbase.NewAttrs("pool", "main"))

		data, ok := collect(t, reader)["attrs.gauge"].Data.(metricdata.Gauge[float64])
		require.True(t, ok)
		require.Len(t, data.DataPoints, 1)
		assert.InDelta(t, 3.14, data.DataPoints[0].Value, 0.0001)
	})

	t.Run("Should record the zero value Attrs without attributes", func(t *testing.T) {
		m, _, reader := newTestMonitoring(t)

		counter := m.Counter("attrs.counter.empty", "A test counter", "1")
		assert.NotPanics(t, func() {
			counter.AddAttrs(context.Background(), 1, colibrimonitoringbase.Attrs{})
		})

		sum := collect(t, reader)["attrs.counter.empty"].Data.(metricdata.Sum[int64])
		require.Len(t, sum.DataPoints, 1)
		assert.Equal(t, 0, sum.DataPoints[0].Attributes.Len())
	})

	t.Run("Should reuse the cached options across instruments", func(t *testing.T) {
		attrs := colibrimonitoringbase.NewAttrs("route", "/api/users")

		first := cachedAttrs(attrs)
		second := cachedAttrs(attrs)

		assert.Same(t, first, second)
	})
}

func TestInstrumentCreationFailure(t *testing.T) {
	// An empty name is rejected by the OTEL SDK. Creation used to call logging.Fatal, so a
	// bad metric name took the process down; it now degrades to an inert instrument.
	m, _, _ := newTestMonitoring(t)

	t.Run("Should return a usable counter", func(t *testing.T) {
		counter := m.Counter("", "No name", "1")

		require.NotNil(t, counter)
		assert.NotPanics(t, func() {
			counter.AddAttrs(context.Background(), 1, colibrimonitoringbase.NewAttrs("k", "v"))
			counter.Add(context.Background(), 1, map[string]string{"k": "v"})
		})
	})

	t.Run("Should return a usable histogram", func(t *testing.T) {
		histogram := m.Histogram("", "No name", "ms")

		require.NotNil(t, histogram)
		assert.NotPanics(t, func() {
			histogram.RecordAttrs(context.Background(), 1, colibrimonitoringbase.NewAttrs("k", "v"))
			histogram.Record(context.Background(), 1, map[string]string{"k": "v"})
		})
	})

	t.Run("Should return a usable gauge", func(t *testing.T) {
		gauge := m.Gauge("", "No name", "1")

		require.NotNil(t, gauge)
		assert.NotPanics(t, func() {
			gauge.RecordAttrs(context.Background(), 1, colibrimonitoringbase.NewAttrs("k", "v"))
			gauge.Record(context.Background(), 1, map[string]string{"k": "v"})
		})
	})

	t.Run("Should return an inert observable gauge registration", func(t *testing.T) {
		registration := m.ObservableGauge("", "No name", "1", func(context.Context) []colibrimonitoringbase.Observation {
			return nil
		})

		require.NotNil(t, registration)
		assert.NoError(t, registration.Unregister())
	})
}

func TestObservableGauge(t *testing.T) {
	t.Run("Should report the callback observations on every collection", func(t *testing.T) {
		m, _, reader := newTestMonitoring(t)
		attrs := colibrimonitoringbase.NewAttrs("pool", "main")
		value := 7.0

		registration := m.ObservableGauge("pool.size", "Connections in the pool", "1",
			func(context.Context) []colibrimonitoringbase.Observation {
				return []colibrimonitoringbase.Observation{{Value: value, Attributes: attrs}}
			})
		require.NotNil(t, registration)

		data, ok := collect(t, reader)["pool.size"].Data.(metricdata.Gauge[float64])
		require.True(t, ok)
		require.Len(t, data.DataPoints, 1)
		assert.InDelta(t, 7.0, data.DataPoints[0].Value, 0.0001)
		assert.Equal(t, attribute.NewSet(attribute.String("pool", "main")), data.DataPoints[0].Attributes)

		value = 9.0
		data = collect(t, reader)["pool.size"].Data.(metricdata.Gauge[float64])
		require.Len(t, data.DataPoints, 1)
		assert.InDelta(t, 9.0, data.DataPoints[0].Value, 0.0001)
	})

	t.Run("Should report one data point per observation", func(t *testing.T) {
		m, _, reader := newTestMonitoring(t)

		m.ObservableGauge("queue.depth", "Messages waiting", "1",
			func(context.Context) []colibrimonitoringbase.Observation {
				return []colibrimonitoringbase.Observation{
					{Value: 1, Attributes: colibrimonitoringbase.NewAttrs("queue", "orders")},
					{Value: 2, Attributes: colibrimonitoringbase.NewAttrs("queue", "payments")},
				}
			})

		data := collect(t, reader)["queue.depth"].Data.(metricdata.Gauge[float64])
		assert.Len(t, data.DataPoints, 2)
	})

	t.Run("Should stop observing after Unregister", func(t *testing.T) {
		m, _, reader := newTestMonitoring(t)
		calls := 0

		registration := m.ObservableGauge("cache.entries", "Entries in cache", "1",
			func(context.Context) []colibrimonitoringbase.Observation {
				calls++
				return []colibrimonitoringbase.Observation{{Value: 1}}
			})

		collect(t, reader)
		require.Equal(t, 1, calls)

		require.NoError(t, registration.Unregister())
		collect(t, reader)

		assert.Equal(t, 1, calls)
	})
}

// TestRecordingAllocations guards the reason Attrs exists. The benchmarks report the same
// numbers, but only a test fails the build when a change puts an allocation back on the
// recording path.
func TestRecordingAllocations(t *testing.T) {
	m, _, _ := newTestMonitoring(t)
	ctx := context.Background()
	attrs := colibrimonitoringbase.NewAttrs("route", "/api/users", "status", "200")

	cases := map[string]func(){
		"counter": func() {
			counter := m.Counter("alloc.counter", "Alloc counter", "1")
			counter.AddAttrs(ctx, 1, attrs)
		},
		"histogram": func() {
			histogram := m.Histogram("alloc.histogram", "Alloc histogram", "ms")
			histogram.RecordAttrs(ctx, 1.5, attrs)
		},
		"gauge": func() {
			gauge := m.Gauge("alloc.gauge", "Alloc gauge", "1")
			gauge.RecordAttrs(ctx, 1.5, attrs)
		},
	}

	for name, record := range cases {
		t.Run("Should not allocate when recording a "+name+" with a reused Attrs", func(t *testing.T) {
			record() // Warm the instrument cache and the Attrs cache.

			assert.Zero(t, testing.AllocsPerRun(100, record))
		})
	}
}
