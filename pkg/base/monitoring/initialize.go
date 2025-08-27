package monitoring

import (
	"context"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	colibriotel "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-otel"
)

var instance colibrimonitoringbase.Monitoring

// Initialize loads the Monitoring settings according to the configured environment.
func Initialize() {
	if UseOTELMonitoring() {
		instance = colibriotel.StartOpenTelemetryMonitoring()
	} else {
		instance = colibrimonitoringbase.NewOthers()
	}
}

// UseOTELMonitoring returns true if OTEL monitoring is enabled
func UseOTELMonitoring() bool {
	return config.OTEL_EXPORTER_OTLP_ENDPOINT != ""
}

// StartTransaction start a transaction in context with name
func StartTransaction(ctx context.Context, name string) (any, context.Context) {
	return instance.StartTransaction(ctx, name)
}

func AddTransactionAttribute(transaction any, key string, value string) {
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
