package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
)

// NullInt64 for an empty int64 field
type NullInt64 sql.NullInt64

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

func (n NullInt64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}

	return n.Int64, nil
}

func (t NullInt64) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Int64)
}

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
