package checks

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/scope"
)

// Check is a query-backed pass/fail policy.
type Check struct {
	ID              int64            `json:"id"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Query           string           `json:"query"`
	Platform        *string          `json:"platform,omitempty"`
	LabelScope      scope.LabelScope `json:"label_scope,omitzero"`
	CreatedByUserID *int64           `json:"created_by_user_id,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// CheckCreate contains editable check fields.
type CheckCreate struct {
	Name            string
	Description     string
	Query           string
	Platform        *string
	LabelScope      scope.LabelScope
	CreatedByUserID *int64
}

// CheckUpdate replaces editable check fields.
type CheckUpdate struct {
	Name        string
	Description string
	Query       string
	Platform    *string
	LabelScope  scope.LabelScope
}

// CheckListParams filters checks.
type CheckListParams struct {
	dbutil.ListParams

	Platform string
}

// CheckStatus is a check's current response value for a host.
type CheckStatus string

const (
	CheckStatusPass CheckStatus = "pass"
	CheckStatusFail CheckStatus = "fail"
)

// CheckHostStatus is a check's current state for one host.
type CheckHostStatus struct {
	CheckID   int64        `json:"check_id"`
	CheckName string       `json:"check_name"`
	HostID    int64        `json:"host_id"`
	HostName  string       `json:"host_name"`
	Response  *CheckStatus `json:"response"             enum:"pass,fail"`
	UpdatedAt *time.Time   `json:"updated_at,omitempty"`
}
