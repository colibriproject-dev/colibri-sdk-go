package validator

import (
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCustomTypesRegistration(t *testing.T) {
	Initialize()

	// Test UUID custom type
	t.Run("UUID custom type", func(t *testing.T) {
		values := map[string][]string{
			"id": {"123e4567-e89b-12d3-a456-426614174000"},
		}

		type TestStruct struct {
			ID uuid.UUID `form:"id"`
		}

		var result TestStruct
		err := FormDecode(&result, values)
		assert.NoError(t, err)
		assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", result.ID.String())
	})

	// Test ISO Date custom type
	t.Run("ISO Date custom type", func(t *testing.T) {
		values := map[string][]string{
			"date": {"2024-03-20"},
		}

		type TestStruct struct {
			Date types.IsoDate `form:"date"`
		}

		var result TestStruct
		err := FormDecode(&result, values)
		assert.NoError(t, err)
		expectedDate, _ := time.Parse("2006-01-02", "2024-03-20")
		assert.Equal(t, expectedDate, time.Time(result.Date))
	})

	// Test ISO Time custom type
	t.Run("ISO Time custom type", func(t *testing.T) {
		values := map[string][]string{
			"time": {"15:04:05"},
		}

		type TestStruct struct {
			Time types.IsoTime `form:"time"`
		}

		var result TestStruct
		err := FormDecode(&result, values)
		assert.NoError(t, err)
		expectedTime, _ := time.Parse("15:04:05", "15:04:05")
		assert.Equal(t, expectedTime, time.Time(result.Time))
	})
}
