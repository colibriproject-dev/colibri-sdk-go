package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDateTime(t *testing.T) {
	t.Run("Should get parsed iso date time", func(t *testing.T) {
		expected := DateTime(time.Date(2022, time.January, 30, 10, 20, 30, 0, time.UTC))

		result, err := ParseDateTime("2022-01-30T10:20:30Z")

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expected, result)
	})

	t.Run("Should return error when parse with a invalid string", func(t *testing.T) {
		result, err := ParseDateTime("invalid")

		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, DateTime(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), result)
	})

	t.Run("Should get string iso date time", func(t *testing.T) {
		expected := "2022-01-30T08:10:20Z"

		result := DateTime(time.Date(2022, time.January, 30, 8, 10, 20, 0, time.UTC)).String()

		assert.NotNil(t, result)
		assert.Equal(t, expected, result)
	})

	t.Run("Should get go string iso date time", func(t *testing.T) {
		expected := "time.Date(2022, time.January, 30, 9, 5, 20, 0, time.UTC)"

		result := DateTime(time.Date(2022, time.January, 30, 9, 5, 20, 0, time.UTC)).GoString()

		assert.NotNil(t, result)
		assert.Equal(t, expected, result)
	})

	t.Run("Should get value with a valid value", func(t *testing.T) {
		expected := time.Date(2022, time.January, 1, 13, 40, 11, 0, time.UTC)

		result, err := DateTime(expected).Value()

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expected, result)
	})

	t.Run("Should get json value with a valid value", func(t *testing.T) {
		expected := DateTime(time.Date(2022, time.January, 1, 20, 55, 11, 0, time.UTC))

		json, err := expected.MarshalJSON()
		result := string(json)

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "\"2022-01-01T20:55:11Z\"", result)
	})

	t.Run("Should get value with a valid json", func(t *testing.T) {
		var result DateTime
		err := result.UnmarshalJSON([]byte("\"2022-01-01T23:30:22Z\""))

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, DateTime(time.Date(2022, time.January, 1, 23, 30, 22, 0, time.UTC)), result)
	})

	t.Run("Should return error when get value with a invalid json", func(t *testing.T) {
		var result DateTime
		err := result.UnmarshalJSON([]byte("invalid"))

		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, DateTime(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), result)
	})

	t.Run("Should convert DateTime to time.Time", func(t *testing.T) {
		dt := DateTime(time.Date(2022, time.January, 1, 10, 20, 30, 0, time.UTC))
		expected := time.Date(2022, time.January, 1, 10, 20, 30, 0, time.UTC)

		result := dt.Time()

		assert.Equal(t, expected, result)
	})

	t.Run("Should add date correctly", func(t *testing.T) {
		dt := DateTime(time.Date(2022, time.January, 1, 10, 20, 30, 0, time.UTC))
		expected := DateTime(time.Date(2023, time.February, 4, 10, 20, 30, 0, time.UTC))

		result := dt.AddDate(1, 1, 3)

		assert.Equal(t, expected, result)
	})

	t.Run("Should correctly determine if a date is after another", func(t *testing.T) {
		earlier := DateTime(time.Date(2022, time.January, 1, 10, 20, 30, 0, time.UTC))
		later := DateTime(time.Date(2022, time.January, 2, 10, 20, 30, 0, time.UTC))

		assert.True(t, later.After(earlier))
		assert.False(t, earlier.After(later))
		assert.False(t, earlier.After(earlier))
	})

	t.Run("Should correctly determine if a date is before another", func(t *testing.T) {
		earlier := DateTime(time.Date(2022, time.January, 1, 10, 20, 30, 0, time.UTC))
		later := DateTime(time.Date(2022, time.January, 2, 10, 20, 30, 0, time.UTC))

		assert.True(t, earlier.Before(later))
		assert.False(t, later.Before(earlier))
		assert.False(t, earlier.Before(earlier))
	})

	t.Run("Should correctly determine if dates are equal", func(t *testing.T) {
		date1 := DateTime(time.Date(2022, time.January, 1, 10, 20, 30, 0, time.UTC))
		date2 := DateTime(time.Date(2022, time.January, 1, 10, 20, 30, 0, time.UTC))
		date3 := DateTime(time.Date(2022, time.January, 2, 10, 20, 30, 0, time.UTC))

		assert.True(t, date1.Equal(date2))
		assert.False(t, date1.Equal(date3))
	})

	t.Run("Should convert time.Time to DateTime", func(t *testing.T) {
		timeValue := time.Date(2022, time.January, 1, 10, 20, 30, 0, time.UTC)
		expected := DateTime(timeValue)

		var dt DateTime
		result := dt.FromTime(timeValue)

		assert.Equal(t, expected, result)
	})

	t.Run("Should get current time", func(t *testing.T) {
		before := time.Now().Add(-1 * time.Second)

		var dt DateTime
		result := dt.Now()

		after := time.Now().Add(1 * time.Second)

		resultTime := time.Time(result)
		assert.True(t, resultTime.After(before))
		assert.True(t, resultTime.Before(after))
	})
}
