package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// NullJsonB for empty jsonb field
type NullJsonB map[string]any

// Scan converts sql driver value to null jsonb
func (j *NullJsonB) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}

	result, valid := value.([]byte)
	if !valid {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(result, &j)
}

// Value converts null jsonb to sql driver value
func (j NullJsonB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}

	return json.Marshal(j)
}
