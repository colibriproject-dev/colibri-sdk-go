package types

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// IsoTime struct
type IsoTime time.Time

// ParseIsoTime converts string to iso time.
//
// It takes a string value as an input.
// Returns IsoTime and an error.
func ParseIsoTime(value string) (IsoTime, error) {
	parsedTime, err := time.Parse(time.TimeOnly, value)
	if err != nil {
		return IsoTime{}, err
	}

	return IsoTime(parsedTime), nil
}

// Value converts iso time to SQL driver value.
//
// Returns driver.Value and an error.
func (t IsoTime) Value() (driver.Value, error) {
	return time.Time(t), nil
}

// String returns the iso time formatted using the format string
//
// No parameters.
// Returns a string.
func (t IsoTime) String() string {
	return time.Time(t).Format(time.TimeOnly)
}

// GoString returns the iso time in Go source code format string.
//
// No parameters.
// Returns a string.
func (t IsoTime) GoString() string {
	return time.Time(t).GoString()
}

// MarshalJSON converts iso time to JSON string format.
//
// No parameters.
// Returns a byte slice and an error.
func (t IsoTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(time.TimeOnly))
}

// UnmarshalJSON converts JSON string to iso time
//
// It takes a byte slice as the input data.
// Returns an error.
func (t *IsoTime) UnmarshalJSON(data []byte) error {
	var ptr *string
	if err := json.Unmarshal(data, &ptr); err != nil {
		return err
	}

	if ptr == nil {
		return nil
	}

	parsedTime, err := time.Parse(time.TimeOnly, *ptr)
	if err != nil {
		return err
	}

	*t = IsoTime(parsedTime)
	return nil
}
