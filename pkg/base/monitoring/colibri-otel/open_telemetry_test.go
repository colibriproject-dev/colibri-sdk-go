package colibri_otel

import (
	"testing"

	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

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

	t.Run("KindToOpenTelemetry", func(t *testing.T) {
		validations := map[colibrimonitoringbase.SpanKind]trace.SpanKind{
			colibrimonitoringbase.SpanKindInternal: trace.SpanKindInternal,
			colibrimonitoringbase.SpanKindClient:   trace.SpanKindClient,
			colibrimonitoringbase.SpanKindServer:   trace.SpanKindServer,
			colibrimonitoringbase.SpanKindProducer: trace.SpanKindProducer,
			colibrimonitoringbase.SpanKindConsumer: trace.SpanKindConsumer,
		}

		for key, value := range validations {
			result := kindToOpenTelemetry(key)
			assert.Equal(t, value, result)
		}
	})
}
