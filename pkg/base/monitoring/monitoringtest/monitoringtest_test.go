package monitoringtest

import (
	"context"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
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
