package colibri_otel

import (
	"context"
	"os"
	"strings"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	colibrimonitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"go.nhat.io/otelsql"
	"go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
)

// splitAndTrim splits s by sep and trims spaces on each part, ignoring empty parts.
func splitAndTrim(s string, sep string) []string {
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

type MonitoringOpenTelemetry struct {
	tracerProvider trace.TracerProvider
	tracer         trace.Tracer
}

func StartOpenTelemetryMonitoring() colibrimonitoringbase.Monitoring {
	ctx := context.Background()

	var appName string
	if otelSrvName := os.Getenv("OTEL_SERVICE_NAME"); otelSrvName != "" {
		appName = otelSrvName
	} else {
		appName = config.APP_NAME
	}

	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(config.OTEL_EXPORTER_OTLP_ENDPOINT),
		otlptracehttp.WithInsecure(),
	}

	if config.OTEL_EXPORTER_OTLP_HEADERS != "" {
		headers := map[string]string{}
		pairs := config.OTEL_EXPORTER_OTLP_HEADERS

		for _, part := range splitAndTrim(pairs, ",") {
			kv := splitAndTrim(part, "=")
			if len(kv) == 2 {
				headers[kv[0]] = kv[1]
			}
		}

		if len(headers) > 0 {
			options = append(options, otlptracehttp.WithHeaders(headers))
		}
	}

	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		logging.Fatal(ctx).Msgf("Creating OTLP HTTP exporter: %v", err)
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(appName),
	)

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	tracer := tracerProvider.Tracer(
		"github.com/colibriproject-dev/colibri-sdk-go",
		trace.WithInstrumentationVersion(contrib.Version()),
	)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return &MonitoringOpenTelemetry{tracer: tracer}
}

func (m *MonitoringOpenTelemetry) StartTransaction(ctx context.Context, name string, kind colibrimonitoringbase.SpanKind) (any, context.Context) {
	ctx, span := m.tracer.Start(ctx, name, trace.WithSpanKind(m.kindToOpenTelemetry(kind)))
	return span, ctx
}

func (m *MonitoringOpenTelemetry) EndTransaction(span any) {
	span.(trace.Span).End()
}

func (m *MonitoringOpenTelemetry) StartTransactionSegment(ctx context.Context, name string, attributes map[string]string) any {
	_, span := m.tracer.Start(ctx, name)

	kv := make([]attribute.KeyValue, 0, len(attributes))
	for key, value := range attributes {
		kv = append(kv, attribute.String(key, value))
	}
	span.SetAttributes(kv...)

	return span
}

func (m *MonitoringOpenTelemetry) AddTransactionAttribute(transaction any, key string, value string) {
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
	)
	if err != nil {
		logging.Fatal(context.Background()).Msgf("could not get sql db driver name: %v", err)
	}
	return driverName
}

func (m *MonitoringOpenTelemetry) kindToOpenTelemetry(kind colibrimonitoringbase.SpanKind) trace.SpanKind {
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
