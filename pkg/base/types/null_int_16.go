package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
)

// NullInt16 for empty int16 field
type NullInt16 sql.NullInt16

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

func (n NullInt16) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}

	return n.Int16, nil
}

func (t NullInt16) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Int16)
}

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
