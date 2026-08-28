package policies

import (
	"fmt"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/validation"
)

// Policy is a query-backed pass/fail rule.
type Policy struct {
	ID               int64                    `json:"id"`
	Name             string                   `json:"name"`
	Description      string                   `json:"description"`
	Resolution       string                   `json:"resolution"`
	Query            string                   `json:"query"`
	Targets          PolicyTargets            `json:"targets"`
	Remediation      PolicyRemediationSummary `json:"remediation"`
	PassingHostCount int32                    `json:"passing_host_count"`
	FailingHostCount int32                    `json:"failing_host_count"`
	ErrorHostCount   int32                    `json:"error_host_count"`
	PendingHostCount int32                    `json:"pending_host_count"`
	CreatedBy        *directory.UserSummary   `json:"created_by,omitempty"`
	CreatedAt        time.Time                `json:"created_at"`
	UpdatedAt        time.Time                `json:"updated_at"`
}

// PolicyMutation is the editable policy state used by create and update.
type PolicyMutation struct {
	Name        string                     `json:"name"                  validate:"required,notblank" minLength:"1"`
	Description string                     `json:"description,omitempty"`
	Resolution  string                     `json:"resolution,omitempty"`
	Query       string                     `json:"query"                 validate:"required,notblank" minLength:"1"`
	Targets     PolicyTargets              `json:"targets"`
	Remediation *PolicyRemediationMutation `json:"remediation,omitempty"`
}

// PolicyRemediationMutation configures an optional policy-owned script.
type PolicyRemediationMutation struct {
	Script    string `json:"script"    validate:"required,notblank" minLength:"1"`
	Automatic bool   `json:"automatic"`
}

// PolicyRemediationSummary is safe to show without exposing script contents.
type PolicyRemediationSummary struct {
	Configured bool `json:"configured"`
	Automatic  bool `json:"automatic"`
}

// PolicyRemediationSource is the administrator-only script source.
type PolicyRemediationSource struct {
	Script string `json:"script"`
}

// PolicyCreateMutation is the create input for a policy.
type PolicyCreateMutation struct {
	PolicyMutation

	CreatedByUserID *int64
}

func (p *PolicyMutation) Validate() error {
	if err := validation.Struct(p); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	if p.Remediation != nil {
		if err := validation.Struct(p.Remediation); err != nil {
			return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
		}
	}
	if err := p.Targets.validate(); err != nil {
		return err
	}
	return nil
}

func (p *PolicyMutation) normalize() {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.Resolution = strings.TrimSpace(p.Resolution)
	p.Query = strings.TrimSpace(p.Query)
	p.Targets = normalizePolicyTargets(p.Targets)
}

// PolicyListParams filters policies.
type PolicyListParams struct {
	ListParams listing.Params
}

// PolicyResultListParams filters and paginates per-host policy state.
type PolicyResultListParams struct {
	ListParams listing.Params

	Statuses            []PolicyStatus
	RemediationStatuses []PolicyRemediationStatusFilter
}

// Evaluation is one issued policy query for a host.
type Evaluation struct {
	PolicyID int64
	Query    string
	Revision int64
	Sequence int64
}

// EvaluationResult is one conclusive or errored osquery response.
type EvaluationResult struct {
	PolicyID  int64
	QueryHash string
	Revision  int64
	Sequence  int64
	Status    PolicyStatus
	Error     string
}

// PolicyStatus is the latest policy result.
type PolicyStatus string

const (
	PolicyStatusPass    PolicyStatus = "pass"
	PolicyStatusFail    PolicyStatus = "fail"
	PolicyStatusPending PolicyStatus = "pending"
	PolicyStatusError   PolicyStatus = "error"
)

var PolicyStatusValues = []PolicyStatus{
	PolicyStatusPending,
	PolicyStatusPass,
	PolicyStatusFail,
	PolicyStatusError,
}

// PolicyHostStatus is one host's policy state.
type PolicyHostStatus struct {
	PolicyID    int64                        `json:"policy_id"`
	PolicyName  string                       `json:"policy_name"`
	HostID      int64                        `json:"host_id"`
	HostName    string                       `json:"host_name"`
	Status      PolicyStatus                 `json:"status"`
	Error       string                       `json:"error,omitempty"`
	UpdatedAt   *time.Time                   `json:"updated_at,omitempty"`
	Remediation *PolicyRemediationRunSummary `json:"remediation,omitempty"`
}

func (PolicyStatus) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(PolicyStatusValues...)
}
