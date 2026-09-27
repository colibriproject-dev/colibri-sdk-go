package monitoring

import (
	"regexp"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// maxSDKAttributes is the attribute ceiling of the metrics the SDK emits, the one
// docs/observability/metrics.md recommends. Third-party metrics follow their own conventions.
const maxSDKAttributes = 5

// forbiddenAttributes are unbounded identifiers, which belong on spans only.
var forbiddenAttributes = []string{
	"correlationId", "correlation_id", "correlation.id",
	"messageId", "message_id", "messaging.message.id",
	"userId", "user_id", "user.id", "enduser.id",
	"tenantId", "tenant_id", "tenant.id",
	"key", "storage.key", "path", "url.path", "url.full", "url.query",
}

var (
	ucumUnits      = []string{"s", "ms", "ns", "By", "1", "%"}
	ucumAnnotation = regexp.MustCompile(`^\{[a-z_]+\}$`)
	validKinds     = []MetricKind{KindCounter, KindUpDownCounter, KindHistogram, KindGauge, KindObservableCounter, KindObservableUpDownCounter, KindObservableGauge}
	validOrigins   = []MetricOrigin{OriginSDK, OriginOtelHTTP, OriginOtelSQL, OriginRedisOtel, OriginRuntime}
)

func TestMetricCatalog(t *testing.T) {
	t.Run("Should not list a metric twice", func(t *testing.T) {
		seen := map[string]bool{}
		for _, m := range Catalog() {
			assert.Falsef(t, seen[m.Name], "%s is listed twice", m.Name)
			seen[m.Name] = true
		}
	})

	t.Run("Should describe every metric completely", func(t *testing.T) {
		for _, m := range Catalog() {
			assert.NotEmptyf(t, m.Name, "metric %+v has no name", m)
			assert.NotEmptyf(t, m.Description, "%s has no description", m.Name)
			assert.NotEmptyf(t, m.Module, "%s has no module", m.Name)
			assert.Containsf(t, validKinds, m.Kind, "%s has an unknown kind", m.Name)
			assert.Containsf(t, validOrigins, m.Origin, "%s has an unknown origin", m.Name)
		}
	})

	t.Run("Should record every SDK metric in a UCUM unit", func(t *testing.T) {
		for _, m := range Catalog() {
			if m.Unit == "" {
				// only third-party instrumentation leaves the unit out
				assert.NotEqualf(t, OriginSDK, m.Origin, "%s has no unit", m.Name)
				continue
			}
			assert.Truef(t, slices.Contains(ucumUnits, m.Unit) || ucumAnnotation.MatchString(m.Unit),
				"%s has unit %q, which is not UCUM", m.Name, m.Unit)
		}
	})

	t.Run("Should never carry an unbounded identifier as attribute", func(t *testing.T) {
		for _, m := range Catalog() {
			for _, attribute := range m.Attributes {
				assert.NotContainsf(t, forbiddenAttributes, attribute, "%s carries %s", m.Name, attribute)
			}
		}
	})

	t.Run("Should keep the SDK metrics within the attribute ceiling", func(t *testing.T) {
		for _, m := range Catalog() {
			if m.Origin == OriginSDK {
				assert.LessOrEqualf(t, len(m.Attributes), maxSDKAttributes, "%s has too many attributes", m.Name)
			}
		}
	})

	t.Run("Should deprecate only the legacy runtime metrics", func(t *testing.T) {
		for _, m := range Catalog() {
			if m.Deprecated {
				assert.Equalf(t, OriginRuntime, m.Origin, "%s is deprecated", m.Name)
			}
		}
	})

	t.Run("Should return a copy of the catalog", func(t *testing.T) {
		definitions := Catalog()
		definitions[0].Name = "changed"
		definitions[0].Attributes[0] = "changed"

		original := Catalog()[0]
		assert.NotEqual(t, "changed", original.Name)
		assert.NotContains(t, original.Attributes, "changed")
	})
}

func TestLookupMetric(t *testing.T) {
	t.Run("Should return the definition of a cataloged metric", func(t *testing.T) {
		m, ok := LookupMetric(MetricMessagingPublished)

		require.True(t, ok)
		assert.Equal(t, KindCounter, m.Kind)
		assert.Equal(t, "{message}", m.Unit)
		assert.Equal(t, []string{"topic", "result"}, m.Attributes)
		assert.Equal(t, OriginSDK, m.Origin)
	})

	t.Run("Should report a metric the SDK does not emit", func(t *testing.T) {
		_, ok := LookupMetric("unknown.metric")

		assert.False(t, ok)
	})

	t.Run("Should return a copy of the definition", func(t *testing.T) {
		m, _ := LookupMetric(MetricMessagingPublished)
		m.Attributes[0] = "changed"

		again, _ := LookupMetric(MetricMessagingPublished)
		assert.Equal(t, "topic", again.Attributes[0])
	})
}
