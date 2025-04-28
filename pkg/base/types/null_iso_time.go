package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"time"
)

// NullIsoTime for empty time field
type NullIsoTime sql.NullTime

func (t *NullIsoTime) Scan(value any) error {
	var i sql.NullTime
	if err := i.Scan(value); err != nil {
		return err
	}

	if reflect.TypeOf(value) == nil {
		*t = NullIsoTime{i.Time, false}
	} else {
		*t = NullIsoTime{i.Time, true}
	}

	return nil
}

func (t NullIsoTime) Value() (driver.Value, error) {
	if !t.Valid {
		return nil, nil
	}

	return t.Time, nil
}

func (t NullIsoTime) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Time.Format(time.TimeOnly))
}

func (t *NullIsoTime) UnmarshalJSON(data []byte) error {
	var ptr *string
	if err := json.Unmarshal(data, &ptr); err != nil {
		return err
	}

	if ptr == nil {
		t.Valid = false
		return nil
	}

	parsedTime, err := time.Parse(time.TimeOnly, *ptr)
	if err != nil {
		return err
	}

	t.Time, t.Valid = parsedTime, true
	return nil
}
