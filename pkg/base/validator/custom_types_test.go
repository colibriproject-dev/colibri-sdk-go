package validator

import (
	"testing"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCustomTypes(t *testing.T) {
	Initialize()

	t.Run("Should parse UUID", func(t *testing.T) {
		values := map[string][]string{
			"id": {"f47ac10b-58cc-0372-8567-0e02b2c3d479"},
		}
		type TestForm struct {
			ID uuid.UUID `form:"id"`
		}
		obj := TestForm{}

		err := FormDecode(&obj, values)

		assert.NoError(t, err)
		assert.Equal(t, "f47ac10b-58cc-0372-8567-0e02b2c3d479", obj.ID.String())
	})

	t.Run("Should parse DateTime", func(t *testing.T) {
		values := map[string][]string{
			"date_time": {"2022-01-30T10:20:30Z"},
		}
		type TestForm struct {
			DateTime types.DateTime `form:"date_time"`
		}
		obj := TestForm{}

		err := FormDecode(&obj, values)

		assert.NoError(t, err)
		assert.Equal(t, "2022-01-30T10:20:30Z", obj.DateTime.String())
	})

	t.Run("Should parse IsoDate", func(t *testing.T) {
		values := map[string][]string{
			"date": {"2022-01-30"},
		}
		type TestForm struct {
			Date types.IsoDate `form:"date"`
		}
		obj := TestForm{}

		err := FormDecode(&obj, values)

		assert.NoError(t, err)
		assert.Equal(t, "2022-01-30", obj.Date.String())
	})

	t.Run("Should parse IsoTime", func(t *testing.T) {
		values := map[string][]string{
			"time": {"14:30:15"},
		}
		type TestForm struct {
			Time types.IsoTime `form:"time"`
		}
		obj := TestForm{}

		err := FormDecode(&obj, values)

		assert.NoError(t, err)
		assert.Equal(t, "14:30:15", obj.Time.String())
	})

	t.Run("Should parse Monetary", func(t *testing.T) {
		values := map[string][]string{
			"amount": {"123.45"},
		}
		type TestForm struct {
			Amount types.Monetary `form:"amount"`
		}
		obj := TestForm{}

		err := FormDecode(&obj, values)

		assert.NoError(t, err)
		assert.Equal(t, "123.45", obj.Amount.String())
	})

	t.Run("Should parse NullBool", func(t *testing.T) {
		values := map[string][]string{
			"active": {"true"},
		}
		type TestForm struct {
			Active types.NullBool `form:"active"`
		}
		obj := TestForm{}

		err := FormDecode(&obj, values)

		assert.NoError(t, err)
		assert.True(t, obj.Active.Valid)
		assert.True(t, obj.Active.Bool)
	})

	t.Run("Should parse NullDateTime", func(t *testing.T) {
		values := map[string][]string{
			"created_at": {"2022-01-30T10:20:30Z"},
		}
		type TestForm struct {
			CreatedAt types.NullDateTime `form:"created_at"`
		}
		obj := TestForm{}

		err := FormDecode(&obj, values)

		assert.NoError(t, err)
		assert.True(t, obj.CreatedAt.Valid)
		assert.Equal(t, "2022-01-30 10:20:30 +0000 UTC", obj.CreatedAt.Time.String())
	})

	t.Run("Should parse NullFloat64", func(t *testing.T) {
		values := map[string][]string{
			"rate": {"42.5"},
		}
		type TestForm struct {
			Rate types.NullFloat64 `form:"rate"`
		}
		obj := TestForm{}

		err := FormDecode(&obj, values)

		assert.NoError(t, err)
		assert.True(t, obj.Rate.Valid)
		assert.Equal(t, 42.5, obj.Rate.Float64)
	})

	t.Run("Should parse NullInt16", func(t *testing.T) {
		values := map[string][]string{
			"code": {"42"},
		}
		type TestForm struct {
			Code types.NullInt16 `form:"code"`
		}
		obj := TestForm{}

		err := FormDecode(&obj, values)

		assert.NoError(t, err)
		assert.True(t, obj.Code.Valid)
		assert.Equal(t, int16(42), obj.Code.Int16)
	})

	t.Run("Should parse NullInt32", func(t *testing.T) {
		values := map[string][]string{
			"code": {"42"},
		}
		type TestForm struct {
			Code types.NullInt32 `form:"code"`
		}
		obj := TestForm{}

		err := FormDecode(&obj, values)

		assert.NoError(t, err)
		assert.True(t, obj.Code.Valid)
		assert.Equal(t, int32(42), obj.Code.Int32)
	})

	t.Run("Should parse NullInt64", func(t *testing.T) {
		values := map[string][]string{
			"code": {"42"},
		}
		type TestForm struct {
			Code types.NullInt64 `form:"code"`
		}
		obj := TestForm{}

		err := FormDecode(&obj, values)

		assert.NoError(t, err)
		assert.True(t, obj.Code.Valid)
		assert.Equal(t, int64(42), obj.Code.Int64)
	})

	t.Run("Should parse NullIsoDate", func(t *testing.T) {
		values := map[string][]string{
			"birth_date": {"2022-01-30"},
		}
		type TestForm struct {
			BirthDate types.NullIsoDate `form:"birth_date"`
		}
		obj := TestForm{}

		err := FormDecode(&obj, values)

		assert.NoError(t, err)
		assert.True(t, obj.BirthDate.Valid)
		assert.Equal(t, "2022-01-30 00:00:00 +0000 UTC", obj.BirthDate.Time.String())
	})

	t.Run("Should parse NullIsoTime", func(t *testing.T) {
		values := map[string][]string{
			"alarm_time": {"14:30:15"},
		}
		type TestForm struct {
			AlarmTime types.NullIsoTime `form:"alarm_time"`
		}
		obj := TestForm{}

		err := FormDecode(&obj, values)

		assert.NoError(t, err)
		assert.True(t, obj.AlarmTime.Valid)
		assert.Equal(t, "0000-01-01 14:30:15 +0000 UTC", obj.AlarmTime.Time.String())
	})
}
