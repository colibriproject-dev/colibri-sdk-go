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

// ParseNullIsoDate converts string to NullIsoDate
func ParseNullIsoDate(value string) (NullIsoDate, error) {
	parsedDate, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return NullIsoDate{}, err
	}

	return NullIsoDate{Time: parsedDate, Valid: true}, nil
}

// Scan converts sql driver value to null iso date
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

// Value converts null iso date to sql driver value
func (t NullIsoDate) Value() (driver.Value, error) {
	if !t.Valid {
		return nil, nil
	}

	return t.Time, nil
}

// MarshalJSON converts null iso date to json iso date format
func (t NullIsoDate) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return json.Marshal(nil)
	}

	return json.Marshal(t.Time.Format(time.DateOnly))
}

// UnmarshalJSON converts json iso date to null iso date
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
