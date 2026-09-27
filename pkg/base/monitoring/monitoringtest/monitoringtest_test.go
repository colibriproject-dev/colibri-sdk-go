package monitoringtest

import (
	"context"
	"runtime"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func TestRecorder(t *testing.T) {
	t.Run("Should read the metrics recorded through the monitoring package", func(t *testing.T) {
		recorder := Install(t)
		ctx := context.Background()

		monitoring.Counter("test.counter", "A counter", "{call}").
			AddAttrs(ctx, 2, colibrimonitoringbase.NewAttrs("result", "success"))
		monitoring.Histogram("test.duration", "A histogram", "s").
			RecordAttrs(ctx, 0.5, colibrimonitoringbase.NewAttrs("result", "success"))
		monitoring.ObservableGauge("test.gauge", "A gauge", "{item}",
			func(context.Context) []colibrimonitoringbase.Observation {
				return []colibrimonitoringbase.Observation{
					{Value: 7, Attributes: colibrimonitoringbase.NewAttrs("queue", "q1")},
				}
			})

		counter := recorder.Metric(t, "test.counter")
		AssertShape(t, counter, "{call}", "result")
		assert.Equal(t, int64(2), CounterValue(t, counter, "result", "success"))

		count, sum := HistogramCount(t, recorder.Metric(t, "test.duration"), "result", "success")
		assert.Equal(t, uint64(1), count)
		assert.InDelta(t, 0.5, sum, 1e-9)

		assert.InDelta(t, 7.0, GaugeValue(t, recorder.Metric(t, "test.gauge"), "queue", "q1"), 1e-9)
	})

	t.Run("Should install the recorder as the global meter provider", func(t *testing.T) {
		recorder := Install(t)

		counter, err := otel.GetMeterProvider().Meter("third-party").Int64Counter("third.party")
		assert.NoError(t, err)
		counter.Add(context.Background(), 1)

		assert.Contains(t, recorder.Collect(t), "third.party")
	})

	t.Run("Should restore the previous global meter provider on cleanup", func(t *testing.T) {
		previous := otel.GetMeterProvider()

		t.Run("installed", func(t *testing.T) {
			Install(t)
			assert.NotSame(t, previous, otel.GetMeterProvider())
		})

		assert.Equal(t, previous, otel.GetMeterProvider())
	})
}

// failureRecorder is a testing.TB that records a failure instead of failing the test, so the
// assertions can be checked against metrics that must fail them.
type failureRecorder struct {
	testing.TB
	failed bool
}

func (f *failureRecorder) Helper()               {}
func (f *failureRecorder) Errorf(string, ...any) { f.failed = true }
func (f *failureRecorder) FailNow() {
	f.failed = true
	runtime.Goexit()
}

// fails reports whether assertion fails, running it where FailNow can stop it.
func fails(t *testing.T, assertion func(testing.TB)) bool {
	recorder := &failureRecorder{TB: t}
	done := make(chan struct{})
	go func() {
		defer close(done)
		assertion(recorder)
	}()
	<-done

	return recorder.failed
}

func TestAssertCataloged(t *testing.T) {
	ctx := context.Background()
	published := colibrimonitoringbase.NewAttrs("topic", "orders", "result", "success")

	t.Run("Should accept an SDK metric recorded as cataloged", func(t *testing.T) {
		recorder := Install(t)
		monitoring.Counter(monitoring.MetricMessagingPublished, "Number of messages published", "{message}").
			AddAttrs(ctx, 1, published)

		m := recorder.Metric(t, monitoring.MetricMessagingPublished)
		assert.False(t, fails(t, func(tb testing.TB) { AssertCataloged(tb, m) }))
	})

	t.Run("Should accept a third-party metric carrying part of the cataloged attributes", func(t *testing.T) {
		recorder := Install(t)
		gauge, err := otel.GetMeterProvider().Meter("otelsql").Int64Gauge(monitoring.MetricDBSQLConnectionsOpen,
			metric.WithDescription("Count of open connections in the pool"), metric.WithUnit("1"))
		require.NoError(t, err)
		gauge.Record(ctx, 3, metric.WithAttributes(attribute.String("db.instance", "main")))

		m := recorder.Metric(t, monitoring.MetricDBSQLConnectionsOpen)
		assert.False(t, fails(t, func(tb testing.TB) { AssertCataloged(tb, m) }))
	})

	failures := map[string]func(){
		"a metric outside the catalog": func() {
			monitoring.Counter("app.unknown", "Number of messages published", "{message}").AddAttrs(ctx, 1, published)
		},
		"a different unit": func() {
			monitoring.Counter(monitoring.MetricMessagingPublished, "Number of messages published", "1").
				AddAttrs(ctx, 1, published)
		},
		"a different description": func() {
			monitoring.Counter(monitoring.MetricMessagingPublished, "Published messages", "{message}").
				AddAttrs(ctx, 1, published)
		},
		"a different instrument kind": func() {
			monitoring.Histogram(monitoring.MetricMessagingPublished, "Number of messages published", "{message}").
				RecordAttrs(ctx, 1, published)
		},
		"a missing attribute": func() {
			monitoring.Counter(monitoring.MetricMessagingPublished, "Number of messages published", "{message}").
				AddAttrs(ctx, 1, colibrimonitoringbase.NewAttrs("topic", "orders"))
		},
		"an uncataloged attribute": func() {
			monitoring.Counter(monitoring.MetricMessagingPublished, "Number of messages published", "{message}").
				AddAttrs(ctx, 1, colibrimonitoringbase.NewAttrs("topic", "orders", "result", "success", "messageId", "42"))
		},
	}
	for name, record := range failures {
		t.Run("Should reject "+name, func(t *testing.T) {
			recorder := Install(t)
			record()

			metrics := recorder.Collect(t)
			require.Len(t, metrics, 1)
			for _, m := range metrics {
				assert.True(t, fails(t, func(tb testing.TB) { AssertCataloged(tb, m) }))
			}
		})
	}
}
