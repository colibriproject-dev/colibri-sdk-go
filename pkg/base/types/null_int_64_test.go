package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNullInt64(t *testing.T) {
	t.Run("Should error when scan with a nil value", func(t *testing.T) {
		var result NullInt64
		err := result.Scan(nil)

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, int64(0), result.Int64)
	})

	t.Run("Should error when scan with a invalid value", func(t *testing.T) {
		value := "invalid"

		var result NullInt64
		err := result.Scan(value)

		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, int64(0), result.Int64)
	})

	t.Run("Should scan with a valid value", func(t *testing.T) {
		value := int64(123)

		var result NullInt64
		err := result.Scan(value)

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, true, result.Valid)
		assert.Equal(t, value, result.Int64)
	})

	t.Run("Should get value with a valid value", func(t *testing.T) {
		expected := NullInt64{123, true}

		result, err := expected.Value()
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expected.Int64, result)
	})

	t.Run("Should return nil when get value with a invalid value", func(t *testing.T) {
		expected := NullInt64{0, false}

		result, err := expected.Value()
		assert.Nil(t, err)
		assert.Nil(t, result)
	})

	t.Run("Should return null when get json value with a invalid value", func(t *testing.T) {
		expected := NullInt64{0, false}

		json, err := expected.MarshalJSON()
		result := string(json)
		assert.Nil(t, err)
		assert.Equal(t, "null", result)
	})

	t.Run("Should get json value with a valid value", func(t *testing.T) {
		expected := NullInt64{123, true}

		json, err := expected.MarshalJSON()
		result := string(json)
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "123", result)
	})

	t.Run("Should get value with a valid json", func(t *testing.T) {
		var result NullInt64
		err := result.UnmarshalJSON([]byte("123"))
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, true, result.Valid)
		assert.Equal(t, int64(123), result.Int64)
	})

	t.Run("Should return error when get value with a invalid json", func(t *testing.T) {
		var result NullInt64
		err := result.UnmarshalJSON([]byte("invalid"))
		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, int64(0), result.Int64)
	})

	t.Run("Should handle null value in json", func(t *testing.T) {
		var result NullInt64
		err := result.UnmarshalJSON([]byte("null"))

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, int64(0), result.Int64)
	})
}
