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

func (m *others) ObservableGauge(name, _, _ string, _ func(context.Context) []Observation) Registration {
	logging.Debug(context.Background()).Msgf("Creating noop observable gauge %s", name)
	return &noopRegistration{}
}

func (m *others) Close() {
	logging.Debug(context.Background()).Msg("Closing noop monitoring")
}

// The instruments below are deliberately silent. They sit on the caller's recording path
// and used to format a log message per measurement, which costs allocations on every call
// even with the log level disabled. Creation is still logged, above.

type noopCounter struct{}

func (c *noopCounter) Add(_ context.Context, _ int64, _ map[string]string) {}

func (c *noopCounter) AddAttrs(_ context.Context, _ int64, _ Attrs) {}

type noopHistogram struct{}

func (h *noopHistogram) Record(_ context.Context, _ float64, _ map[string]string) {}

func (h *noopHistogram) RecordAttrs(_ context.Context, _ float64, _ Attrs) {}

type noopGauge struct{}

func (g *noopGauge) Record(_ context.Context, _ float64, _ map[string]string) {}

func (g *noopGauge) RecordAttrs(_ context.Context, _ float64, _ Attrs) {}

type noopRegistration struct{}

func (r *noopRegistration) Unregister() error { return nil }

// The constructors below let a real Monitoring implementation degrade to an inert
// instrument when the provider refuses to create one, instead of returning nil and
// turning a bad metric name into a panic on the caller's recording path.

// NoopCounter returns a Counter that discards every measurement.
func NoopCounter() Counter { return &noopCounter{} }

// NoopHistogram returns a HistogramRecorder that discards every measurement.
func NoopHistogram() HistogramRecorder { return &noopHistogram{} }

// NoopGauge returns a GaugeRecorder that discards every measurement.
func NoopGauge() GaugeRecorder { return &noopGauge{} }

// NoopRegistration returns a Registration whose callback is never invoked.
func NoopRegistration() Registration { return &noopRegistration{} }
