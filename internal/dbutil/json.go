package dbutil

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONSlice stores a slice in a PostgreSQL JSON column. SQL NULL, empty
// driver values, and JSON null all scan as an empty, non-nil slice. A nil Go
// slice is written as an empty JSON array rather than SQL NULL or JSON null.
type JSONSlice[T any] []T

func (s *JSONSlice[T]) Scan(src any) error {
	var data []byte
	switch value := src.(type) {
	case nil:
		*s = JSONSlice[T]{}
		return nil
	case []byte:
		data = value
	case string:
		data = []byte(value)
	default:
		return fmt.Errorf("cannot scan %T into JSON slice", src)
	}

	if len(data) == 0 {
		*s = JSONSlice[T]{}
		return nil
	}

	var values JSONSlice[T]
	if err := json.Unmarshal(data, &values); err != nil {
		return fmt.Errorf("decode JSON slice: %w", err)
	}
	if values == nil {
		values = JSONSlice[T]{}
	}
	*s = values
	return nil
}

func (s JSONSlice[T]) Value() (driver.Value, error) {
	if s == nil {
		s = JSONSlice[T]{}
	}
	return json.Marshal(s)
}
