package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
)

// NullInt32 for empty int32 field
type NullInt32 sql.NullInt32

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

func (n NullInt32) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}

	return n.Int32, nil
}

func (t NullInt32) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Int32)
}

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
