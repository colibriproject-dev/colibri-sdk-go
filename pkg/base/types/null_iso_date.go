package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"time"
)

// NullIsoDate for empty date field
type NullIsoDate sql.NullTime

func (t *NullIsoDate) Scan(value any) error {
	var i sql.NullTime
	if err := i.Scan(value); err != nil {
		return err
	}

	if reflect.TypeOf(value) == nil {
		*t = NullIsoDate{i.Time, false}
	} else {
		*t = NullIsoDate{i.Time, true}
	}

	return nil
}

func (t NullIsoDate) Value() (driver.Value, error) {
	if !t.Valid {
		return nil, nil
	}

	return t.Time, nil
}

func (t NullIsoDate) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Time.Format(time.DateOnly))
}

func (t *NullIsoDate) UnmarshalJSON(data []byte) error {
	var ptr *string
	if err := json.Unmarshal(data, &ptr); err != nil {
		return err
	}

	if ptr == nil {
		t.Valid = false
		return nil
	}

	parsedDate, err := time.Parse(time.DateOnly, *ptr)
	if err != nil {
		return err
	}

	t.Time, t.Valid = parsedDate, true
	return nil
}
