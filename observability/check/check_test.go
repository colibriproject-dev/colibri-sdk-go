package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/otlptranslator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const (
	dashboardsGlob = "../dashboards/*.json"
	rulesPath      = "../alerts/rules.yaml"
	brokenFixture  = "testdata/broken-dashboard.json"
)

// TestDashboardsAndRulesQueryOnlyCatalogedMetrics is the check CI runs: it fails as soon as
// a dashboard or an alert rule queries a metric the SDK does not emit.
func TestDashboardsAndRulesQueryOnlyCatalogedMetrics(t *testing.T) {
	known, err := KnownSeries()
	require.NoError(t, err)

	dashboards, err := filepath.Glob(dashboardsGlob)
	require.NoError(t, err)
	require.Len(t, dashboards, 4, "the http, messaging, database and runtime dashboards")

	var queries []Query
	for _, path := range dashboards {
		dashboardQueries, err := DashboardQueries(path)
		require.NoError(t, err)
		require.NotEmptyf(t, dashboardQueries, "%s has no queries", path)
		queries = append(queries, dashboardQueries...)
	}

	ruleQueries, err := RuleQueries(rulesPath)
	require.NoError(t, err)
	require.NotEmpty(t, ruleQueries, "the rule file has no rules")
	queries = append(queries, ruleQueries...)

	for _, problem := range Check(known, queries) {
		t.Error(problem)
	}
}

func TestCheckReportsAnUnknownMetric(t *testing.T) {
	t.Run("Should fail a dashboard querying a metric the catalog does not list", func(t *testing.T) {
		known, err := KnownSeries()
		require.NoError(t, err)
		queries, err := DashboardQueries(brokenFixture)
		require.NoError(t, err)

		problems := Check(known, queries)

		require.Len(t, problems, 2)
		assert.Contains(t, problems[0].Reason, "metric http_server_request_latency_seconds_bucket is not in the SDK metric catalog")
		assert.Contains(t, problems[0].Query.Location, `panel "Latency"`)
		assert.Contains(t, problems[1].Reason, "metric messaging_queue_depth is not in the SDK metric catalog")
		assert.Contains(t, problems[1].Query.Location, `variable "queue"`)
	})

	t.Run("Should fail an expression that does not parse", func(t *testing.T) {
		problems := Check(map[string]bool{}, []Query{{Expr: "rate(up[5m]"}})

		require.Len(t, problems, 1)
		assert.Contains(t, problems[0].Reason, "parsing the expression")
	})

	t.Run("Should fail a selector without a metric name", func(t *testing.T) {
		problems := Check(map[string]bool{}, []Query{{Expr: `{__name__=~"http_.*"}`}})

		require.Len(t, problems, 1)
		assert.Contains(t, problems[0].Reason, "has no metric name")
	})
}

func TestMetricNames(t *testing.T) {
	t.Run("Should return every selected metric once", func(t *testing.T) {
		names, err := MetricNames(`sum(rate(a_total{job="$job"}[$__rate_interval])) / sum(rate(a_total[${__range}])) + on() b`)

		require.NoError(t, err)
		assert.Equal(t, []string{"a_total", "b"}, names)
	})

	t.Run("Should take the name of a __name__ equality matcher", func(t *testing.T) {
		names, err := MetricNames(`{__name__="up", job="api"}`)

		require.NoError(t, err)
		assert.Equal(t, []string{"up"}, names)
	})
}

func TestVariableExpr(t *testing.T) {
	for query, expected := range map[string]string{
		`"label_values(target_info, job)"`:                        "target_info",
		`{"query": "label_values(up{job=\"$job\"}, instance)"}`:   `up{job="$job"}`,
		`"query_result(topk(5, go_goroutines))"`:                  "topk(5, go_goroutines)",
		`"label_values(job)"`:                                     "",
		`{"query": "", "refId": "PrometheusVariableQueryEditor"}`: "",
		`42`: "",
	} {
		assert.Equal(t, expected, variableExpr([]byte(query)), query)
	}
}

// TestSeriesNamesMatchTheExporter guards the translation the check relies on: each catalog
// metric is recorded through the same Prometheus exporter the SDK registers, and the name it
// exposes must be the one the check expects. A dependency bump that changes the translation
// fails here rather than in a dashboard.
func TestSeriesNamesMatchTheExporter(t *testing.T) {
	registry := prometheus.NewRegistry()
	exporter, err := otelprometheus.New(otelprometheus.WithRegisterer(registry))
	require.NoError(t, err)
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	meter := provider.Meter("check")

	expected := map[string]string{}
	for _, m := range monitoring.Catalog() {
		if m.Origin == monitoring.OriginPromClient {
			continue
		}

		record(t, meter, m)
		names, err := SeriesNames(m)
		require.NoError(t, err)
		expected[m.Name] = strings.TrimSuffix(names[0], "_bucket")
	}

	families, err := registry.Gather()
	require.NoError(t, err)
	exposed := map[string]bool{}
	for _, family := range families {
		exposed[family.GetName()] = true
	}

	for name, family := range expected {
		assert.Truef(t, exposed[family], "%s is not exposed as %s", name, family)
	}
}

// record creates the instrument of a catalog metric and records one value with it.
func record(t *testing.T, meter metric.Meter, m monitoring.MetricDefinition) {
	t.Helper()

	ctx := context.Background()
	description, unit := metric.WithDescription(m.Description), metric.WithUnit(m.Unit)
	observe := func(_ context.Context, o metric.Float64Observer) error {
		o.Observe(1)
		return nil
	}

	var err error
	switch m.Kind {
	case monitoring.KindCounter:
		var c metric.Float64Counter
		if c, err = meter.Float64Counter(m.Name, description, unit); err == nil {
			c.Add(ctx, 1)
		}
	case monitoring.KindUpDownCounter:
		var c metric.Float64UpDownCounter
		if c, err = meter.Float64UpDownCounter(m.Name, description, unit); err == nil {
			c.Add(ctx, 1)
		}
	case monitoring.KindHistogram:
		var h metric.Float64Histogram
		if h, err = meter.Float64Histogram(m.Name, description, unit); err == nil {
			h.Record(ctx, 1)
		}
	case monitoring.KindGauge:
		var g metric.Float64Gauge
		if g, err = meter.Float64Gauge(m.Name, description, unit); err == nil {
			g.Record(ctx, 1)
		}
	case monitoring.KindObservableCounter:
		_, err = meter.Float64ObservableCounter(m.Name, description, unit, metric.WithFloat64Callback(observe))
	case monitoring.KindObservableUpDownCounter:
		_, err = meter.Float64ObservableUpDownCounter(m.Name, description, unit, metric.WithFloat64Callback(observe))
	case monitoring.KindObservableGauge:
		_, err = meter.Float64ObservableGauge(m.Name, description, unit, metric.WithFloat64Callback(observe))
	default:
		t.Fatalf("%s has kind %s, which the check cannot record", m.Name, m.Kind)
	}
	require.NoErrorf(t, err, "creating %s", m.Name)
}

func TestSeriesNames(t *testing.T) {
	t.Run("Should expose the sum and count of a Prometheus client summary", func(t *testing.T) {
		names, err := SeriesNames(monitoring.MetricDefinition{Name: "go_gc_duration_seconds",
			Kind: monitoring.KindSummary, Origin: monitoring.OriginPromClient})

		require.NoError(t, err)
		assert.Equal(t, []string{"go_gc_duration_seconds", "go_gc_duration_seconds_sum", "go_gc_duration_seconds_count"}, names)
	})

	t.Run("Should fail a name the translation leaves empty", func(t *testing.T) {
		_, err := SeriesNames(monitoring.MetricDefinition{Kind: monitoring.KindGauge, Origin: monitoring.OriginSDK})

		assert.ErrorContains(t, err, "translating")
	})
}

func TestMetricType(t *testing.T) {
	assert.Equal(t, otlptranslator.MetricType(otlptranslator.MetricTypeSummary), metricType(monitoring.KindSummary))
	assert.Equal(t, otlptranslator.MetricType(otlptranslator.MetricTypeUnknown), metricType("unknown"))
}

func TestProblemString(t *testing.T) {
	problem := Problem{Query{File: "http.json", Location: `panel "Latency", target A`, Expr: "up"}, "metric up is missing"}

	assert.Equal(t, "http.json (panel \"Latency\", target A): metric up is missing\n\tup", problem.String())
}

func TestDashboardQueriesFails(t *testing.T) {
	t.Run("Should fail a missing file", func(t *testing.T) {
		_, err := DashboardQueries(filepath.Join(t.TempDir(), "missing.json"))

		assert.Error(t, err)
	})

	t.Run("Should fail a file that is not JSON", func(t *testing.T) {
		_, err := DashboardQueries(writeFile(t, "dashboard.json", "{"))

		assert.ErrorContains(t, err, "dashboard.json")
	})
}

func TestRuleQueriesFails(t *testing.T) {
	t.Run("Should fail a missing file", func(t *testing.T) {
		_, err := RuleQueries(filepath.Join(t.TempDir(), "missing.yaml"))

		assert.Error(t, err)
	})

	t.Run("Should fail a file that is not YAML", func(t *testing.T) {
		_, err := RuleQueries(writeFile(t, "rules.yaml", "groups: ["))

		assert.ErrorContains(t, err, "rules.yaml")
	})
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}
