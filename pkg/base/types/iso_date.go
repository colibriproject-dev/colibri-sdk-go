package types

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// IsoDate struct
type IsoDate time.Time

// ParseIsoDate converts string to iso date
func ParseIsoDate(value string) (IsoDate, error) {
	parsedDate, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return IsoDate{}, err
	}

	return IsoDate(parsedDate), nil
}

// MustParseIsoDate converts string to iso date and panics if error occurs
func MustParseIsoDate(value string) IsoDate {
	parsedDate, err := time.Parse(time.DateOnly, value)
	if err != nil {
		panic(err)
	}

	return IsoDate(parsedDate)
}

// Value converts iso date to sql driver value
func (t IsoDate) Value() (driver.Value, error) {
	return time.Time(t), nil
}

// String returns the iso date formatted using the format string
func (t IsoDate) String() string {
	return time.Time(t).Format(time.DateOnly)
}

// GoString returns the iso date in Go source code format string
func (t IsoDate) GoString() string {
	return time.Time(MustParseIsoDate(t.String())).GoString()
}

// MarshalJSON converts iso date to json string format
func (t IsoDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(time.DateOnly))
}

// UnmarshalJSON converts json string to iso date
func (t *IsoDate) UnmarshalJSON(data []byte) error {
	var ptr *string
	if err := json.Unmarshal(data, &ptr); err != nil {
		return err
	}

	if ptr == nil {
		return nil
	}

	parsedDate, err := time.Parse(time.DateOnly, *ptr)
	if err != nil {
		return err
	}

	*t = IsoDate(parsedDate)
	return nil
}

// Time returns the iso date as time.Time
func (i IsoDate) Time() time.Time {
	return time.Time(MustParseIsoDate(i.String()))
}

// Before compares two iso dates
func (i IsoDate) Before(i2 IsoDate) bool {
	return i.Time().Before(i2.Time())
}

// After compares two iso dates
func (i IsoDate) After(i2 IsoDate) bool {
	return i.Time().After(i2.Time())
}

// Equal compares two iso dates
func (i IsoDate) Equal(i2 IsoDate) bool {
	return i.Time().Equal(i2.Time())
}

// IsZero checks if the iso date is zero
func (i IsoDate) IsZero() bool {
	return i.Time().IsZero()
}

// Compare compares two iso dates
func (i IsoDate) Compare(i2 IsoDate) int {
	return i.Time().Compare(i2.Time())
}

// AddDate adds years, months, and days to the iso date
func (i IsoDate) AddDate(years int, months int, days int) IsoDate {
	return IsoDate(i.Time().AddDate(years, months, days))
}

// FromTime converts time.Time to iso date
func (i IsoDate) FromTime(t time.Time) IsoDate {
	return IsoDate(MustParseIsoDate(t.Format(time.DateOnly)))
}

// FromDateTime converts DateTime to iso date
func (i IsoDate) FromDateTime(t DateTime) IsoDate {
	return i.FromTime(t.Time())
}
