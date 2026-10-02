// Package check verifies that the Grafana dashboards and the Prometheus alert rules under
// observability/ only query metrics the SDK emits, as listed in its metric catalog. A metric
// renamed or removed in the SDK then fails the build here instead of leaving a dashboard
// panel empty or an alert that can never fire.
package check

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/prometheus/otlptranslator"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"gopkg.in/yaml.v3"
)

// builtinSeries are the series every Prometheus target has, whatever the SDK emits: up is
// added by the scrape itself and target_info by the OpenTelemetry exporter.
var builtinSeries = []string{"up", "target_info"}

// KnownSeries returns every series name a query may select: the catalog metrics as /metrics
// exposes them, plus the built-in series.
func KnownSeries() (map[string]bool, error) {
	known := map[string]bool{}
	for _, name := range builtinSeries {
		known[name] = true
	}

	for _, m := range monitoring.Catalog() {
		names, err := SeriesNames(m)
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			known[name] = true
		}
	}

	return known, nil
}

// SeriesNames returns the series /metrics exposes for a catalog metric. OpenTelemetry names are
// translated the way the Prometheus exporter of the SDK does it: dots to underscores, the unit
// as a suffix and _total on counters. Histograms and summaries expose more than one series.
func SeriesNames(m monitoring.MetricDefinition) ([]string, error) {
	if m.Origin == monitoring.OriginPromClient {
		if m.Kind == monitoring.KindSummary {
			return []string{m.Name, m.Name + "_sum", m.Name + "_count"}, nil
		}
		return []string{m.Name}, nil
	}

	namer := otlptranslator.NewMetricNamer("", otlptranslator.UnderscoreEscapingWithSuffixes)
	name, err := namer.Build(otlptranslator.Metric{Name: m.Name, Unit: m.Unit, Type: metricType(m.Kind)})
	if err != nil {
		return nil, fmt.Errorf("translating %s: %w", m.Name, err)
	}

	if m.Kind == monitoring.KindHistogram {
		return []string{name + "_bucket", name + "_sum", name + "_count"}, nil
	}

	return []string{name}, nil
}

func metricType(kind monitoring.MetricKind) otlptranslator.MetricType {
	switch kind {
	case monitoring.KindCounter, monitoring.KindObservableCounter:
		return otlptranslator.MetricTypeMonotonicCounter
	case monitoring.KindUpDownCounter, monitoring.KindObservableUpDownCounter:
		return otlptranslator.MetricTypeNonMonotonicCounter
	case monitoring.KindGauge, monitoring.KindObservableGauge:
		return otlptranslator.MetricTypeGauge
	case monitoring.KindHistogram:
		return otlptranslator.MetricTypeHistogram
	case monitoring.KindSummary:
		return otlptranslator.MetricTypeSummary
	default:
		return otlptranslator.MetricTypeUnknown
	}
}

// Query is a PromQL expression found in a dashboard or rule file, with where it came from.
type Query struct {
	File     string
	Location string
	Expr     string
}

// Problem is a query that does not pass the check.
type Problem struct {
	Query  Query
	Reason string
}

func (p Problem) String() string {
	return fmt.Sprintf("%s (%s): %s\n\t%s", p.Query.File, p.Query.Location, p.Reason, p.Query.Expr)
}

// Check returns the problems of every query in the files: an expression that does not parse,
// a selector without a metric name, or a metric the catalog does not list.
func Check(known map[string]bool, queries []Query) []Problem {
	var problems []Problem
	for _, q := range queries {
		names, err := MetricNames(q.Expr)
		if err != nil {
			problems = append(problems, Problem{q, err.Error()})
			continue
		}
		for _, name := range names {
			if !known[name] {
				problems = append(problems, Problem{q, fmt.Sprintf("metric %s is not in the SDK metric catalog", name)})
			}
		}
	}

	return problems
}

// grafanaVariables are the Grafana interval variables, which are not PromQL. They are replaced
// with a duration before parsing; any other variable must sit inside a label matcher string.
var grafanaVariables = regexp.MustCompile(`\$__(rate_interval|interval|range)\b|\$\{__(rate_interval|interval|range)\}`)

// MetricNames returns the metric names an expression selects, sorted and without duplicates.
func MetricNames(expr string) ([]string, error) {
	parsed, err := promql.ParseExpr(grafanaVariables.ReplaceAllString(expr, "5m"))
	if err != nil {
		return nil, fmt.Errorf("parsing the expression: %w", err)
	}

	var names []string
	var selectorErr error
	parser.Inspect(parsed, func(node parser.Node, _ []parser.Node) error {
		selector, ok := node.(*parser.VectorSelector)
		if !ok {
			return nil
		}

		name := selectorName(selector)
		if name == "" {
			selectorErr = fmt.Errorf("selector %s has no metric name, so it cannot be checked", selector)
			return selectorErr
		}
		names = append(names, name)
		return nil
	})
	if selectorErr != nil {
		return nil, selectorErr
	}

	sort.Strings(names)
	return slices.Compact(names), nil
}

func selectorName(selector *parser.VectorSelector) string {
	if selector.Name != "" {
		return selector.Name
	}

	for _, m := range selector.LabelMatchers {
		if m.Name == labels.MetricName && m.Type == labels.MatchEqual {
			return m.Value
		}
	}

	return ""
}

// promql parses the expressions with the options Prometheus itself defaults to, so a query
// using an experimental feature fails here as it would on a stock server.
var promql = parser.NewParser(parser.Options{})

// labelValues matches the label_values and query_result variable queries of Grafana, whose
// argument is a PromQL expression.
var (
	labelValues = regexp.MustCompile(`^\s*label_values\((.+),\s*[a-zA-Z_][a-zA-Z0-9_]*\s*\)\s*$`)
	queryResult = regexp.MustCompile(`^\s*query_result\((.+)\)\s*$`)
)

// DashboardQueries returns the queries of a Grafana dashboard: the targets of every panel,
// including panels nested in rows, and the template variable queries.
func DashboardQueries(path string) ([]Query, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var dashboard struct {
		Panels     []panel `json:"panels"`
		Templating struct {
			List []struct {
				Name  string          `json:"name"`
				Type  string          `json:"type"`
				Query json.RawMessage `json:"query"`
			} `json:"list"`
		} `json:"templating"`
	}
	if err = json.Unmarshal(content, &dashboard); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	file := filepath.Base(path)
	var queries []Query
	var walk func([]panel)
	walk = func(panels []panel) {
		for _, p := range panels {
			for _, target := range p.Targets {
				if target.Expr != "" {
					queries = append(queries, Query{file, fmt.Sprintf("panel %q, target %s", p.Title, target.RefID), target.Expr})
				}
			}
			walk(p.Panels)
		}
	}
	walk(dashboard.Panels)

	for _, variable := range dashboard.Templating.List {
		if variable.Type != "query" {
			continue
		}
		if expr := variableExpr(variable.Query); expr != "" {
			queries = append(queries, Query{file, fmt.Sprintf("variable %q", variable.Name), expr})
		}
	}

	return queries, nil
}

type panel struct {
	Title   string `json:"title"`
	Targets []struct {
		RefID string `json:"refId"`
		Expr  string `json:"expr"`
	} `json:"targets"`
	Panels []panel `json:"panels"`
}

// variableExpr returns the PromQL expression of a variable query, which Grafana stores either
// as a string or as an object holding it.
func variableExpr(raw json.RawMessage) string {
	var query string
	if err := json.Unmarshal(raw, &query); err != nil {
		var object struct {
			Query string `json:"query"`
		}
		if err = json.Unmarshal(raw, &object); err != nil {
			return ""
		}
		query = object.Query
	}

	for _, pattern := range []*regexp.Regexp{labelValues, queryResult} {
		if match := pattern.FindStringSubmatch(query); match != nil {
			return match[1]
		}
	}

	return ""
}

// RuleQueries returns the expressions of the recording and alerting rules of a rule file.
func RuleQueries(path string) ([]Query, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var file struct {
		Groups []struct {
			Name  string `yaml:"name"`
			Rules []struct {
				Alert  string `yaml:"alert"`
				Record string `yaml:"record"`
				Expr   string `yaml:"expr"`
			} `yaml:"rules"`
		} `yaml:"groups"`
	}
	if err = yaml.Unmarshal(content, &file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	name := filepath.Base(path)
	var queries []Query
	for _, group := range file.Groups {
		for _, rule := range group.Rules {
			queries = append(queries, Query{name, fmt.Sprintf("group %q, rule %q", group.Name, rule.Alert+rule.Record), rule.Expr})
		}
	}

	return queries, nil
}
