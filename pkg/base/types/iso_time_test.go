package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsoTime(t *testing.T) {
	t.Run("Should get parsed iso time", func(t *testing.T) {
		expected := IsoTime(time.Date(0, time.January, 1, 10, 20, 30, 0, time.UTC))

		result, err := ParseIsoTime("10:20:30")

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expected, result)
	})

	t.Run("Should return error when parse with a invalid string", func(t *testing.T) {
		result, err := ParseIsoTime("invalid")

		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, IsoTime(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), result)
	})

	t.Run("Should get string iso time", func(t *testing.T) {
		expected := "10:20:30"

		result := IsoTime(time.Date(0, time.January, 1, 10, 20, 30, 0, time.UTC)).String()

		assert.NotNil(t, result)
		assert.Equal(t, expected, result)
	})

	t.Run("Should get go string iso time", func(t *testing.T) {
		expected := "time.Date(0, time.January, 1, 10, 20, 30, 0, time.UTC)"

		result := IsoTime(time.Date(0, time.January, 1, 10, 20, 30, 0, time.UTC)).GoString()

		assert.NotNil(t, result)
		assert.Equal(t, expected, result)
	})

	t.Run("Should get value with a valid value", func(t *testing.T) {
		expected := time.Date(1, time.January, 1, 10, 20, 30, 0, time.UTC)

		result, err := IsoTime(expected).Value()

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expected, result)
	})

	t.Run("Should get json value with a valid value", func(t *testing.T) {
		expected := IsoTime(time.Date(0, time.January, 1, 10, 20, 30, 0, time.UTC))

		json, err := expected.MarshalJSON()
		result := string(json)

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "\"10:20:30\"", result)
	})

	t.Run("Should get value with a valid json", func(t *testing.T) {
		var result IsoTime
		err := result.UnmarshalJSON([]byte("\"10:20:30\""))

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, IsoTime(time.Date(0, time.January, 1, 10, 20, 30, 0, time.UTC)), result)
	})

	t.Run("Should return error when get value with a invalid json", func(t *testing.T) {
		var result IsoTime
		err := result.UnmarshalJSON([]byte("invalid"))

		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, IsoTime(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), result)
	})

	t.Run("Should return error when get value with a isoDate json", func(t *testing.T) {
		var result IsoTime
		err := result.UnmarshalJSON([]byte("\"2022-01-01\""))

		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, IsoTime(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), result)
	})

	t.Run("Should handle null value in json", func(t *testing.T) {
		var result IsoTime
		err := result.UnmarshalJSON([]byte("null"))

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, IsoTime(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), result)
	})
}
