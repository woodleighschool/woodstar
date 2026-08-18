package policies

import (
	"errors"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/fault"
)

func TestRemediationRunSummaryUsesConfiguredExecutionTimeout(t *testing.T) {
	t.Parallel()

	now := time.Now()
	claimedAt := now.Add(-7 * time.Minute)
	run := remediationRunRow{
		ExecutionID: "execution-id",
		QueuedAt:    claimedAt.Add(-time.Minute),
		ClaimedAt:   &claimedAt,
	}

	if got := remediationRunSummary(run, now, 30*time.Minute).Status; got != PolicyRemediationRunStatusInProgress {
		t.Fatalf("30 minute timeout status = %q, want in_progress", got)
	}
	if got := remediationRunSummary(run, now, 5*time.Minute).Status; got != PolicyRemediationRunStatusNoResponse {
		t.Fatalf("5 minute timeout status = %q, want no_response", got)
	}
}

func TestValidatePolicyRemediationStatusFilters(t *testing.T) {
	t.Parallel()

	if err := validatePolicyRemediationStatusFilters(policyRemediationStatusFilterValues); err != nil {
		t.Fatalf("validatePolicyRemediationStatusFilters(%q) = %v, want nil", policyRemediationStatusFilterValues, err)
	}
	if err := validatePolicyRemediationStatusFilters([]PolicyRemediationStatusFilter{"unknown"}); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("unknown remediation status error = %v, want ErrInvalidInput", err)
	}
}
