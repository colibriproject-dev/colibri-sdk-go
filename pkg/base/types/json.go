package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JsonB for json binary field
type JsonB map[string]any

func (j *JsonB) Scan(value any) error {
	result, valid := value.([]byte)
	if !valid {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(result, &j)
}

func (j JsonB) Value() (driver.Value, error) {
	return json.Marshal(j)
}
