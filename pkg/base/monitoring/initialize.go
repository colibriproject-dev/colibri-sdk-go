package monitoring

import (
	"context"
	"net/http"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	colibri_monitoring_base "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	colibri_otel "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-otel"
)

var instance colibri_monitoring_base.Monitoring

// Initialize loads the Monitoring settings according to the configured environment.
func Initialize() {
	if useOTELMonitoring() {
		instance = colibri_otel.StartOpenTelemetryMonitoring()
	} else {
		instance = colibri_monitoring_base.NewOthers()
	}
}

func useOTELMonitoring() bool {
	return config.OTEL_EXPORTER_OTLP_ENDPOINT != ""
}

// StartTransaction start a transaction in context with name
func StartTransaction(ctx context.Context, name string) (any, context.Context) {
	return instance.StartTransaction(ctx, name)
}

// EndTransaction ends the transaction
func EndTransaction(transaction any) {
	instance.EndTransaction(transaction)
}

// StartWebRequest sets a web request config inside transaction
func StartWebRequest(ctx context.Context, header http.Header, path string, method string) (any, context.Context) {
	return instance.StartWebRequest(ctx, header, path, method)
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
