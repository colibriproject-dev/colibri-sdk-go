package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"time"
)

// NullDateTime for empty date/time field
type NullDateTime sql.NullTime

// ParseNullDateTime converts string to NullDateTime
func ParseNullDateTime(value string) (NullDateTime, error) {
	parsedDateTime, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return NullDateTime{}, err
	}

	return NullDateTime{Time: parsedDateTime, Valid: true}, nil
}

// Scan converts sql driver value to null date time
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

// Value converts null date time to sql driver value
func (n NullDateTime) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}

	return n.Time, nil
}

// MarshalJSON converts null date time to json date time format
func (t NullDateTime) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Time)
}

// UnmarshalJSON converts json date time to null date time
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
