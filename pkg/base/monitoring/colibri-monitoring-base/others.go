package colibri_monitoring_base

import (
	"context"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
)

type others struct{}

func NewOthers() Monitoring {
	return &others{}
}

func (m *others) StartTransaction(ctx context.Context, name string, kind SpanKind) (any, context.Context) {
	logging.Debug(ctx).Msgf("Starting transaction %s Monitoring with name %s", kind, name)
	return nil, ctx
}

func (m *others) EndTransaction(_ any) {
	logging.Debug(context.Background()).Msg("Ending transaction Monitoring")
}

func (m *others) StartTransactionSegment(ctx context.Context, name string, _ map[string]string) any {
	logging.Debug(ctx).Msgf("Starting transaction segment Monitoring with name %s", name)
	return nil
}

func (m *others) AddTransactionAttribute(_ any, key, value string) {
	logging.Debug(context.Background()).
		AddParam("key", key).
		AddParam("value", value).
		Msg("Adding transaction attribute Monitoring")
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

func (m *others) Counter(name, _, _ string) Counter {
	logging.Debug(context.Background()).Msgf("Creating noop counter %s", name)
	return &noopCounter{}
}

func (m *others) Histogram(name, _, _ string) HistogramRecorder {
	logging.Debug(context.Background()).Msgf("Creating noop histogram %s", name)
	return &noopHistogram{}
}

func (m *others) Gauge(name, _, _ string) GaugeRecorder {
	logging.Debug(context.Background()).Msgf("Creating noop gauge %s", name)
	return &noopGauge{}
}

func (m *others) Close() {
	logging.Debug(context.Background()).Msg("Closing noop monitoring")
}

type noopCounter struct{}

func (c *noopCounter) Add(_ context.Context, value int64, attributes map[string]string) {
	logging.Debug(context.Background()).Msgf("Add counter: value[%d];attributes[%v]", value, attributes)
}

type noopHistogram struct{}

func (h *noopHistogram) Record(_ context.Context, value float64, attributes map[string]string) {
	logging.Debug(context.Background()).Msgf("Record histogram: value[%f];attributes[%v]", value, attributes)
}

type noopGauge struct{}

func (g *noopGauge) Record(_ context.Context, value float64, attributes map[string]string) {
	logging.Debug(context.Background()).Msgf("Record gauge: value[%f];attributes[%v]", value, attributes)
}
