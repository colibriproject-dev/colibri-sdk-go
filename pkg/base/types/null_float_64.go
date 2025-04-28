package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
)

// NullFloat64 for empty float field
type NullFloat64 sql.NullFloat64

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

func (n NullFloat64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}

	return n.Float64, nil
}

func (t NullFloat64) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Float64)
}

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
