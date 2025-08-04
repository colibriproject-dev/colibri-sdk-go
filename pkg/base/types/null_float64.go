package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"strconv"
)

// NullFloat64 for empty float field
type NullFloat64 sql.NullFloat64

// ParseNullFloat64 converts string to NullFloat64
func ParseNullFloat64(value string) (NullFloat64, error) {
	parsedFloat64, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return NullFloat64{}, err
	}

	return NullFloat64{Float64: parsedFloat64, Valid: true}, nil
}

// Scan converts sql driver value to null float64
func (t *NullFloat64) Scan(value any) error {
	var i sql.NullFloat64
	if err := i.Scan(value); err != nil {
		return err
	}

	if reflect.TypeOf(value) == nil {
		*t = NullFloat64{i.Float64, false}
	} else {
		*t = NullFloat64{i.Float64, true}
	}

	return nil
}

// Value converts null float64 to sql driver value
func (n NullFloat64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}

	return n.Float64, nil
}

// MarshalJSON converts null float64 to json float64 format
func (t NullFloat64) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Float64)
}

// UnmarshalJSON converts json float64 to null float64
func (t *NullFloat64) UnmarshalJSON(data []byte) error {
	var ptr *float64
	if err := json.Unmarshal(data, &ptr); err != nil {
		return err
	}

	if ptr != nil {
		t.Valid = true
		t.Float64 = *ptr
	} else {
		t.Valid = false
	}

	return nil
}
