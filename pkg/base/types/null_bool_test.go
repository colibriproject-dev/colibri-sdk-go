package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNullBool(t *testing.T) {
	t.Run("Should error when scan with a nil value", func(t *testing.T) {
		var result NullBool
		err := result.Scan(nil)

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, false, result.Bool)
	})

	t.Run("Should error when scan with a invalid value", func(t *testing.T) {
		value := "invalid"

		var result NullBool
		err := result.Scan(value)

		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, false, result.Bool)
	})

	t.Run("Should scan with a valid value", func(t *testing.T) {
		value := true

		var result NullBool
		err := result.Scan(value)

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, true, result.Valid)
		assert.Equal(t, value, result.Bool)
	})

	t.Run("Should get value with a valid value", func(t *testing.T) {
		expected := NullBool{true, true}

		result, err := expected.Value()
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expected.Bool, result)
	})

	t.Run("Should return nil when get value with a invalid value", func(t *testing.T) {
		expected := NullBool{false, false}

		result, err := expected.Value()
		assert.Nil(t, err)
		assert.Nil(t, result)
	})

	t.Run("Should return null when get json value with a invalid value", func(t *testing.T) {
		expected := NullBool{false, false}

		json, err := expected.MarshalJSON()
		result := string(json)
		assert.Nil(t, err)
		assert.Equal(t, "null", result)
	})

	t.Run("Should get json value with a valid value", func(t *testing.T) {
		expected := NullBool{true, true}

		json, err := expected.MarshalJSON()
		result := string(json)
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "true", result)
	})

	t.Run("Should get value with a valid json", func(t *testing.T) {
		var result NullBool
		err := result.UnmarshalJSON([]byte("true"))
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, true, result.Valid)
		assert.Equal(t, true, result.Bool)
	})

	t.Run("Should return error when get value with a invalid json", func(t *testing.T) {
		var result NullBool
		err := result.UnmarshalJSON([]byte("invalid"))
		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, false, result.Bool)
	})

	t.Run("Should handle null value in json", func(t *testing.T) {
		var result NullBool
		err := result.UnmarshalJSON([]byte("null"))

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, false, result.Bool)
	})
}
