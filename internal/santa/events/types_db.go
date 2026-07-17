package events

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type signingChainColumn []signingChainEntry

type processChainColumn []Process

func (v *signingChainColumn) Scan(src any) error          { return scanJSONSlice(src, v) }
func (v signingChainColumn) Value() (driver.Value, error) { return jsonSliceValue(v) }

func (v *processChainColumn) Scan(src any) error          { return scanJSONSlice(src, v) }
func (v processChainColumn) Value() (driver.Value, error) { return jsonSliceValue(v) }

func scanJSONSlice(src, dst any) error {
	switch data := src.(type) {
	case nil:
		return json.Unmarshal([]byte("[]"), dst)
	case []byte:
		if len(data) == 0 {
			return json.Unmarshal([]byte("[]"), dst)
		}
		return json.Unmarshal(data, dst)
	case string:
		if data == "" {
			return json.Unmarshal([]byte("[]"), dst)
		}
		return json.Unmarshal([]byte(data), dst)
	default:
		return fmt.Errorf("events: cannot scan %T into %T", src, dst)
	}
}

func jsonSliceValue[T any](values []T) (driver.Value, error) {
	if values == nil {
		values = []T{}
	}
	return json.Marshal(values)
}
