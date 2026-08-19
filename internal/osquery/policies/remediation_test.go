package policies

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/fault"
)

func TestValidatePolicyRemediationStatusFilters(t *testing.T) {
	t.Parallel()

	if err := validatePolicyRemediationStatusFilters(policyRemediationStatusFilterValues); err != nil {
		t.Fatalf("validatePolicyRemediationStatusFilters(%q) = %v, want nil", policyRemediationStatusFilterValues, err)
	}
	if err := validatePolicyRemediationStatusFilters([]PolicyRemediationStatusFilter{"unknown"}); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("unknown remediation status error = %v, want ErrInvalidInput", err)
	}
}
