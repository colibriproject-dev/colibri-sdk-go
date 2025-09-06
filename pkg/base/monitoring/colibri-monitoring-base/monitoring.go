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

// Monitoring is a contract to implement all necessary functions
type Monitoring interface {
	StartTransaction(ctx context.Context, name string, kind SpanKind) (any, context.Context)
	EndTransaction(transaction any)
	StartTransactionSegment(ctx context.Context, name string, attributes map[string]string) any
	AddTransactionAttribute(transaction any, key string, value string)
	EndTransactionSegment(segment any)
	GetTransactionInContext(ctx context.Context) any
	NoticeError(transaction any, err error)
	GetSQLDBDriverName() string
}
