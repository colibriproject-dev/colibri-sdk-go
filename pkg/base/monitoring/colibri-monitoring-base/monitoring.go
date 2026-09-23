package colibri_monitoring_base

import (
	"context"
)

type SpanKind string

const (
	SpanKindInternal SpanKind = "internal"
	SpanKindClient   SpanKind = "client"
	SpanKindServer   SpanKind = "server"
	SpanKindProducer SpanKind = "producer"
	SpanKindConsumer SpanKind = "consumer"
)

// Counter is a monotonically increasing instrument.
type Counter interface {
	// Deprecated: use AddAttrs with a reused Attrs. The map is converted on every call.
	Add(ctx context.Context, value int64, attributes map[string]string)
	AddAttrs(ctx context.Context, value int64, attrs Attrs)
}

// HistogramRecorder records a distribution of values.
type HistogramRecorder interface {
	// Deprecated: use RecordAttrs with a reused Attrs. The map is converted on every call.
	Record(ctx context.Context, value float64, attributes map[string]string)
	RecordAttrs(ctx context.Context, value float64, attrs Attrs)
}

// GaugeRecorder records the current value of a measurement.
type GaugeRecorder interface {
	// Deprecated: use RecordAttrs with a reused Attrs. The map is converted on every call.
	Record(ctx context.Context, value float64, attributes map[string]string)
	RecordAttrs(ctx context.Context, value float64, attrs Attrs)
}

// Observation is a single value reported by an ObservableGauge callback.
type Observation struct {
	Value      float64
	Attributes Attrs
}

// Registration is the handle of an ObservableGauge callback. Unregister stops the
// callback from being invoked on later collections.
type Registration interface {
	Unregister() error
}

// Monitoring is a contract to implement all necessary functions
type Monitoring interface {
	StartTransaction(ctx context.Context, name string, kind SpanKind) (any, context.Context)
	EndTransaction(transaction any)
	StartTransactionSegment(ctx context.Context, name string, attributes map[string]string) any
	AddTransactionAttribute(transaction any, key, value string)
	EndTransactionSegment(segment any)
	GetTransactionInContext(ctx context.Context) any
	NoticeError(transaction any, err error)
	GetSQLDBDriverName() string

	Counter(name, description, unit string) Counter
	Histogram(name, description, unit string) HistogramRecorder
	Gauge(name, description, unit string) GaugeRecorder
	ObservableGauge(name, description, unit string, callback func(context.Context) []Observation) Registration

	Close()
}
