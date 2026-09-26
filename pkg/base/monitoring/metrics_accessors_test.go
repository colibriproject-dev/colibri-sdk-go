package monitoring

import (
	"context"
	"testing"

	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/stretchr/testify/assert"
)

func TestMetricAccessorsWithoutInitialize(t *testing.T) {
	restore := ReplaceInstance(nil)
	defer restore()

	t.Run("Should return inert instruments instead of panicking", func(t *testing.T) {
		ctx := context.Background()
		attrs := colibrimonitoringbase.NewAttrs("key", "value")

		assert.NotPanics(t, func() {
			Counter("uninitialized.counter", "", "1").AddAttrs(ctx, 1, attrs)
			Histogram("uninitialized.histogram", "", "s").RecordAttrs(ctx, 1, attrs)
			Gauge("uninitialized.gauge", "", "1").RecordAttrs(ctx, 1, attrs)

			registration := ObservableGauge("uninitialized.observable", "", "1",
				func(context.Context) []colibrimonitoringbase.Observation { return nil })
			assert.NoError(t, registration.Unregister())
		})
	})
}

func TestReplaceInstance(t *testing.T) {
	t.Run("Should restore the previous instance", func(t *testing.T) {
		previous := instance
		replacement := colibrimonitoringbase.NewOthers()

		restore := ReplaceInstance(replacement)
		assert.Same(t, replacement, instance)

		restore()
		assert.Equal(t, previous, instance)
	})
}
