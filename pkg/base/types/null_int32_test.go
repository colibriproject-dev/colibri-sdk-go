package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNullInt32(t *testing.T) {
	t.Run("Should error when scan with a nil value", func(t *testing.T) {
		var result NullInt32
		err := result.Scan(nil)

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, int32(0), result.Int32)
	})

	t.Run("Should error when scan with a invalid value", func(t *testing.T) {
		value := "invalid"

		var result NullInt32
		err := result.Scan(value)

		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, int32(0), result.Int32)
	})

	t.Run("Should scan with a valid value", func(t *testing.T) {
		value := int32(123)

		var result NullInt32
		err := result.Scan(value)

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, true, result.Valid)
		assert.Equal(t, value, result.Int32)
	})

	t.Run("Should get value with a valid value", func(t *testing.T) {
		expected := NullInt32{12, true}

		result, err := expected.Value()
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expected.Int32, result)
	})

	t.Run("Should return nil when get value with a invalid value", func(t *testing.T) {
		expected := NullInt32{0, false}

		result, err := expected.Value()
		assert.Nil(t, err)
		assert.Nil(t, result)
	})

	t.Run("Should return null when get json value with a invalid value", func(t *testing.T) {
		expected := NullInt32{0, false}

		json, err := expected.MarshalJSON()
		result := string(json)
		assert.Nil(t, err)
		assert.Equal(t, "null", result)
	})

	t.Run("Should get json value with a valid value", func(t *testing.T) {
		expected := NullInt32{123, true}

		json, err := expected.MarshalJSON()
		result := string(json)
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "123", result)
	})

	t.Run("Should get value with a valid json", func(t *testing.T) {
		var result NullInt32
		err := result.UnmarshalJSON([]byte("123"))
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, true, result.Valid)
		assert.Equal(t, int32(123), result.Int32)
	})

	t.Run("Should return error when get value with a invalid json", func(t *testing.T) {
		var result NullInt32
		err := result.UnmarshalJSON([]byte("invalid"))
		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, int32(0), result.Int32)
	})
}
