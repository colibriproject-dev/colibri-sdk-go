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
	Add(ctx context.Context, value int64, attributes map[string]string)
}

// HistogramRecorder records a distribution of values.
type HistogramRecorder interface {
	Record(ctx context.Context, value float64, attributes map[string]string)
}

// GaugeRecorder records the current value of a measurement.
type GaugeRecorder interface {
	Record(ctx context.Context, value float64, attributes map[string]string)
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

	Close()
}
