package scope

import (
	"database/sql/driver"
	"fmt"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/humaschema"
)

// LabelScopeMode says how to read LabelIDs.
type LabelScopeMode string

const (
	ScopeNone       LabelScopeMode = "none"
	ScopeIncludeAny LabelScopeMode = "include_any"
	ScopeIncludeAll LabelScopeMode = "include_all"
	ScopeExcludeAny LabelScopeMode = "exclude_any"
)

var LabelScopeModeValues = []LabelScopeMode{ScopeIncludeAny, ScopeIncludeAll, ScopeExcludeAny}

// LabelScope is shared label targeting.
type LabelScope struct {
	Mode     LabelScopeMode `json:"mode,omitempty"`
	LabelIDs []int64        `json:"label_ids,omitempty"`
}

func (LabelScopeMode) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(LabelScopeModeValues...)
}

// IsZero lets json:omitzero drop the no-scope value.
func (s LabelScope) IsZero() bool {
	return (s.Mode == "" || s.Mode == ScopeNone) && len(s.LabelIDs) == 0
}

// NormalizeLabelScope validates the mode and collapses empty scopes.
func NormalizeLabelScope(s LabelScope) LabelScope {
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
