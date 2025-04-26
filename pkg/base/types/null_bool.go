package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
)

// NullBoll for empty boolean field
type NullBool sql.NullBool

func (t *NullBool) Scan(value any) error {
	var i sql.NullBool
	if err := i.Scan(value); err != nil {
		return err
	}

	if reflect.TypeOf(value) == nil {
		*t = NullBool{i.Bool, false}
	} else {
		*t = NullBool{i.Bool, true}
	}

	return nil
}

func (n NullBool) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}

	return n.Bool, nil
}

func (t NullBool) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Bool)
}

func (t *NullBool) UnmarshalJSON(data []byte) error {
	var ptr *bool
	if err := json.Unmarshal(data, &ptr); err != nil {
		return err
	}

	if ptr != nil {
		t.Valid = true
		t.Bool = *ptr
	} else {
		t.Valid = false
	}

	return nil
}
