package colibri_monitoring_base

import (
	"context"
	"net/http"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

type others struct {
}

func NewOthers() Monitoring {
	return &others{}
}

func (m *others) StartTransaction(ctx context.Context, name string) (any, context.Context) {
	logging.Debug(ctx).Msgf("Starting transaction Monitoring with name %s", name)
	return nil, ctx
}

func (m *others) EndTransaction(_ any) {
	logging.Debug(context.Background()).Msg("Ending transaction Monitoring")
}

func (m *others) StartWebRequest(ctx context.Context, _ http.Header, path string, _ string) (any, context.Context) {
	logging.Debug(ctx).Msgf("Start web request in transaction with path %s", path)
	return nil, ctx
}

func (m *others) StartTransactionSegment(ctx context.Context, name string, _ map[string]string) any {
	logging.Debug(ctx).Msgf("Starting transaction segment Monitoring with name %s", name)
	return nil
}

func (m *others) EndTransactionSegment(_ any) {
	logging.Debug(context.Background()).Msg("Ending transaction segment Monitoring")
}

func (m *others) GetTransactionInContext(_ context.Context) any {
	logging.Debug(context.Background()).Msg("Getting transaction in context")
	return nil
}

func (m *others) NoticeError(_ any, err error) {
	logging.Debug(context.Background()).Msgf("Warning error %v", err)
}

func (m *others) GetSQLDBDriverName() string {
	return "postgres"
}
