package colibri_monitoring_base

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAttrs(t *testing.T) {
	t.Run("Should build pairs sorted by key", func(t *testing.T) {
		attrs := NewAttrs("status", "200", "route", "/api/users")

		assert.Equal(t, []Attr{{Key: "route", Value: "/api/users"}, {Key: "status", Value: "200"}}, attrs.Pairs())
		assert.Equal(t, 2, attrs.Len())
	})

	t.Run("Should ignore a trailing key without value", func(t *testing.T) {
		attrs := NewAttrs("route", "/api/users", "status")

		assert.Equal(t, []Attr{{Key: "route", Value: "/api/users"}}, attrs.Pairs())
	})

	t.Run("Should keep the last value of a duplicated key", func(t *testing.T) {
		attrs := NewAttrs("route", "/first", "route", "/last")

		assert.Equal(t, []Attr{{Key: "route", Value: "/last"}}, attrs.Pairs())
	})

	t.Run("Should build an empty set without arguments", func(t *testing.T) {
		assert.Empty(t, NewAttrs().Pairs())
	})
}

func TestAttrsFromMap(t *testing.T) {
	t.Run("Should build the same set regardless of map iteration order", func(t *testing.T) {
		source := map[string]string{"route": "/api/users", "status": "200", "method": "GET"}

		first := AttrsFromMap(source)
		for range 20 {
			assert.Equal(t, first.Pairs(), AttrsFromMap(source).Pairs())
		}
	})

	t.Run("Should build an empty set from a nil map", func(t *testing.T) {
		assert.Empty(t, AttrsFromMap(nil).Pairs())
	})
}

func TestAttrsZeroValue(t *testing.T) {
	var attrs Attrs

	t.Run("Should carry no attributes", func(t *testing.T) {
		assert.Nil(t, attrs.Pairs())
		assert.Equal(t, 0, attrs.Len())
	})

	t.Run("Should build with nil pairs", func(t *testing.T) {
		assert.Equal(t, "built", attrs.Cached(func(pairs []Attr) any {
			assert.Nil(t, pairs)
			return "built"
		}))
	})
}

func TestAttrsCached(t *testing.T) {
	t.Run("Should build once and reuse the result", func(t *testing.T) {
		attrs := NewAttrs("route", "/api/users")
		builds := 0

		build := func(pairs []Attr) any {
			builds++
			return len(pairs)
		}

		require.Equal(t, 1, attrs.Cached(build))
		assert.Equal(t, 1, attrs.Cached(build))
		assert.Equal(t, 1, builds)
	})

	t.Run("Should share the cache across copies", func(t *testing.T) {
		attrs := NewAttrs("route", "/api/users")
		copied := attrs
		builds := 0

		build := func([]Attr) any {
			builds++
			return builds
		}

		attrs.Cached(build)
		copied.Cached(build)

		assert.Equal(t, 1, builds)
	})
}
