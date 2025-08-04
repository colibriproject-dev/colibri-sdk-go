package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"strconv"
)

// NullInt32 for empty int32 field
type NullInt32 sql.NullInt32

// ParseNullInt32 converts string to int32
func ParseNullInt32(value string) (NullInt32, error) {
	parsedInt32, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return NullInt32{}, err
	}

	return NullInt32{Int32: int32(parsedInt32), Valid: true}, nil
}

// Scan converts sql driver value to null int32
func (t *NullInt32) Scan(value any) error {
	var i sql.NullInt32
	if err := i.Scan(value); err != nil {
		return err
	}

	if reflect.TypeOf(value) == nil {
		*t = NullInt32{i.Int32, false}
	} else {
		*t = NullInt32{i.Int32, true}
	}

	return nil
}

// Value converts null int32 to sql driver value
func (n NullInt32) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}

	return n.Int32, nil
}

// MarshalJSON converts null int32 to json int32 format
func (t NullInt32) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Int32)
}

// UnmarshalJSON converts json int32 to null int32
func (t *NullInt32) UnmarshalJSON(data []byte) error {
	var ptr *int32
	if err := json.Unmarshal(data, &ptr); err != nil {
		return err
	}

	if ptr != nil {
		t.Valid = true
		t.Int32 = *ptr
	} else {
		t.Valid = false
	}

	return nil
}
