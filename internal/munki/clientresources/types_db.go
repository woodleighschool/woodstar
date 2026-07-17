package clientresources

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type linksValue []Link

func (v *linksValue) Scan(src any) error {
	switch data := src.(type) {
	case nil:
		return json.Unmarshal([]byte("[]"), v)
	case []byte:
		return json.Unmarshal(data, v)
	case string:
		return json.Unmarshal([]byte(data), v)
	default:
		return fmt.Errorf("clientresources: cannot scan %T into links", src)
	}
}

func (v linksValue) Value() (driver.Value, error) {
	if v == nil {
		v = linksValue{}
	}
	return json.Marshal(v)
}
