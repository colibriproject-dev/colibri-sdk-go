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
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MonitoringOpenTelemetry struct {
	tracerProvider trace.TracerProvider
	tracer         trace.Tracer
}

func StartOpenTelemetryMonitoring() colibri_monitoring_base.Monitoring {
	ctx := context.Background()
	conn, err := grpc.NewClient(
		config.OTEL_EXPORTER_OTLP_ENDPOINT,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logging.Fatal(ctx).Msgf("Creating grpc client: %v", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		logging.Fatal(ctx).Msgf("Creating OTLP client exporter: %v", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.APP_NAME),
		),
	)
	if err != nil {
		logging.Fatal(ctx).Msgf("Creating resource: %v", err)
	}

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
