package main

import (
	"os"
	"strings"
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const docPath = "../../../docs/observability/metrics.md"

func TestMetricsDocIsUpToDate(t *testing.T) {
	doc, err := os.ReadFile(docPath)
	require.NoError(t, err)

	updated, err := updateDoc(doc, renderTables(monitoring.Catalog()))
	require.NoError(t, err)

	assert.Equal(t, string(updated), string(doc),
		"docs/observability/metrics.md is out of date with the metric catalog: run `make metrics-doc`")
}

func TestUpdateDoc(t *testing.T) {
	t.Run("Should replace only the text between the markers", func(t *testing.T) {
		doc := "intro\n" + beginMarker + "\nstale\n" + endMarker + "\noutro\n"

		updated, err := updateDoc([]byte(doc), "fresh\n")

		require.NoError(t, err)
		assert.Equal(t, "intro\n"+beginMarker+"\nfresh\n"+endMarker+"\noutro\n", string(updated))
	})

	t.Run("Should reject a doc without the markers", func(t *testing.T) {
		for _, doc := range []string{"no markers", endMarker + beginMarker, beginMarker} {
			_, err := updateDoc([]byte(doc), "fresh\n")
			assert.Error(t, err, doc)
		}
	})
}

func TestRenderTables(t *testing.T) {
	t.Run("Should list every cataloged metric once", func(t *testing.T) {
		tables := renderTables(monitoring.Catalog())

		for _, m := range monitoring.Catalog() {
			assert.Equalf(t, 1, strings.Count(tables, "| `"+m.Name+"` |"), "%s is not listed once", m.Name)
		}
	})

	t.Run("Should render a missing unit or attribute set as a dash", func(t *testing.T) {
		tables := renderTables([]monitoring.MetricDefinition{{
			Name: "app.metric", Description: "A metric", Kind: monitoring.KindCounter,
			Origin: monitoring.OriginSDK, Module: "app",
		}})

		assert.Contains(t, tables, "| `app.metric` | counter | — | — | app | sdk | A metric |")
	})
}
