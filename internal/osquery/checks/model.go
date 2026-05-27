package checks

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/scope"
)

// Check is a query-backed pass/fail rule.
type Check struct {
	ID              int64            `json:"id"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Query           string           `json:"query"`
	LabelScope      scope.LabelScope `json:"label_scope,omitzero"`
	CreatedByUserID *int64           `json:"created_by_user_id,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// CheckCreate is a new check.
type CheckCreate struct {
	Name            string           `json:"name"`
	Description     string           `json:"description,omitempty"`
	Query           string           `json:"query"`
	LabelScope      scope.LabelScope `json:"label_scope"`
	CreatedByUserID *int64           `json:"-"`
}

// CheckUpdate is the editable check state.
type CheckUpdate struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Query       string           `json:"query"`
	LabelScope  scope.LabelScope `json:"label_scope"`
}

// CheckListParams filters checks.
type CheckListParams struct {
	dbutil.ListParams
}

// CheckStatus is the latest check result.
type CheckStatus string

const (
	CheckStatusPass CheckStatus = "pass"
	CheckStatusFail CheckStatus = "fail"
)

// CheckHostStatus is one host's check state.
type CheckHostStatus struct {
	CheckID   int64        `json:"check_id"`
	CheckName string       `json:"check_name"`
	HostID    int64        `json:"host_id"`
	HostName  string       `json:"host_name"`
	Response  *CheckStatus `json:"response"             enum:"pass,fail"`
	UpdatedAt *time.Time   `json:"updated_at,omitempty"`
}
