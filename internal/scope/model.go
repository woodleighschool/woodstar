package scope

import "github.com/woodleighschool/woodstar/internal/database/sqlc"

// LabelScopeMode is the Postgres enum that selects how LabelIDs are interpreted.
type LabelScopeMode = sqlc.LabelScopeMode

const (
	ScopeNone       = sqlc.LabelScopeModeNone
	ScopeIncludeAny = sqlc.LabelScopeModeIncludeAny
	ScopeIncludeAll = sqlc.LabelScopeModeIncludeAll
	ScopeExcludeAny = sqlc.LabelScopeModeExcludeAny
)

// LabelScope is the shared label targeting shape for queries, checks, and campaigns.
type LabelScope struct {
	Mode     LabelScopeMode `json:"mode,omitempty"      enum:"include_any,include_all,exclude_any"`
	LabelIDs []int64        `json:"label_ids,omitempty"`
}

// IsZero reports whether s is the "no scope" value so json:omitzero can drop
// it from wire output. ScopeNone is the canonical empty mode.
func (s LabelScope) IsZero() bool {
	return (s.Mode == "" || s.Mode == ScopeNone) && len(s.LabelIDs) == 0
}
