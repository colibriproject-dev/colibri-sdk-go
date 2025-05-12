package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

var (
	ErrInvalidValue = errors.New("invalid []byte value")
)

type JsonB map[string]any

func (t *JsonB) Scan(value any) error {
	result, valid := value.([]byte)
	if !valid {
		return ErrInvalidValue
	}

	return json.Unmarshal(result, &t)
}

func (t *JsonB) Value() (driver.Value, error) {
	return json.Marshal(t)
}
