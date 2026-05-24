package monitoring

import (
	"context"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	colibriotel "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-otel"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
)

var instance colibrimonitoringbase.Monitoring

// Initialize loads the Monitoring settings according to the configured environment.
func Initialize() {
	if UseOTELMonitoring() {
		instance = colibriotel.StartOpenTelemetryMonitoring()
		observer.Attach(instance.(observer.Observer))
	} else {
		instance = colibrimonitoringbase.NewOthers()
	}
}

// UseOTELMonitoring returns true if OTEL monitoring is enabled
func UseOTELMonitoring() bool {
	return config.OTEL_EXPORTER_OTLP_ENDPOINT != ""
}

// StartTransaction start a transaction in context with name
func StartTransaction(ctx context.Context, name string, kind colibrimonitoringbase.SpanKind) (any, context.Context) {
	return instance.StartTransaction(ctx, name, kind)
}

func AddTransactionAttribute(transaction any, key, value string) {
	instance.AddTransactionAttribute(transaction, key, value)
}

// EndTransaction ends the transaction
func EndTransaction(transaction any) {
	instance.EndTransaction(transaction)
}

// StartTransactionSegment start a transaction segment inside opened transaction with name and atributes
func StartTransactionSegment(ctx context.Context, name string, attributes map[string]string) any {
	return instance.StartTransactionSegment(ctx, name, attributes)
}

// EndTransactionSegment ends the transaction segment
func EndTransactionSegment(segment any) {
	instance.EndTransactionSegment(segment)
}

// GetTransactionInContext returns transaction inside a context
func GetTransactionInContext(ctx context.Context) any {
	return instance.GetTransactionInContext(ctx)
}

// NoticeError notices an error in Monitoring provider
func NoticeError(transaction any, err error) {
	instance.NoticeError(transaction, err)
}

// GetSQLDBDriverName return driver name for monitoring provider
func GetSQLDBDriverName() string {
	return instance.GetSQLDBDriverName()
}

// Counter returns a named counter instrument for recording monotonically increasing values.
func Counter(name, description, unit string) colibrimonitoringbase.Counter {
	return instance.Counter(name, description, unit)
}

// Histogram returns a named histogram instrument for recording value distributions.
func Histogram(name, description, unit string) colibrimonitoringbase.Histogram {
	return instance.Histogram(name, description, unit)
}

// Gauge returns a named gauge instrument for recording current values.
func Gauge(name, description, unit string) colibrimonitoringbase.Gauge {
	return instance.Gauge(name, description, unit)
}
