package types

import (
	"database/sql/driver"
	"strings"

	"github.com/shopspring/decimal"
)

// Monetary for value field
type Monetary struct{ decimal.Decimal }

// ParseMonetary converts string to monetary
func ParseMonetary(value string) (Monetary, error) {
	parsedDecimal, err := decimal.NewFromString(value)
	if err != nil {
		return Monetary{}, err
	}

	return Monetary{parsedDecimal}, nil
}

// Value converts monetary value to sql driver value
func (m Monetary) Value() (driver.Value, error) {
	return m.Decimal, nil
}

// Add creates a new Monetary by adding incoming value to current value
func (m Monetary) Add(value Monetary) Monetary {
	result := Monetary(m)
	result.Decimal = m.Decimal.Add(value.Decimal)
	return result
}

// Sub creates a new Monetary by subtracting incoming value from current value
func (m Monetary) Sub(value Monetary) Monetary {
	result := Monetary(m)
	result.Decimal = m.Decimal.Sub(value.Decimal)
	return result
}

// Mul creates a new Monetary by multiplying current value with incoming value
func (m Monetary) Mul(value Monetary) Monetary {
	result := Monetary(m)
	result.Decimal = m.Decimal.Mul(value.Decimal)
	return result
}

// Div creates a new Monetary by dividing current value by incoming value
func (m Monetary) Div(value Monetary) Monetary {
	result := Monetary(m)
	result.Decimal = m.Decimal.Div(value.Decimal)
	return result
}

// Compare compares current value with incoming value
func (m Monetary) Compare(value Monetary) int {
	return m.Decimal.Compare(value.Decimal)
}

// Equal checks if current value is equal to incoming value
func (m Monetary) Equal(value Monetary) bool {
	return m.Decimal.Equal(value.Decimal)
}

// GreaterThanOrEqual checks if current value is greater than or equal to incoming value
func (m Monetary) GreaterThanOrEqual(value Monetary) bool {
	return m.Decimal.GreaterThanOrEqual(value.Decimal)
}

// LessThanOrEqual checks if current value is less than or equal to incoming value
func (m Monetary) LessThanOrEqual(value Monetary) bool {
	return m.Decimal.LessThanOrEqual(value.Decimal)
}

// Cents returns the monetary value as cents (integer)
func (m Monetary) Cents() int64 {
	cents := m.Decimal.Mul(decimal.NewFromInt(100))
	return cents.Round(0).IntPart()
}

// NewMonetaryFromDecimal creates a new Monetary from a decimal.Decimal
func NewMonetaryFromDecimal(value decimal.Decimal) Monetary {
	return Monetary{
		Decimal: value,
	}
}

// NewMonetaryFromFloat creates a new Monetary from a float64
func NewMonetaryFromFloat(value float64) Monetary {
	decimal := decimal.NewFromFloat(value)
	return Monetary{
		Decimal: decimal,
	}
}

// NewMonetaryFromCents creates a new Monetary from an integer cent value
func NewMonetaryFromCents(cents int64) Monetary {
	decimal := decimal.NewFromInt(cents).Div(decimal.NewFromInt(100))
	return Monetary{
		Decimal: decimal,
	}
}

// MarshalJSON converts the Monetary value to a JSON string format
func (m Monetary) MarshalJSON() ([]byte, error) {
	// Get the raw string representation
	str := m.String()

	minDP := 2

	parts := strings.Split(str, ".")
	if len(parts) == 1 {
		// No decimal point
		str += "." + strings.Repeat("0", minDP)
	} else if len(parts[1]) < minDP {
		// Less than minimum decimal places
		str += strings.Repeat("0", minDP-len(parts[1]))
	}
	return []byte(`"` + str + `"`), nil
}

// UnmarshalJSON converts a JSON string or float to a Monetary value
func (m *Monetary) UnmarshalJSON(data []byte) error {
	var value decimal.Decimal
	if err := value.UnmarshalJSON(data); err != nil {
		return err
	}

	m.Decimal = value

	return nil
}
