package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNullIsoTime(t *testing.T) {
	t.Run("Should error when scan with a nil value", func(t *testing.T) {
		var result NullIsoTime
		err := result.Scan(nil)

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, time.Time(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), result.Time)
	})

	t.Run("Should error when scan with a invalid value", func(t *testing.T) {
		var result NullIsoTime
		err := result.Scan("invalid")

		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, time.Time(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), result.Time)
	})

	t.Run("Should scan with a valid value", func(t *testing.T) {
		expected := time.Now()

		var result NullIsoTime
		err := result.Scan(expected)

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, true, result.Valid)
		assert.Equal(t, expected, result.Time)
	})

	t.Run("Should get value with a valid value", func(t *testing.T) {
		expected := NullIsoTime{time.Now(), true}

		result, err := expected.Value()

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expected.Time, result)
	})

	t.Run("Should return nil when get value with a invalid value", func(t *testing.T) {
		expected := NullIsoTime{time.Time(time.Date(1, time.January, 1, 10, 20, 30, 0, time.UTC)), false}

		result, err := expected.Value()
		assert.Nil(t, err)
		assert.Nil(t, result)
	})

	t.Run("Should return null when get json value with a invalid value", func(t *testing.T) {
		expected := NullIsoTime{time.Time(time.Date(1, time.January, 1, 10, 20, 30, 0, time.UTC)), false}

		json, err := expected.MarshalJSON()
		result := string(json)

		assert.Nil(t, err)
		assert.Equal(t, "null", result)
	})

	t.Run("Should get json value with a valid value", func(t *testing.T) {
		expected := NullIsoTime{time.Time(time.Date(0, time.January, 1, 10, 20, 30, 0, time.UTC)), true}

		json, err := expected.MarshalJSON()
		result := string(json)

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "\"10:20:30\"", result)
	})

	t.Run("Should get value with a valid json", func(t *testing.T) {
		var result NullIsoTime
		err := result.UnmarshalJSON([]byte("\"10:20:30\""))

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, true, result.Valid)
		assert.Equal(t, time.Time(time.Date(0, time.January, 1, 10, 20, 30, 0, time.UTC)), result.Time)
	})

	t.Run("Should return error when get value with a invalid json", func(t *testing.T) {
		var result NullIsoTime
		err := result.UnmarshalJSON([]byte("invalid"))

		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, time.Time(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), result.Time)
	})

	t.Run("Should return error when get value with a isoDate json", func(t *testing.T) {
		var result NullIsoTime
		err := result.UnmarshalJSON([]byte("\"2022-01-01\""))

		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, time.Time(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), result.Time)
	})

	t.Run("Should handle null value in json", func(t *testing.T) {
		var result NullIsoTime
		err := result.UnmarshalJSON([]byte("null"))

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, time.Time(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), result.Time)
	})
}
