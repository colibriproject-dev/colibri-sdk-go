package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsoDate(t *testing.T) {
	t.Run("Should get parsed iso date", func(t *testing.T) {
		expected := IsoDate(time.Date(2022, time.January, 30, 0, 0, 0, 0, time.UTC))

		result, err := ParseIsoDate("2022-01-30")

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expected, result)
	})

	t.Run("Should return error when parse with a invalid string", func(t *testing.T) {
		result, err := ParseIsoDate("invalid")

		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, IsoDate(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), result)
	})

	t.Run("Should get string iso date", func(t *testing.T) {
		expected := "2022-01-30"

		result := IsoDate(time.Date(2022, time.January, 30, 0, 0, 0, 0, time.UTC)).String()

		assert.NotNil(t, result)
		assert.Equal(t, expected, result)
	})

	t.Run("Should get go string iso date", func(t *testing.T) {
		expected := "time.Date(2022, time.January, 30, 0, 0, 0, 0, time.UTC)"

		result := IsoDate(time.Date(2022, time.January, 30, 0, 0, 0, 0, time.UTC)).GoString()

		assert.NotNil(t, result)
		assert.Equal(t, expected, result)
	})

	t.Run("Should get go string iso date ignoring timestamp", func(t *testing.T) {
		expected := "time.Date(2022, time.January, 30, 0, 0, 0, 0, time.UTC)"

		result := IsoDate(time.Date(2022, time.January, 30, 1, 3, 5, 7, time.UTC)).GoString()

		assert.NotNil(t, result)
		assert.Equal(t, expected, result)
	})

	t.Run("Should get value with a valid value", func(t *testing.T) {
		expected := time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)

		result, err := IsoDate(expected).Value()

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expected, result)
	})

	t.Run("Should get json value with a valid value", func(t *testing.T) {
		expected := IsoDate(time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC))

		json, err := expected.MarshalJSON()
		result := string(json)

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "\"2022-01-01\"", result)
	})

	t.Run("Should get value with a valid json", func(t *testing.T) {
		var result IsoDate
		err := result.UnmarshalJSON([]byte("\"2022-01-01\""))

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, IsoDate(time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)), result)
	})

	t.Run("Should return error when get value with a invalid json", func(t *testing.T) {
		var result IsoDate
		err := result.UnmarshalJSON([]byte("invalid"))

		assert.NotNil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, IsoDate(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)), result)
	})

	t.Run("should return the time", func(t *testing.T) {
		value := IsoDate(time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)).Time()

		assert.Equal(t, time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC), value)
	})
	t.Run("should return the time with only the date", func(t *testing.T) {
		value := IsoDate(time.Date(2022, time.January, 1, 12, 2, 3, 3, time.UTC)).Time()

		assert.Equal(t, time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC), value)
	})

	t.Run("should check if its before when its", func(t *testing.T) {
		assert.False(t, MustParseIsoDate("2022-01-02").Before(MustParseIsoDate("2022-01-01")))
	})

	t.Run("should check if its before when it isnt", func(t *testing.T) {
		assert.True(t, MustParseIsoDate("2022-01-01").Before(MustParseIsoDate("2022-01-02")))
	})

	t.Run("should check as false if its equal", func(t *testing.T) {
		assert.False(t, MustParseIsoDate("2022-01-02").Before(MustParseIsoDate("2022-01-02")))
	})

	t.Run("should check as true if is after", func(t *testing.T) {
		assert.True(t, MustParseIsoDate("2022-01-02").After(MustParseIsoDate("2022-01-01")))
	})

	t.Run("should check as false if is not after", func(t *testing.T) {
		assert.False(t, MustParseIsoDate("2022-01-01").After(MustParseIsoDate("2022-01-02")))
	})

	t.Run("should check as false if is not after cause is equals", func(t *testing.T) {
		assert.False(t, MustParseIsoDate("2022-01-01").After(MustParseIsoDate("2022-01-02")))
	})

	t.Run("should check as false if is not equals", func(t *testing.T) {
		assert.False(t, MustParseIsoDate("2022-01-01").Equal(MustParseIsoDate("2022-01-02")))
	})

	t.Run("should check as false if is equals", func(t *testing.T) {
		assert.True(t, MustParseIsoDate("2022-01-01").Equal(MustParseIsoDate("2022-01-01")))
	})

	t.Run("should check as true if is 0", func(t *testing.T) {
		assert.True(t, MustParseIsoDate(time.Time{}.Format(time.DateOnly)).IsZero())
	})

	t.Run("should check as false if is not 0", func(t *testing.T) {
		assert.False(t, MustParseIsoDate(time.Now().Format(time.DateOnly)).IsZero())
	})

	t.Run("should returns 0 if its equal", func(t *testing.T) {
		assert.Zero(t, MustParseIsoDate("2022-01-01").Compare(MustParseIsoDate("2022-01-01")))
	})

	t.Run("should returns 1 if its before", func(t *testing.T) {
		assert.Positive(t, MustParseIsoDate("2022-01-02").Compare(MustParseIsoDate("2022-01-01")))
	})

	t.Run("should returns -1 if its before", func(t *testing.T) {
		assert.Negative(t, MustParseIsoDate("2022-01-01").Compare(MustParseIsoDate("2022-01-02")))
	})

	t.Run("should add date", func(t *testing.T) {
		assert.Equal(t, MustParseIsoDate("2022-01-02"), MustParseIsoDate("2022-01-01").AddDate(0, 0, 1))
	})

	t.Run("should subtract date", func(t *testing.T) {
		assert.Equal(t, MustParseIsoDate("2022-01-01"), MustParseIsoDate("2022-01-02").AddDate(0, 0, -1))
	})

	t.Run("should convert time.Time to IsoDate", func(t *testing.T) {
		timeValue := time.Date(2022, time.February, 15, 10, 20, 30, 0, time.UTC)
		expected := MustParseIsoDate("2022-02-15")

		var isoDate IsoDate
		result := isoDate.FromTime(timeValue)

		assert.Equal(t, expected, result)
	})

	t.Run("should convert time.Time to IsoDate ignoring time part", func(t *testing.T) {
		timeValue := time.Date(2022, time.March, 20, 23, 59, 59, 999, time.UTC)
		expected := MustParseIsoDate("2022-03-20")

		var isoDate IsoDate
		result := isoDate.FromTime(timeValue)

		assert.Equal(t, expected, result)
	})

	t.Run("should convert DateTime to IsoDate", func(t *testing.T) {
		dateTime := DateTime(time.Date(2022, time.April, 10, 12, 30, 45, 0, time.UTC))
		expected := MustParseIsoDate("2022-04-10")

		var isoDate IsoDate
		result := isoDate.FromDateTime(dateTime)

		assert.Equal(t, expected, result)
	})

	t.Run("should convert DateTime to IsoDate ignoring time part", func(t *testing.T) {
		dateTime := DateTime(time.Date(2022, time.May, 5, 18, 45, 30, 0, time.UTC))
		expected := MustParseIsoDate("2022-05-05")

		var isoDate IsoDate
		result := isoDate.FromDateTime(dateTime)

		assert.Equal(t, expected, result)
	})
}
