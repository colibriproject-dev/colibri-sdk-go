package colibri_otel

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"go.nhat.io/otelsql"
	"go.opentelemetry.io/contrib"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

// instrumentationName is the scope of the tracer and meter the SDK records through.
const instrumentationName = "github.com/colibriproject-dev/colibri-sdk-go"

// normalizeEndpoint strips the scheme (http:// or https://) and any trailing path from
// an OTLP endpoint, leaving just host:port as expected by WithEndpoint options.
func normalizeEndpoint(endpoint string) string {
	for _, scheme := range []string{"https://", "http://"} {
		if after, ok := strings.CutPrefix(endpoint, scheme); ok {
			endpoint = after
			break
		}
	}
	// Drop any path component (e.g. /v1/traces → keep only host:port).
	if idx := strings.Index(endpoint, "/"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	return endpoint
}

// isInsecureEndpoint reports whether the OTLP exporter should send without TLS.
// An explicit https:// scheme enables TLS; anything else (http:// or no scheme)
// stays insecure for backwards compatibility with the previous hardcoded behavior.
func isInsecureEndpoint(endpoint string) bool {
	return !strings.HasPrefix(endpoint, "https://")
}

// splitAndTrim splits s by sep and trims spaces on each part, ignoring empty parts.
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			res = append(res, p)
		}
	}
	return res
}

// parseHeaders parses a comma-separated "key=value" header string.
func parseHeaders(raw string) map[string]string {
	headers := map[string]string{}
	for _, part := range splitAndTrim(raw, ",") {
		kv := splitAndTrim(part, "=")
		if len(kv) == 2 {
			headers[kv[0]] = kv[1]
		}
	}
	return headers
}

// attrsFromMap converts a string map to OTEL attribute slice.
func attrsFromMap(m map[string]string) []attribute.KeyValue {
	kv := make([]attribute.KeyValue, 0, len(m))
	for k, v := range m {
		kv = append(kv, attribute.String(k, v))
	}
	return kv
}

// otelAttrs is the OTEL representation of a colibri Attrs, cached inside the Attrs itself.
// The option slices are pre-built so that recording a measurement passes an existing slice
// to the variadic instrument call instead of allocating one per measurement.
type otelAttrs struct {
	addOpts     []metric.AddOption
	recordOpts  []metric.RecordOption
	observeOpts []metric.ObserveOption
}

// buildOtelAttrs converts colibri attribute pairs into the cached OTEL options. It runs at
// most once per Attrs.
func buildOtelAttrs(pairs []colibrimonitoringbase.Attr) any {
	kv := make([]attribute.KeyValue, len(pairs))
	for i, pair := range pairs {
		kv[i] = attribute.String(pair.Key, pair.Value)
	}

	set := attribute.NewSet(kv...)
	option := metric.WithAttributeSet(set)

	return &otelAttrs{
		addOpts:     []metric.AddOption{option},
		recordOpts:  []metric.RecordOption{option},
		observeOpts: []metric.ObserveOption{option},
	}
}

// cachedAttrs returns the OTEL options of an Attrs, building them on first use.
func cachedAttrs(attrs colibrimonitoringbase.Attrs) *otelAttrs {
	return attrs.Cached(buildOtelAttrs).(*otelAttrs)
}

// Signals selects which OTEL signals a monitoring instance exports. Each is independent:
// a service can scrape Prometheus metrics with no collector configured, or export traces
// only. A signal that is off gets a noop provider, so the instrumentation calls scattered
// through the SDK stay valid and simply do nothing.
type Signals struct {
	Tracing           bool
	OTLPMetrics       bool
	PrometheusMetrics bool

	// PrometheusRegisterer receives the Prometheus collector. Defaults to
	// prometheus.DefaultRegisterer, which is what the /metrics route serves.
	PrometheusRegisterer prometheus.Registerer
}

// AnyEnabled reports whether at least one signal is on.
func (s Signals) AnyEnabled() bool {
	return s.Tracing || s.OTLPMetrics || s.PrometheusMetrics
}

// AnyMetrics reports whether at least one metric reader is on.
func (s Signals) AnyMetrics() bool {
	return s.OTLPMetrics || s.PrometheusMetrics
}

type MonitoringOpenTelemetry struct {
	// The SDK providers are nil when their signal is disabled; the tracer and meter are
	// always usable, backed by a noop provider in that case.
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         trace.Tracer
	meter          metric.Meter

	// The wrappers, not the bare instruments, are cached: a cache hit then costs nothing,
	// where returning a fresh wrapper allocated once per lookup.
	countersMu sync.Mutex
	counters   map[string]*otelCounter

	histogramsMu sync.Mutex
	histograms   map[string]*otelHistogram

	gaugesMu sync.Mutex
	gauges   map[string]*otelGauge
}

// StartOpenTelemetryMonitoring builds a monitoring instance exporting the enabled signals.
func StartOpenTelemetryMonitoring(signals Signals) colibrimonitoringbase.Monitoring {
	ctx := context.Background()
	res := buildResource(ctx)
	parsedHeaders := parseHeaders(config.OTEL_EXPORTER_OTLP_HEADERS)

	tracerProvider := buildTracerProvider(ctx, signals, res, parsedHeaders)
	meterProvider := buildMeterProvider(ctx, signals, res, parsedHeaders)

	// ── Global providers ──────────────────────────────────────────────────────
	// Instrumentation libraries (otelfiber, otelhttp, otelsql, otelruntime) read the
	// globals, so a disabled signal has to install a noop provider rather than leave the
	// previous one in place.
	if tracerProvider != nil {
		otel.SetTracerProvider(tracerProvider)
	} else {
		otel.SetTracerProvider(nooptrace.NewTracerProvider())
	}

	if meterProvider != nil {
		otel.SetMeterProvider(meterProvider)
	} else {
		otel.SetMeterProvider(noopmetric.NewMeterProvider())
	}

	// ── Runtime metrics ───────────────────────────────────────────────────────
	// Non-critical: if runtime metrics fail to start (e.g. already running), continue.
	if signals.AnyMetrics() {
		_ = otelruntime.Start(otelruntime.WithMinimumReadMemStatsInterval(time.Second))
	}

	// ── Propagators ───────────────────────────────────────────────────────────
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &MonitoringOpenTelemetry{
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
		tracer: otel.GetTracerProvider().Tracer(instrumentationName,
			trace.WithInstrumentationVersion(contrib.Version())),
		meter: otel.GetMeterProvider().Meter(instrumentationName,
			metric.WithInstrumentationVersion(contrib.Version())),
		counters:   make(map[string]*otelCounter),
		histograms: make(map[string]*otelHistogram),
		gauges:     make(map[string]*otelGauge),
	}
}

// NewWithMeterProvider builds a monitoring instance recording metrics into the given
// provider, with tracing disabled. It does not touch the global providers. It exists so
// tests can read what the SDK records through a ManualReader — see the monitoringtest
// package.
func NewWithMeterProvider(meterProvider *sdkmetric.MeterProvider) *MonitoringOpenTelemetry {
	return &MonitoringOpenTelemetry{
		meterProvider: meterProvider,
		tracer:        nooptrace.NewTracerProvider().Tracer(instrumentationName),
		meter:         meterProvider.Meter(instrumentationName),
		counters:      make(map[string]*otelCounter),
		histograms:    make(map[string]*otelHistogram),
		gauges:        make(map[string]*otelGauge),
	}
}

// buildResource assembles the resource attributes shared by every signal.
func buildResource(ctx context.Context) *resource.Resource {
	appName := os.Getenv("OTEL_SERVICE_NAME")
	if appName == "" {
		appName = config.APP_NAME
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(appName),
			semconv.ServiceVersionKey.String(config.VERSION),
			semconv.ServiceInstanceIDKey.String(uuid.New().String()),
		),
	)
	if err != nil {
		logging.Fatal(ctx).Msgf("Building OTEL resource: %v", err)
	}

	return res
}

// buildTracerProvider returns the SDK tracer provider, or nil when tracing is disabled.
func buildTracerProvider(
	ctx context.Context,
	signals Signals,
	res *resource.Resource,
	headers map[string]string,
) *sdktrace.TracerProvider {
	if !signals.Tracing {
		return nil
	}

	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(normalizeEndpoint(config.OTEL_EXPORTER_OTLP_ENDPOINT)),
	}
	if isInsecureEndpoint(config.OTEL_EXPORTER_OTLP_ENDPOINT) {
		options = append(options, otlptracehttp.WithInsecure())
	}
	if len(headers) > 0 {
		options = append(options, otlptracehttp.WithHeaders(headers))
	}

	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		logging.Fatal(ctx).Msgf("Creating OTLP trace exporter: %v", err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exporter)),
	)
}

// buildMeterProvider returns the SDK meter provider fed by every enabled reader, or nil
// when no metric reader is enabled.
func buildMeterProvider(
	ctx context.Context,
	signals Signals,
	res *resource.Resource,
	headers map[string]string,
) *sdkmetric.MeterProvider {
	readers := make([]sdkmetric.Reader, 0, 2)

	if signals.OTLPMetrics {
		readers = append(readers, sdkmetric.NewPeriodicReader(buildOTLPMetricExporter(ctx, headers)))
	}

	if signals.PrometheusMetrics {
		if reader := buildPrometheusReader(ctx, signals.PrometheusRegisterer); reader != nil {
			readers = append(readers, reader)
		}
	}

	if len(readers) == 0 {
		return nil
	}

	options := make([]sdkmetric.Option, 0, len(readers)+1)
	options = append(options, sdkmetric.WithResource(res))
	for _, reader := range readers {
		options = append(options, sdkmetric.WithReader(reader))
	}

	return sdkmetric.NewMeterProvider(options...)
}

// buildOTLPMetricExporter creates the exporter pushing metrics to the collector.
func buildOTLPMetricExporter(ctx context.Context, headers map[string]string) sdkmetric.Exporter {
	endpoint := config.OTEL_EXPORTER_OTLP_METRICS_ENDPOINT
	if endpoint == "" {
		endpoint = config.OTEL_EXPORTER_OTLP_ENDPOINT
	}

	options := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(normalizeEndpoint(endpoint)),
	}
	if isInsecureEndpoint(endpoint) {
		options = append(options, otlpmetrichttp.WithInsecure())
	}
	if len(headers) > 0 {
		options = append(options, otlpmetrichttp.WithHeaders(headers))
	}

	exporter, err := otlpmetrichttp.New(ctx, options...)
	if err != nil {
		logging.Fatal(ctx).Msgf("Creating OTLP metric exporter: %v", err)
	}

	return exporter
}

// buildPrometheusReader registers the OTEL collector on the Prometheus registry serving
// the /metrics route, and returns it as a second reader on the meter provider. It returns
// nil, after a warning, when the collector cannot be registered: exposing no metrics is a
// better outcome than failing to boot.
//
// Note that a repeated registration does not surface as AlreadyRegisteredError. The OTEL
// collector describes no metrics up front, which makes it an unchecked collector, and the
// Prometheus registry accepts those without deduplicating — the duplicate only shows up
// later, as a failing scrape. Idempotency is enforced by monitoring.Initialize instead;
// the error is handled here for the case where a registry does reject the collector.
func buildPrometheusReader(ctx context.Context, registerer prometheus.Registerer) sdkmetric.Reader {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	exporter, err := otelprometheus.New(otelprometheus.WithRegisterer(registerer))
	if err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			logging.Warn(ctx).Msg("Prometheus metrics collector already registered, skipping")
			return nil
		}

		logging.Warn(ctx).Msgf("Creating Prometheus metrics exporter: %v", err)
		return nil
	}

	return exporter
}

func (m *MonitoringOpenTelemetry) StartTransaction(ctx context.Context, name string, kind colibrimonitoringbase.SpanKind) (any, context.Context) {
	ctx, span := m.tracer.Start(ctx, name, trace.WithSpanKind(kindToOpenTelemetry(kind)))
	return span, ctx
}

func (m *MonitoringOpenTelemetry) EndTransaction(span any) {
	span.(trace.Span).End()
}

func (m *MonitoringOpenTelemetry) StartTransactionSegment(ctx context.Context, name string, attributes map[string]string) any {
	_, span := m.tracer.Start(ctx, name)
	span.SetAttributes(attrsFromMap(attributes)...)
	return span
}

func (m *MonitoringOpenTelemetry) AddTransactionAttribute(transaction any, key, value string) {
	transaction.(trace.Span).SetAttributes(attribute.String(key, value))
}

func (m *MonitoringOpenTelemetry) EndTransactionSegment(segment any) {
	segment.(trace.Span).End()
}

func (m *MonitoringOpenTelemetry) GetTransactionInContext(ctx context.Context) any {
	return trace.SpanFromContext(ctx)
}

func (m *MonitoringOpenTelemetry) NoticeError(transaction any, err error) {
	transaction.(trace.Span).RecordError(err)
	transaction.(trace.Span).SetStatus(codes.Error, err.Error())
}

func (m *MonitoringOpenTelemetry) GetSQLDBDriverName() string {
	driverName, err := otelsql.Register("postgres",
		otelsql.AllowRoot(),
		otelsql.TraceQueryWithoutArgs(),
		otelsql.TraceRowsClose(),
		otelsql.TraceRowsAffected(),
		otelsql.WithDatabaseName(os.Getenv(config.SQL_DB_NAME)),
		otelsql.WithSystem(semconv.DBSystemNamePostgreSQL),
		otelsql.WithMeterProvider(m.sqlMeterProvider()),
	)
	if err != nil {
		logging.Fatal(context.Background()).Msgf("could not get sql db driver name: %v", err)
	}
	return driverName
}

// sqlMeterProvider returns the provider the SQL instrumentation should report to. The
// driver is registered even with metrics disabled, so it needs a noop provider then.
func (m *MonitoringOpenTelemetry) sqlMeterProvider() metric.MeterProvider {
	if m.meterProvider == nil {
		return noopmetric.NewMeterProvider()
	}

	return m.meterProvider
}

// ── Metrics ───────────────────────────────────────────────────────────────────

func (m *MonitoringOpenTelemetry) Counter(name, description, unit string) colibrimonitoringbase.Counter {
	m.countersMu.Lock()
	defer m.countersMu.Unlock()
	if c, ok := m.counters[name]; ok {
		return c
	}
	c, err := m.meter.Int64Counter(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		logging.Warn(context.Background()).Msgf("Creating counter %s: %v", name, err)
		return colibrimonitoringbase.NoopCounter()
	}
	counter := &otelCounter{instrument: c}
	m.counters[name] = counter
	return counter
}

func (m *MonitoringOpenTelemetry) Histogram(name, description, unit string) colibrimonitoringbase.HistogramRecorder {
	m.histogramsMu.Lock()
	defer m.histogramsMu.Unlock()
	if h, ok := m.histograms[name]; ok {
		return h
	}
	h, err := m.meter.Float64Histogram(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		logging.Warn(context.Background()).Msgf("Creating histogram %s: %v", name, err)
		return colibrimonitoringbase.NoopHistogram()
	}
	histogram := &otelHistogram{instrument: h}
	m.histograms[name] = histogram
	return histogram
}

func (m *MonitoringOpenTelemetry) Gauge(name, description, unit string) colibrimonitoringbase.GaugeRecorder {
	m.gaugesMu.Lock()
	defer m.gaugesMu.Unlock()
	if g, ok := m.gauges[name]; ok {
		return g
	}
	g, err := m.meter.Float64Gauge(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		logging.Warn(context.Background()).Msgf("Creating gauge %s: %v", name, err)
		return colibrimonitoringbase.NoopGauge()
	}
	gauge := &otelGauge{instrument: g}
	m.gauges[name] = gauge
	return gauge
}

// ObservableGauge registers a callback that reports the current value of a measurement on
// every collection, for values that are sampled rather than pushed — pool sizes, queue
// depth, cache entries.
//
// Unlike the synchronous instruments, the result is not cached by name: each call
// registers its own callback, and the caller owns its lifetime through the returned
// Registration. Register a given name once.
func (m *MonitoringOpenTelemetry) ObservableGauge(
	name, description, unit string,
	callback func(context.Context) []colibrimonitoringbase.Observation,
) colibrimonitoringbase.Registration {
	gauge, err := m.meter.Float64ObservableGauge(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		logging.Warn(context.Background()).Msgf("Creating observable gauge %s: %v", name, err)
		return colibrimonitoringbase.NoopRegistration()
	}

	registration, err := m.meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		for _, observation := range callback(ctx) {
			observer.ObserveFloat64(gauge, observation.Value, cachedAttrs(observation.Attributes).observeOpts...)
		}
		return nil
	}, gauge)
	if err != nil {
		logging.Warn(context.Background()).Msgf("Registering observable gauge callback %s: %v", name, err)
		return colibrimonitoringbase.NoopRegistration()
	}

	return registration
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

func (m *MonitoringOpenTelemetry) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if m.tracerProvider != nil {
		if err := m.tracerProvider.Shutdown(ctx); err != nil {
			logging.Warn(ctx).Msgf("OTEL tracer provider shutdown: %v", err)
		}
	}
	if m.meterProvider != nil {
		if err := m.meterProvider.Shutdown(ctx); err != nil {
			logging.Warn(ctx).Msgf("OTEL meter provider shutdown: %v", err)
		}
	}
}

// ── Instrument wrappers ───────────────────────────────────────────────────────

type otelCounter struct{ instrument metric.Int64Counter }

// Deprecated: use AddAttrs. The map is converted to OTEL attributes on every call.
func (c *otelCounter) Add(ctx context.Context, value int64, attributes map[string]string) {
	c.instrument.Add(ctx, value, metric.WithAttributes(attrsFromMap(attributes)...))
}

func (c *otelCounter) AddAttrs(ctx context.Context, value int64, attrs colibrimonitoringbase.Attrs) {
	c.instrument.Add(ctx, value, cachedAttrs(attrs).addOpts...)
}

type otelHistogram struct{ instrument metric.Float64Histogram }

// Deprecated: use RecordAttrs. The map is converted to OTEL attributes on every call.
func (h *otelHistogram) Record(ctx context.Context, value float64, attributes map[string]string) {
	h.instrument.Record(ctx, value, metric.WithAttributes(attrsFromMap(attributes)...))
}

func (h *otelHistogram) RecordAttrs(ctx context.Context, value float64, attrs colibrimonitoringbase.Attrs) {
	h.instrument.Record(ctx, value, cachedAttrs(attrs).recordOpts...)
}

type otelGauge struct{ instrument metric.Float64Gauge }

// Deprecated: use RecordAttrs. The map is converted to OTEL attributes on every call.
func (g *otelGauge) Record(ctx context.Context, value float64, attributes map[string]string) {
	g.instrument.Record(ctx, value, metric.WithAttributes(attrsFromMap(attributes)...))
}

func (g *otelGauge) RecordAttrs(ctx context.Context, value float64, attrs colibrimonitoringbase.Attrs) {
	g.instrument.Record(ctx, value, cachedAttrs(attrs).recordOpts...)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func kindToOpenTelemetry(kind colibrimonitoringbase.SpanKind) trace.SpanKind {
	switch kind {
	case colibrimonitoringbase.SpanKindClient:
		return trace.SpanKindClient
	case colibrimonitoringbase.SpanKindServer:
		return trace.SpanKindServer
	case colibrimonitoringbase.SpanKindProducer:
		return trace.SpanKindProducer
	case colibrimonitoringbase.SpanKindConsumer:
		return trace.SpanKindConsumer
	case colibrimonitoringbase.SpanKindInternal:
		return trace.SpanKindInternal
	default:
		return trace.SpanKindUnspecified
	}
}
