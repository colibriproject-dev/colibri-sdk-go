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

// ParseNullIsoTime converts string to NullIsoTime
func ParseNullIsoTime(value string) (NullIsoTime, error) {
	parsedTime, err := time.Parse(time.TimeOnly, value)
	if err != nil {
		return NullIsoTime{}, err
	}

	return NullIsoTime{Time: parsedTime, Valid: true}, nil
}

// Scan converts sql driver value to null iso time
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

// Value converts null iso time to sql driver value
func (t NullIsoTime) Value() (driver.Value, error) {
	if !t.Valid {
		return nil, nil
	}

	return t.Time, nil
}

// MarshalJSON converts null iso time to json iso time format
func (t NullIsoTime) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Time.Format(time.TimeOnly))
}

// UnmarshalJSON converts json iso time to null iso time
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
