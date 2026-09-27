// Package monitoringtest lets tests read the metrics the SDK records.
//
// Install swaps the active Monitoring for one backed by a ManualReader, and installs the
// same meter provider as the OTEL global so third-party instrumentation (redisotel,
// otelsql) records into it too. Everything is restored when the test ends.
//
// Instruments are bound to the Monitoring active when they are created, so install the
// recorder before initializing the component under test.
package monitoringtest

import (
	"context"
	"slices"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	colibriotel "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// Recorder reads the metrics recorded while it is installed.
type Recorder struct {
	reader *sdkmetric.ManualReader
}

// Install makes every metric recorded from now until the end of the test readable
// through the returned Recorder.
func Install(t testing.TB) *Recorder {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	previousGlobal := otel.GetMeterProvider()
	otel.SetMeterProvider(meterProvider)
	restore := monitoring.ReplaceInstance(colibriotel.NewWithMeterProvider(meterProvider))

	t.Cleanup(func() {
		restore()
		otel.SetMeterProvider(previousGlobal)
		_ = meterProvider.Shutdown(context.Background())
	})

	return &Recorder{reader: reader}
}

// Collect returns every metric currently held, keyed by instrument name.
func (r *Recorder) Collect(t testing.TB) map[string]metricdata.Metrics {
	t.Helper()

	var collected metricdata.ResourceMetrics
	require.NoError(t, r.reader.Collect(context.Background(), &collected))

	byName := map[string]metricdata.Metrics{}
	for _, scope := range collected.ScopeMetrics {
		for _, m := range scope.Metrics {
			byName[m.Name] = m
		}
	}

	return byName
}

// Metric returns the named metric, failing the test when it has not been recorded.
func (r *Recorder) Metric(t testing.TB, name string) metricdata.Metrics {
	t.Helper()

	m, ok := r.Collect(t)[name]
	require.Truef(t, ok, "metric %s was not recorded", name)

	return m
}

// AssertShape asserts the metric unit and that every data point carries exactly the given
// attribute keys — no more, which is what keeps an attribute outside the allowlist from
// slipping in.
func AssertShape(t testing.TB, m metricdata.Metrics, unit string, keys ...string) {
	t.Helper()

	assert.Equalf(t, unit, m.Unit, "unit of %s", m.Name)

	sets := attributeSets(m)
	require.NotEmptyf(t, sets, "metric %s has no data points", m.Name)

	expected := slices.Clone(keys)
	slices.Sort(expected)
	for _, set := range sets {
		assert.Equalf(t, expected, attributeKeys(set), "attribute keys of %s", m.Name)
	}
}

// AssertCataloged asserts the metric matches its entry in the SDK metric catalog: the
// description, the unit, the instrument kind and the attribute keys. A metric of the SDK
// must carry exactly the cataloged keys; a third-party metric may leave some out.
func AssertCataloged(t testing.TB, m metricdata.Metrics) {
	t.Helper()

	definition, ok := monitoring.LookupMetric(m.Name)
	require.Truef(t, ok, "metric %s is not in the catalog", m.Name)

	assert.Equalf(t, definition.Description, m.Description, "description of %s", m.Name)
	assert.Equalf(t, definition.Unit, m.Unit, "unit of %s", m.Name)
	assert.Containsf(t, kindsOf(m), definition.Kind, "%s is %T, not a %s", m.Name, m.Data, definition.Kind)

	sets := attributeSets(m)
	require.NotEmptyf(t, sets, "metric %s has no data points", m.Name)

	allowed := slices.Clone(definition.Attributes)
	slices.Sort(allowed)
	for _, set := range sets {
		keys := attributeKeys(set)
		if definition.Origin == monitoring.OriginSDK {
			assert.Equalf(t, allowed, keys, "attribute keys of %s", m.Name)
			continue
		}
		for _, key := range keys {
			assert.Containsf(t, allowed, key, "%s carries the uncataloged attribute %s", m.Name, key)
		}
	}
}

// kindsOf returns the instrument kinds that record the metric data type. Synchronous and
// observable instruments produce the same data, so both are accepted.
func kindsOf(m metricdata.Metrics) []monitoring.MetricKind {
	switch data := m.Data.(type) {
	case metricdata.Sum[int64]:
		return sumKinds(data.IsMonotonic)
	case metricdata.Sum[float64]:
		return sumKinds(data.IsMonotonic)
	case metricdata.Gauge[int64], metricdata.Gauge[float64]:
		return []monitoring.MetricKind{monitoring.KindGauge, monitoring.KindObservableGauge}
	case metricdata.Histogram[int64], metricdata.Histogram[float64]:
		return []monitoring.MetricKind{monitoring.KindHistogram}
	default:
		return nil
	}
}

func sumKinds(monotonic bool) []monitoring.MetricKind {
	if monotonic {
		return []monitoring.MetricKind{monitoring.KindCounter, monitoring.KindObservableCounter}
	}

	return []monitoring.MetricKind{monitoring.KindUpDownCounter, monitoring.KindObservableUpDownCounter}
}

// CounterValue returns the value of the integer sum data point carrying exactly the given
// attributes, as alternating key/value arguments.
func CounterValue(t testing.TB, m metricdata.Metrics, kv ...string) int64 {
	t.Helper()

	sum, ok := m.Data.(metricdata.Sum[int64])
	require.Truef(t, ok, "metric %s is %T, not an int64 sum", m.Name, m.Data)

	want := attrSet(kv)
	for _, dp := range sum.DataPoints {
		if dp.Attributes.Equals(&want) {
			return dp.Value
		}
	}

	failMissingDataPoint(t, m, kv)
	return 0
}

// HistogramCount returns how many values the float histogram data point carrying exactly
// the given attributes has recorded, and their sum.
func HistogramCount(t testing.TB, m metricdata.Metrics, kv ...string) (count uint64, sum float64) {
	t.Helper()

	histogram, ok := m.Data.(metricdata.Histogram[float64])
	require.Truef(t, ok, "metric %s is %T, not a float64 histogram", m.Name, m.Data)

	want := attrSet(kv)
	for _, dp := range histogram.DataPoints {
		if dp.Attributes.Equals(&want) {
			return dp.Count, dp.Sum
		}
	}

	failMissingDataPoint(t, m, kv)
	return 0, 0
}

// GaugeValue returns the value of the float gauge data point carrying exactly the given
// attributes.
func GaugeValue(t testing.TB, m metricdata.Metrics, kv ...string) float64 {
	t.Helper()

	gauge, ok := m.Data.(metricdata.Gauge[float64])
	require.Truef(t, ok, "metric %s is %T, not a float64 gauge", m.Name, m.Data)

	want := attrSet(kv)
	for _, dp := range gauge.DataPoints {
		if dp.Attributes.Equals(&want) {
			return dp.Value
		}
	}

	failMissingDataPoint(t, m, kv)
	return 0
}

// failMissingDataPoint fails the test when no data point carries the requested attributes.
func failMissingDataPoint(t testing.TB, m metricdata.Metrics, kv []string) {
	t.Helper()

	require.Failf(t, "data point not found", "metric %s has no data point with %v", m.Name, kv)
}

// attributeKeys returns the keys of the set, sorted.
func attributeKeys(set attribute.Set) []string {
	keys := make([]string, 0, set.Len())
	for _, kv := range set.ToSlice() {
		keys = append(keys, string(kv.Key))
	}

	return keys
}

func attrSet(kv []string) attribute.Set {
	pairs := make([]attribute.KeyValue, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		pairs = append(pairs, attribute.String(kv[i], kv[i+1]))
	}

	return attribute.NewSet(pairs...)
}

// attributeSets returns the attribute set of every data point, whatever the metric type.
func attributeSets(m metricdata.Metrics) []attribute.Set {
	switch data := m.Data.(type) {
	case metricdata.Sum[int64]:
		return setsOf(data.DataPoints)
	case metricdata.Sum[float64]:
		return setsOf(data.DataPoints)
	case metricdata.Gauge[int64]:
		return setsOf(data.DataPoints)
	case metricdata.Gauge[float64]:
		return setsOf(data.DataPoints)
	case metricdata.Histogram[int64]:
		return histogramSetsOf(data.DataPoints)
	case metricdata.Histogram[float64]:
		return histogramSetsOf(data.DataPoints)
	default:
		return nil
	}
}

func setsOf[N int64 | float64](dps []metricdata.DataPoint[N]) []attribute.Set {
	sets := make([]attribute.Set, len(dps))
	for i, dp := range dps {
		sets[i] = dp.Attributes
	}

	return sets
}

func histogramSetsOf[N int64 | float64](dps []metricdata.HistogramDataPoint[N]) []attribute.Set {
	sets := make([]attribute.Set, len(dps))
	for i, dp := range dps {
		sets[i] = dp.Attributes
	}

	return sets
}
