package checks

import (
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/humaschema"
)

// Check is a query-backed pass/fail rule.
type Check struct {
	ID               int64        `json:"id"`
	Name             string       `json:"name"`
	Description      string       `json:"description"`
	Query            string       `json:"query"`
	Targets          CheckTargets `json:"targets"`
	PassingHostCount int32        `json:"passing_host_count"`
	FailingHostCount int32        `json:"failing_host_count"`
	CreatedByUserID  *int64       `json:"created_by_user_id,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

// CheckMutation is the editable check state used by create and update.
type CheckMutation struct {
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Query       string       `json:"query"`
	Targets     CheckTargets `json:"targets"`
}

// CheckCreateMutation is the create input for a check.
type CheckCreateMutation struct {
	CheckMutation

	CreatedByUserID *int64
}

func (p CheckMutation) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	if p.Query == "" {
		return fmt.Errorf("%w: query is required", dbutil.ErrInvalidInput)
	}
	if err := p.Targets.validate(); err != nil {
		return err
	}
	return nil
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

var CheckStatusValues = []CheckStatus{CheckStatusPass, CheckStatusFail}

// CheckHostStatus is one host's check state.
type CheckHostStatus struct {
	CheckID   int64        `json:"check_id"`
	CheckName string       `json:"check_name"`
	HostID    int64        `json:"host_id"`
	HostName  string       `json:"host_name"`
	Response  *CheckStatus `json:"response"`
	UpdatedAt *time.Time   `json:"updated_at,omitempty"`
}

func (CheckStatus) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(CheckStatusValues...)
}
