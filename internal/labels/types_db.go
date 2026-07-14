package labels

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// labelCriteria scans the nullable criteria jsonb column inline and marshals the
// normalized criteria back out. A pointer receiver is needed for sql.Scanner (it
// sets the wrapped value) and a value receiver for driver.Valuer (the write
// struct holds it as a non-addressable field), so recvcheck is suppressed.
//
//nolint:recvcheck // Scanner needs a pointer receiver; Valuer needs a value receiver.
type labelCriteria struct {
	value *Criteria
}

func (c *labelCriteria) Scan(src any) error {
	c.value = nil
	switch data := src.(type) {
	case nil:
		return nil
	case []byte:
		return c.unmarshal(data)
	case string:
		return c.unmarshal([]byte(data))
	default:
		return fmt.Errorf("labels: cannot scan %T into criteria", src)
	}
}

func (c *labelCriteria) unmarshal(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var criteria Criteria
	if err := json.Unmarshal(data, &criteria); err != nil {
		return fmt.Errorf("decode label criteria: %w", err)
	}
	c.value = &criteria
	return nil
}

func (c labelCriteria) Value() (driver.Value, error) {
	if c.value == nil {
		return nil, nil
	}
	normalized := Criteria{Attribute: c.value.Attribute, Values: normalizeCriteriaValues(c.value.Values)}
	return json.Marshal(normalized)
}
