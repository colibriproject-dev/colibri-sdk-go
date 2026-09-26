package monitoring

import (
	"context"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	colibriotel "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-otel"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
)

var instance colibrimonitoringbase.Monitoring

// Initialize loads the Monitoring settings according to the configured environment.
//
// It is idempotent: a second call is ignored. Rebuilding the providers would register a
// second Prometheus collector on the default registry, and since the OTEL collector
// describes no metrics up front the registry accepts the duplicate silently — the damage
// only shows up later, as a /metrics scrape failing on duplicate metric families.
func Initialize() {
	if instance != nil {
		logging.Warn(context.Background()).Msg("Monitoring already initialized, skipping")
		return
	}

	signals := colibriotel.Signals{
		Tracing:           UseTracing(),
		OTLPMetrics:       UseOTLPMetrics(),
		PrometheusMetrics: UsePrometheusMetrics(),
	}

	if !signals.AnyEnabled() {
		instance = colibrimonitoringbase.NewOthers()
		return
	}

	instance = colibriotel.StartOpenTelemetryMonitoring(signals)
	observer.Attach(instance.(observer.Observer))
}

// UseTracing returns true if the trace signal is enabled. Traces need a collector, so the
// OTLP endpoint is what turns them on; OTEL_TRACES_ENABLED only turns them off.
func UseTracing() bool {
	return config.OTEL_TRACES_ENABLED && config.OTEL_EXPORTER_OTLP_ENDPOINT != ""
}

// UseOTLPMetrics returns true if metrics are pushed to an OTLP collector.
func UseOTLPMetrics() bool {
	if !config.OTEL_METRICS_ENABLED {
		return false
	}

	return config.OTEL_EXPORTER_OTLP_METRICS_ENDPOINT != "" || config.OTEL_EXPORTER_OTLP_ENDPOINT != ""
}

// UsePrometheusMetrics returns true if metrics are exposed on the /metrics route through
// the Prometheus registry. Enabled by default, so metrics are scrapable with no collector
// and no configuration.
func UsePrometheusMetrics() bool {
	return config.OTEL_METRICS_ENABLED && config.OTEL_METRICS_PROMETHEUS_ENABLED
}

// UseMetrics returns true if the metric signal is enabled through any reader.
func UseMetrics() bool {
	return UseOTLPMetrics() || UsePrometheusMetrics()
}

// UseOTELMonitoring returns true if OTEL monitoring is enabled.
//
// Deprecated: traces and metrics are now independent. Use UseTracing or UseMetrics, and
// pick the one matching what the caller instruments — this reports true when either is on.
func UseOTELMonitoring() bool {
	return UseTracing() || UseMetrics()
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
func Histogram(name, description, unit string) colibrimonitoringbase.HistogramRecorder {
	return instance.Histogram(name, description, unit)
}

// Gauge returns a named gauge instrument for recording current values.
func Gauge(name, description, unit string) colibrimonitoringbase.GaugeRecorder {
	return instance.Gauge(name, description, unit)
}

// ObservableGauge registers a callback invoked on every metric collection, for values that
// are sampled rather than pushed — connection pool size, queue depth, cache entries.
// Unregister the returned Registration when the observed resource goes away.
func ObservableGauge(
	name, description, unit string,
	callback func(context.Context) []colibrimonitoringbase.Observation,
) colibrimonitoringbase.Registration {
	return instance.ObservableGauge(name, description, unit, callback)
}
