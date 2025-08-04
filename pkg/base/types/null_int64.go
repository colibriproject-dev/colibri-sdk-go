package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"strconv"
)

// NullInt64 for empty int64 field
type NullInt64 sql.NullInt64

// ParseNullInt64 converts string to int64
func ParseNullInt64(value string) (NullInt64, error) {
	parsedInt64, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return NullInt64{}, err
	}

	return NullInt64{Int64: parsedInt64, Valid: true}, nil
}

// Scan converts sql driver value to null int64
func (t *NullInt64) Scan(value any) error {
	var i sql.NullInt64
	if err := i.Scan(value); err != nil {
		return err
	}

	if reflect.TypeOf(value) == nil {
		*t = NullInt64{i.Int64, false}
	} else {
		*t = NullInt64{i.Int64, true}
	}

	return nil
}

// Value converts null int64 to sql driver value
func (n NullInt64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}

	return n.Int64, nil
}

// MarshalJSON converts null int64 to json int64 format
func (t NullInt64) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Int64)
}

// UnmarshalJSON converts json int64 to null int64
func (t *NullInt64) UnmarshalJSON(data []byte) error {
	var ptr *int64
	if err := json.Unmarshal(data, &ptr); err != nil {
		return err
	}

	if ptr != nil {
		t.Valid = true
		t.Int64 = *ptr
	} else {
		t.Valid = false
	}

	return nil
}
