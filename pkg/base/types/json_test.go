package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonB(t *testing.T) {
	t.Run("should error with nil value", func(t *testing.T) {
		var result JsonB
		err := result.Scan(nil)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidValue)
		assert.Nil(t, result)
	})

	t.Run("should error with a syntax error", func(t *testing.T) {
		value := "{\"msg\":\"test\""

		var result JsonB
		err := result.Scan([]byte(value))

		assert.Error(t, err)
		assert.ErrorContains(t, err, "unexpected end of JSON input")
		assert.Nil(t, result)
	})

	t.Run("should process ok", func(t *testing.T) {
		value := "{\"msg\":\"test\"}"

		var result JsonB
		err := result.Scan([]byte(value))

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test", result["msg"])
	})

	t.Run("should get value ok", func(t *testing.T) {
		expected := "{\"msg\":\"test\"}"

		var value JsonB
		err := value.Scan([]byte(expected))
		assert.NoError(t, err)

		result, err := value.Value()
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expected, string(result.([]byte)))
	})
}
