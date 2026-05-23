package scope

import (
	"database/sql/driver"
	"fmt"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// LabelScopeMode is the Postgres enum that selects how LabelIDs are interpreted.
type LabelScopeMode string

const (
	ScopeNone       LabelScopeMode = "none"
	ScopeIncludeAny LabelScopeMode = "include_any"
	ScopeIncludeAll LabelScopeMode = "include_all"
	ScopeExcludeAny LabelScopeMode = "exclude_any"
)

// LabelScope is the shared label targeting shape for reports and checks.
type LabelScope struct {
	Mode     LabelScopeMode `json:"mode,omitempty"      enum:"include_any,include_all,exclude_any"`
	LabelIDs []int64        `json:"label_ids,omitempty"`
}

// IsZero reports whether s is the "no scope" value so json:omitzero can drop
// it from wire output. ScopeNone is the canonical empty mode.
func (s LabelScope) IsZero() bool {
	return (s.Mode == "" || s.Mode == ScopeNone) && len(s.LabelIDs) == 0
}

// NormalizeLabelScope sorts label IDs, removes invalid duplicates, and collapses empty scopes.
func NormalizeLabelScope(s LabelScope) LabelScope {
	s.LabelIDs = dbutil.CleanPositiveIDs(s.LabelIDs)
	switch s.Mode {
	case ScopeNone, ScopeIncludeAny, ScopeIncludeAll, ScopeExcludeAny:
	default:
		s.Mode = ScopeNone
	}
	if len(s.LabelIDs) == 0 {
		s.Mode = ScopeNone
	}
	return s
}

func (m *LabelScopeMode) Scan(src any) error {
	switch value := src.(type) {
	case string:
		*m = LabelScopeMode(value)
	case []byte:
		*m = LabelScopeMode(value)
	default:
		return fmt.Errorf("scope: unsupported label scope mode scan type %T", src)
	}
	return nil
}

func (m LabelScopeMode) Value() (driver.Value, error) {
	return string(m), nil
}
