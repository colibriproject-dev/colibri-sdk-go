package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"time"
)

// NullDateTime for an empty date / time field
type NullDateTime sql.NullTime

func (t *NullDateTime) Scan(value any) error {
	var i sql.NullTime
	if err := i.Scan(value); err != nil {
		return err
	}

	if reflect.TypeOf(value) == nil {
		*t = NullDateTime{i.Time, false}
	} else {
		*t = NullDateTime{i.Time, true}
	}

	return nil
}

func (n NullDateTime) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}

	return n.Time, nil
}

func (t NullDateTime) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Time)
}

func (t *NullDateTime) UnmarshalJSON(data []byte) error {
	var ptr *time.Time
	if err := json.Unmarshal(data, &ptr); err != nil {
		return err
	}

	if ptr != nil {
		t.Valid = true
		t.Time = *ptr
	} else {
		t.Valid = false
	}

	return nil
}
