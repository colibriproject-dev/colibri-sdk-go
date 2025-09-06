package colibri_otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitHeadersConfig(t *testing.T) {

	t.Run("Should return empty slice when headers is empty", func(t *testing.T) {
		result := splitAndTrim("", ",")

		assert.Empty(t, result)
	})

	t.Run("Should return slice with trimmed values", func(t *testing.T) {
		result := splitAndTrim("  a, b,  c  ", ",")

		assert.Len(t, result, 3)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})
}
