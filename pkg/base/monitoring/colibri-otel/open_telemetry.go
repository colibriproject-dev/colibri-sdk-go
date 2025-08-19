package colibri_otel

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	colibri_monitoring_base "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
	"go.nhat.io/otelsql"
	"go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
)

type MonitoringOpenTelemetry struct {
	tracerProvider trace.TracerProvider
	tracer         trace.Tracer
}

func newResource() *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(config.APP_NAME),
		semconv.ServiceVersion(config.APP_VERSION),
	)
}

func StartOpenTelemetryMonitoring() colibri_monitoring_base.Monitoring {
	client := otlptracehttp.NewClient()
	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		logging.Fatal(context.Background()).Msgf("Creating OTLP trace exporter: %v", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(newResource()),
	)
	otel.SetTracerProvider(tracerProvider)

	tracer := tracerProvider.Tracer(
		"github.com/colibriproject-dev/colibri-sdk-go",
		trace.WithInstrumentationVersion(contrib.SemVersion()),
	)

	return &MonitoringOpenTelemetry{tracer: tracer}
}

func (m *MonitoringOpenTelemetry) StartTransaction(ctx context.Context, name string) (any, context.Context) {
	ctx, span := m.tracer.Start(ctx, name)
	return span, ctx
}

func (m *MonitoringOpenTelemetry) EndTransaction(span any) {
	span.(trace.Span).End()
}

func (m *MonitoringOpenTelemetry) StartWebRequest(ctx context.Context, header http.Header, path string, method string) (any, context.Context) {
	attrs := []attribute.KeyValue{
		semconv.HTTPRequestMethodKey.String(method),
		//semconv.HTTPRequestContentLengthKey.String(header.Get("Content-Length")),
		semconv.URLSchemeKey.String(header.Get("X-Protocol")),
		//semconv.HTTPTargetKey.String(header.Get("X-Request-URI")),
		semconv.URLPathKey.String(path),
		semconv.UserAgentOriginal(header.Get("User-Agent")),
		semconv.HostNameKey.String(header.Get("Host")),
		semconv.NetworkTransportTCP,
	}

	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindServer),
	}
	ctx, span := m.tracer.Start(ctx, fmt.Sprintf("%s %s", method, path), opts...)

	return span, ctx
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
