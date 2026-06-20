package packages

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// The JSON column types below need a pointer receiver for sql.Scanner (it mutates
// the slice) and a value receiver for driver.Valuer (the struct fields holding them
// are non-addressable values), so recvcheck is suppressed.

//nolint:recvcheck // Scanner needs a pointer receiver; Valuer needs a value receiver.
type packageInstallItems []PackageInstallItem

//nolint:recvcheck // Scanner needs a pointer receiver; Valuer needs a value receiver.
type packageReceipts []PackageReceipt

//nolint:recvcheck // Scanner needs a pointer receiver; Valuer needs a value receiver.
type packageItemsToCopy []PackageItemToCopy

//nolint:recvcheck // Scanner needs a pointer receiver; Valuer needs a value receiver.
type packageInstallerChoices []PackageInstallerChoice

//nolint:recvcheck // Scanner needs a pointer receiver; Valuer needs a value receiver.
type packageInstallerEnvironment []PackageInstallerEnvironmentVariable

func (v *packageInstallItems) Scan(src any) error          { return scanJSONSlice(src, v) }
func (v packageInstallItems) Value() (driver.Value, error) { return jsonSliceValue(v) }

func (v *packageReceipts) Scan(src any) error          { return scanJSONSlice(src, v) }
func (v packageReceipts) Value() (driver.Value, error) { return jsonSliceValue(v) }

func (v *packageItemsToCopy) Scan(src any) error          { return scanJSONSlice(src, v) }
func (v packageItemsToCopy) Value() (driver.Value, error) { return jsonSliceValue(v) }

func (v *packageInstallerChoices) Scan(src any) error          { return scanJSONSlice(src, v) }
func (v packageInstallerChoices) Value() (driver.Value, error) { return jsonSliceValue(v) }

func (v *packageInstallerEnvironment) Scan(src any) error          { return scanJSONSlice(src, v) }
func (v packageInstallerEnvironment) Value() (driver.Value, error) { return jsonSliceValue(v) }

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
		return fmt.Errorf("packages: cannot scan %T into %T", src, dst)
	}
}

func jsonSliceValue[T any](values []T) (driver.Value, error) {
	if values == nil {
		values = []T{}
	}
	return json.Marshal(values)
}
