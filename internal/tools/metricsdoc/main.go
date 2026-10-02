// Command metricsdoc rewrites the metric tables of docs/observability/metrics.md from the SDK
// metric catalog, so the doc cannot drift from what the SDK emits. Only the text between the
// markers is generated; the rest of the doc is written by hand.
//
// Run it from the repository root with `make metrics-doc`. It is deliberately not a
// go:generate directive: `make test` runs go generate, which would rewrite the doc before
// the drift test gets to compare it.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
)

const (
	beginMarker = "<!-- metrics:begin -->"
	endMarker   = "<!-- metrics:end -->"
)

func main() {
	path := flag.String("doc", "docs/observability/metrics.md", "path of the metrics reference doc")
	flag.Parse()

	doc, err := os.ReadFile(*path)
	if err != nil {
		log.Fatal(err)
	}

	updated, err := updateDoc(doc, renderTables(monitoring.Catalog()))
	if err != nil {
		log.Fatalf("%s: %v", *path, err)
	}

	if err = os.WriteFile(*path, updated, 0o644); err != nil {
		log.Fatal(err)
	}
}

// updateDoc replaces the text between the markers with the tables.
func updateDoc(doc []byte, tables string) ([]byte, error) {
	begin := bytes.Index(doc, []byte(beginMarker))
	end := bytes.Index(doc, []byte(endMarker))
	if begin < 0 || end < 0 || end < begin {
		return nil, errors.New("the doc must have the " + beginMarker + " and " + endMarker + " markers, in this order")
	}

	var out bytes.Buffer
	out.Write(doc[:begin+len(beginMarker)])
	out.WriteString("\n")
	out.WriteString(tables)
	out.Write(doc[end:])

	return out.Bytes(), nil
}

// section groups the catalog entries rendered in one table.
type section struct {
	title   string
	include func(monitoring.MetricDefinition) bool
}

var sections = []section{
	{"Emitted by the SDK", func(m monitoring.MetricDefinition) bool {
		return m.Origin == monitoring.OriginSDK
	}},
	{"Emitted by third-party instrumentation", func(m monitoring.MetricDefinition) bool {
		return m.Origin != monitoring.OriginSDK && m.Origin != monitoring.OriginPromClient && !m.Deprecated
	}},
	{"Exposed only on `/metrics`, by the Prometheus client collectors", func(m monitoring.MetricDefinition) bool {
		return m.Origin == monitoring.OriginPromClient
	}},
	{"Deprecated, emitted only with `OTEL_GO_X_DEPRECATED_RUNTIME_METRICS=true`", func(m monitoring.MetricDefinition) bool {
		return m.Deprecated
	}},
}

// renderTables renders one markdown table per section, in catalog order.
func renderTables(catalog []monitoring.MetricDefinition) string {
	var b strings.Builder
	for _, s := range sections {
		fmt.Fprintf(&b, "\n#### %s\n\n", s.title)
		b.WriteString("| Metric | Type | Unit | Attributes | Module | Origin | Description |\n")
		b.WriteString("|--------|------|------|------------|--------|--------|-------------|\n")
		for _, m := range catalog {
			if s.include(m) {
				fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s | %s |\n",
					m.Name, m.Kind, code(m.Unit), codeList(m.Attributes), m.Module, m.Origin, m.Description)
			}
		}
	}
	b.WriteString("\n")

	return b.String()
}

func code(s string) string {
	if s == "" {
		return "—"
	}

	return "`" + s + "`"
}

func codeList(values []string) string {
	if len(values) == 0 {
		return "—"
	}

	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = code(v)
	}

	return strings.Join(quoted, ", ")
}
