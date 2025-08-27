package colibri_monitoring_base

import (
	"context"
)

// Monitoring is a contract to implement all necessary functions
type Monitoring interface {
	StartTransaction(ctx context.Context, name string) (any, context.Context)
	EndTransaction(transaction any)
	StartTransactionSegment(ctx context.Context, name string, attributes map[string]string) any
	AddTransactionAttribute(transaction any, key string, value string)
	EndTransactionSegment(segment any)
	GetTransactionInContext(ctx context.Context) any
	NoticeError(transaction any, err error)
	GetSQLDBDriverName() string
}
