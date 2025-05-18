package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSON custom type for handling JSON data in database
type JSON map[string]interface{}

// Value implements the driver.Valuer interface for database/sql
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for database/sql
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSON)
		return nil
	}

	var byteData []byte
	switch v := value.(type) {
	case []byte:
		byteData = v
	case string:
		byteData = []byte(v)
	default:
		return fmt.Errorf("failed to scan JSON value: unsupported type %T", value)
	}

	var result map[string]interface{}
	err := json.Unmarshal(byteData, &result)
	if err != nil {
		return err
	}

	*j = JSON(result)
	return nil
}

// MarshalJSON implements json.Marshaler interface
func (j JSON) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]interface{}(j))
}

// UnmarshalJSON implements json.Unmarshaler interface
func (j *JSON) UnmarshalJSON(data []byte) error {
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	*j = JSON(temp)
	return nil
}
