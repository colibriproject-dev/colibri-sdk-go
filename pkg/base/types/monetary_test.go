package types

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestMonetary(t *testing.T) {
	t.Run("Should add two instances", func(t *testing.T) {
		m1 := NewMonetaryFromFloat(10.50)
		m2 := NewMonetaryFromFloat(5.25)
		expected := NewMonetaryFromFloat(15.75)

		result := m1.Add(m2)

		assert.Equal(t, expected.Decimal, result.Decimal)
	})

	t.Run("Should subtract two instances", func(t *testing.T) {
		m1 := NewMonetaryFromFloat(10.50)
		m2 := NewMonetaryFromFloat(5.25)
		expected := NewMonetaryFromFloat(5.25)

		result := m1.Sub(m2)

		assert.Equal(t, expected.Decimal, result.Decimal)
	})

	t.Run("Should multiply two instances", func(t *testing.T) {
		m1 := NewMonetaryFromFloat(10.00)
		m2 := NewMonetaryFromFloat(2.00)
		expected := NewMonetaryFromFloat(20.00)

		result := m1.Mul(m2)

		assert.Equal(t, expected.Decimal, result.Decimal)
	})

	t.Run("Should divide two instances", func(t *testing.T) {
		m1 := NewMonetaryFromFloat(10.00)
		m2 := NewMonetaryFromFloat(2.00)
		expected := NewMonetaryFromFloat(5.00)

		result := m1.Div(m2)

		assert.True(t, expected.Equal(result),
			"Expected %s but got %s", expected.String(), result.Decimal.String())
	})

	t.Run("Should marshal to JSON with at least 2 decimal places", func(t *testing.T) {
		testCases := []struct {
			value    Monetary
			expected string
		}{
			{NewMonetaryFromFloat(10), `"10.00"`},
			{NewMonetaryFromFloat(10.5), `"10.50"`},
			{NewMonetaryFromFloat(10.55), `"10.55"`},
			{NewMonetaryFromFloat(10.555), `"10.555"`},
		}

		for _, tc := range testCases {
			data, err := json.Marshal(tc.value)

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, string(data))
		}
	})

	t.Run("Should unmarshal from JSON string", func(t *testing.T) {
		input := `"10.55"`
		var m Monetary

		err := json.Unmarshal([]byte(input), &m)

		assert.NoError(t, err)
		assert.Equal(t, decimal.NewFromFloat(10.55), m.Decimal)
	})

	t.Run("Should fail to unmarshal from invalid JSON string", func(t *testing.T) {
		input := `"test"`
		var m Monetary

		err := json.Unmarshal([]byte(input), &m)

		assert.Error(t, err)
	})

	t.Run("Should chain operations", func(t *testing.T) {
		m1 := NewMonetaryFromFloat(10.50)
		m2 := NewMonetaryFromFloat(5.25)
		m3 := NewMonetaryFromFloat(2.00)
		expected := NewMonetaryFromFloat(10.50)

		// (10.50 + 5.25) / 3 * 2
		result := m1.Add(m2).Div(NewMonetaryFromFloat(3)).Mul(m3)

		// Compare the values rather than the exact decimal representations
		assert.True(t, expected.Equal(result),
			"Expected %s but got %s", expected.String(), result.Decimal.String())
	})

	t.Run("Should create from decimal", func(t *testing.T) {
		d := decimal.NewFromFloat(123.45)

		m := NewMonetaryFromDecimal(d)

		assert.Equal(t, d, m.Decimal)
	})

	t.Run("Should handle zero value", func(t *testing.T) {
		m := NewMonetaryFromFloat(0)

		data, _ := json.Marshal(m)
		assert.Equal(t, `"0.00"`, string(data))
	})

	t.Run("Should handle negative values", func(t *testing.T) {
		m := NewMonetaryFromFloat(-10.5)

		data, _ := json.Marshal(m)
		assert.Equal(t, `"-10.50"`, string(data))
	})

	t.Run("Should create from cents", func(t *testing.T) {
		cents := int64(1055)
		expected := NewMonetaryFromFloat(10.55)

		m := NewMonetaryFromCents(cents)

		assert.True(t, expected.Equal(m),
			"Expected %s but got %s", expected.String(), m.String())
	})

	t.Run("Should convert to cents", func(t *testing.T) {
		testCases := []struct {
			monetary Monetary
			expected int64
		}{
			{NewMonetaryFromFloat(10.55), 1055},
			{NewMonetaryFromFloat(0), 0},
			{NewMonetaryFromFloat(-5.75), -575},
			{NewMonetaryFromFloat(100.0), 10000},
		}

		for _, tc := range testCases {
			cents := tc.monetary.Cents()

			assert.Equal(t, tc.expected, cents,
				"Expected %d cents but got %d for value %s",
				tc.expected, cents, tc.monetary.String())
		}
	})

	t.Run("Should round-trip between units and cents", func(t *testing.T) {
		original := NewMonetaryFromFloat(123.45)

		cents := original.Cents()
		roundTrip := NewMonetaryFromCents(cents)

		assert.True(t, original.Equal(roundTrip),
			"Expected %s but got %s after round trip",
			original.String(), roundTrip.String())
	})

	t.Run("Should parse monetary from valid string", func(t *testing.T) {
		str := "123.45"
		expected := NewMonetaryFromFloat(123.45)

		result, err := ParseMonetary(str)

		assert.Nil(t, err)
		assert.True(t, expected.Equal(result),
			"Expected %s but got %s", expected.String(), result.String())
	})

	t.Run("Should return error when parsing invalid string", func(t *testing.T) {
		str := "invalid"

		result, err := ParseMonetary(str)

		assert.NotNil(t, err)
		assert.Equal(t, Monetary{}, result)
	})

	t.Run("Should convert to sql driver value", func(t *testing.T) {
		m := NewMonetaryFromFloat(123.45)
		expected := decimal.NewFromFloat(123.45)

		result, err := m.Value()

		assert.Nil(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("Should compare values correctly", func(t *testing.T) {
		testCases := []struct {
			a        Monetary
			b        Monetary
			expected int
		}{
			{NewMonetaryFromFloat(10), NewMonetaryFromFloat(10), 0},
			{NewMonetaryFromFloat(10), NewMonetaryFromFloat(5), 1},
			{NewMonetaryFromFloat(5), NewMonetaryFromFloat(10), -1},
		}

		for _, tc := range testCases {
			result := tc.a.Compare(tc.b)

			assert.Equal(t, tc.expected, result,
				"Expected comparison of %s and %s to be %d, got %d",
				tc.a.String(), tc.b.String(), tc.expected, result)
		}
	})

	t.Run("Should check greater than or equal correctly", func(t *testing.T) {
		testCases := []struct {
			a        Monetary
			b        Monetary
			expected bool
		}{
			{NewMonetaryFromFloat(10), NewMonetaryFromFloat(10), true},
			{NewMonetaryFromFloat(10), NewMonetaryFromFloat(5), true},
			{NewMonetaryFromFloat(5), NewMonetaryFromFloat(10), false},
		}

		for _, tc := range testCases {
			result := tc.a.GreaterThanOrEqual(tc.b)

			assert.Equal(t, tc.expected, result,
				"Expected %s >= %s to be %v, got %v",
				tc.a.String(), tc.b.String(), tc.expected, result)
		}
	})

	t.Run("Should check less than or equal correctly", func(t *testing.T) {
		testCases := []struct {
			a        Monetary
			b        Monetary
			expected bool
		}{
			{NewMonetaryFromFloat(10), NewMonetaryFromFloat(10), true},
			{NewMonetaryFromFloat(10), NewMonetaryFromFloat(5), false},
			{NewMonetaryFromFloat(5), NewMonetaryFromFloat(10), true},
		}

		for _, tc := range testCases {
			result := tc.a.LessThanOrEqual(tc.b)

			assert.Equal(t, tc.expected, result,
				"Expected %s <= %s to be %v, got %v",
				tc.a.String(), tc.b.String(), tc.expected, result)
		}
	})
}
