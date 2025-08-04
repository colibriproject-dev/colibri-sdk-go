package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"strconv"
)

// NullInt16 for empty int16 field
type NullInt16 sql.NullInt16

// ParseNullInt16 converts string to NullInt16
func ParseNullInt16(value string) (NullInt16, error) {
	parsedInt16, err := strconv.ParseInt(value, 10, 16)
	if err != nil {
		return NullInt16{}, err
	}

	return NullInt16{Int16: int16(parsedInt16), Valid: true}, nil
}

// Scan converts sql driver value to null int16
func (t *NullInt16) Scan(value any) error {
	var i sql.NullInt16
	if err := i.Scan(value); err != nil {
		return err
	}

	if reflect.TypeOf(value) == nil {
		*t = NullInt16{i.Int16, false}
	} else {
		*t = NullInt16{i.Int16, true}
	}

	return nil
}

// Value converts null int16 to sql driver value
func (n NullInt16) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}

	return n.Int16, nil
}

// MarshalJSON converts null int16 to json int16 format
func (t NullInt16) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Int16)
}

// UnmarshalJSON converts json int16 to null int16
func (t *NullInt16) UnmarshalJSON(data []byte) error {
	var ptr *int16
	if err := json.Unmarshal(data, &ptr); err != nil {
		return err
	}

	if ptr != nil {
		t.Valid = true
		t.Int16 = *ptr
	} else {
		t.Valid = false
	}

	return nil
}
