package colibri_otel

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"github.com/google/uuid"
	"go.nhat.io/otelsql"
	"go.opentelemetry.io/contrib"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

// normalizeEndpoint strips the scheme (http:// or https://) and any trailing path from
// an OTLP endpoint, leaving just host:port as expected by WithEndpoint options.
func normalizeEndpoint(endpoint string) string {
	for _, scheme := range []string{"https://", "http://"} {
		if strings.HasPrefix(endpoint, scheme) {
			endpoint = strings.TrimPrefix(endpoint, scheme)
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

type MonitoringOpenTelemetry struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         trace.Tracer
	meter          metric.Meter

	countersMu sync.Mutex
	counters   map[string]metric.Int64Counter

	histogramsMu sync.Mutex
	histograms   map[string]metric.Float64Histogram

	gaugesMu sync.Mutex
	gauges   map[string]metric.Float64Gauge
}

func StartOpenTelemetryMonitoring() colibrimonitoringbase.Monitoring {
	ctx := context.Background()

	appName := os.Getenv("OTEL_SERVICE_NAME")
	if appName == "" {
		appName = config.APP_NAME
	}

	parsedHeaders := parseHeaders(config.OTEL_EXPORTER_OTLP_HEADERS)

	// ── Shared resource ──────────────────────────────────────────────────────
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

	// ── Trace exporter + provider ─────────────────────────────────────────────
	traceOptions := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(normalizeEndpoint(config.OTEL_EXPORTER_OTLP_ENDPOINT)),
	}
	if isInsecureEndpoint(config.OTEL_EXPORTER_OTLP_ENDPOINT) {
		traceOptions = append(traceOptions, otlptracehttp.WithInsecure())
	}
	if len(parsedHeaders) > 0 {
		traceOptions = append(traceOptions, otlptracehttp.WithHeaders(parsedHeaders))
	}

	traceExporter, err := otlptracehttp.New(ctx, traceOptions...)
	if err != nil {
		logging.Fatal(ctx).Msgf("Creating OTLP trace exporter: %v", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(traceExporter)),
	)
	otel.SetTracerProvider(tracerProvider)

	// ── Metric exporter + provider ────────────────────────────────────────────
	metricsEndpoint := config.OTEL_EXPORTER_OTLP_METRICS_ENDPOINT
	if metricsEndpoint == "" {
		metricsEndpoint = config.OTEL_EXPORTER_OTLP_ENDPOINT
	}

	metricOptions := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(normalizeEndpoint(metricsEndpoint)),
	}
	if isInsecureEndpoint(metricsEndpoint) {
		metricOptions = append(metricOptions, otlpmetrichttp.WithInsecure())
	}
	if len(parsedHeaders) > 0 {
		metricOptions = append(metricOptions, otlpmetrichttp.WithHeaders(parsedHeaders))
	}

	metricExporter, err := otlpmetrichttp.New(ctx, metricOptions...)
	if err != nil {
		logging.Fatal(ctx).Msgf("Creating OTLP metric exporter: %v", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)
	otel.SetMeterProvider(meterProvider)

	// ── Runtime metrics ───────────────────────────────────────────────────────
	// Non-critical: if runtime metrics fail to start (e.g. already running), continue.
	_ = otelruntime.Start(otelruntime.WithMinimumReadMemStatsInterval(time.Second))

	// ── Propagators ───────────────────────────────────────────────────────────
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := tracerProvider.Tracer(
		"github.com/colibriproject-dev/colibri-sdk-go",
		trace.WithInstrumentationVersion(contrib.Version()),
	)
	meter := meterProvider.Meter(
		"github.com/colibriproject-dev/colibri-sdk-go",
		metric.WithInstrumentationVersion(contrib.Version()),
	)

	return &MonitoringOpenTelemetry{
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
		tracer:         tracer,
		meter:          meter,
		counters:       make(map[string]metric.Int64Counter),
		histograms:     make(map[string]metric.Float64Histogram),
		gauges:         make(map[string]metric.Float64Gauge),
	}
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
		otelsql.WithMeterProvider(m.meterProvider),
	)
	if err != nil {
		logging.Fatal(context.Background()).Msgf("could not get sql db driver name: %v", err)
	}
	return driverName
}

// ── Metrics ───────────────────────────────────────────────────────────────────

func (m *MonitoringOpenTelemetry) Counter(name, description, unit string) colibrimonitoringbase.Counter {
	m.countersMu.Lock()
	defer m.countersMu.Unlock()
	if c, ok := m.counters[name]; ok {
		return &otelCounter{instrument: c}
	}
	c, err := m.meter.Int64Counter(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		logging.Fatal(context.Background()).Msgf("Creating counter %s: %v", name, err)
	}
	m.counters[name] = c
	return &otelCounter{instrument: c}
}

func (m *MonitoringOpenTelemetry) Histogram(name, description, unit string) colibrimonitoringbase.HistogramRecorder {
	m.histogramsMu.Lock()
	defer m.histogramsMu.Unlock()
	if h, ok := m.histograms[name]; ok {
		return &otelHistogram{instrument: h}
	}
	h, err := m.meter.Float64Histogram(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		logging.Fatal(context.Background()).Msgf("Creating histogram %s: %v", name, err)
	}
	m.histograms[name] = h
	return &otelHistogram{instrument: h}
}

func (m *MonitoringOpenTelemetry) Gauge(name, description, unit string) colibrimonitoringbase.GaugeRecorder {
	m.gaugesMu.Lock()
	defer m.gaugesMu.Unlock()
	if g, ok := m.gauges[name]; ok {
		return &otelGauge{instrument: g}
	}
	g, err := m.meter.Float64Gauge(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		logging.Fatal(context.Background()).Msgf("Creating gauge %s: %v", name, err)
	}
	m.gauges[name] = g
	return &otelGauge{instrument: g}
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

func (m *MonitoringOpenTelemetry) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.tracerProvider.Shutdown(ctx); err != nil {
		logging.Warn(ctx).Msgf("OTEL tracer provider shutdown: %v", err)
	}
	if err := m.meterProvider.Shutdown(ctx); err != nil {
		logging.Warn(ctx).Msgf("OTEL meter provider shutdown: %v", err)
	}
}

// ── Instrument wrappers ───────────────────────────────────────────────────────

type otelCounter struct{ instrument metric.Int64Counter }

func (c *otelCounter) Add(ctx context.Context, value int64, attributes map[string]string) {
	c.instrument.Add(ctx, value, metric.WithAttributes(attrsFromMap(attributes)...))
}

type otelHistogram struct{ instrument metric.Float64Histogram }

func (h *otelHistogram) Record(ctx context.Context, value float64, attributes map[string]string) {
	h.instrument.Record(ctx, value, metric.WithAttributes(attrsFromMap(attributes)...))
}

type otelGauge struct{ instrument metric.Float64Gauge }

func (g *otelGauge) Record(ctx context.Context, value float64, attributes map[string]string) {
	g.instrument.Record(ctx, value, metric.WithAttributes(attrsFromMap(attributes)...))
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
