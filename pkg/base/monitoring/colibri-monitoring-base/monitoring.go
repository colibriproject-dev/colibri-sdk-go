package colibri_monitoring_base

import (
	"context"
	"net/http"
)

// Monitoring is a contract to implements all necessary functions
type Monitoring interface {
	StartTransaction(ctx context.Context, name string) (any, context.Context)
	EndTransaction(transaction any)
	StartWebRequest(ctx context.Context, header http.Header, path string, method string) (any, context.Context)
	StartTransactionSegment(ctx context.Context, name string, attributes map[string]string) any
	EndTransactionSegment(segment any)
	GetTransactionInContext(ctx context.Context) any
	NoticeError(transaction any, err error)
	GetSQLDBDriverName() string
}
