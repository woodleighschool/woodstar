package policies

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

func TestPolicyMutationValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      PolicyMutation
		wantErr bool
	}{
		{
			name: "valid",
			in:   PolicyMutation{Name: "Gatekeeper disabled", Query: "select 1;"},
		},
		{
			name:    "missing name",
			in:      PolicyMutation{Query: "select 1;"},
			wantErr: true,
		},
		{
			name:    "missing query",
			in:      PolicyMutation{Name: "No query"},
			wantErr: true,
		},
		{
			name: "duplicate include label",
			in: PolicyMutation{
				Name:  "Duplicate include",
				Query: "select 1;",
				Targets: PolicyTargets{
					Include: []targeting.LabelRef{{LabelID: 1}, {LabelID: 1}},
				},
			},
			wantErr: true,
		},
		{
			name: "non-positive include label",
			in: PolicyMutation{
				Name:  "Zero include",
				Query: "select 1;",
				Targets: PolicyTargets{
					Include: []targeting.LabelRef{{LabelID: 0}},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate exclude label",
			in: PolicyMutation{
				Name:  "Duplicate exclude",
				Query: "select 1;",
				Targets: PolicyTargets{
					Exclude: []targeting.LabelRef{{LabelID: 2}, {LabelID: 2}},
				},
			},
			wantErr: true,
		},
		{
			name: "include exclude overlap",
			in: PolicyMutation{
				Name:  "Overlap",
				Query: "select 1;",
				Targets: PolicyTargets{
					Include: []targeting.LabelRef{{LabelID: 3}},
					Exclude: []targeting.LabelRef{{LabelID: 3}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if tt.wantErr {
				if !errors.Is(err, fault.ErrInvalidInput) {
					t.Fatalf("Validate error = %v, want ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate error = %v", err)
			}
		})
	}
}

func TestValidatePolicyStatusFilters(t *testing.T) {
	t.Parallel()

	if err := validatePolicyStatusFilters(PolicyStatusValues); err != nil {
		t.Fatalf("validatePolicyStatusFilters(%q) = %v, want nil", PolicyStatusValues, err)
	}
	if err := validatePolicyStatusFilters([]PolicyStatus{"unknown"}); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("unknown status error = %v, want ErrInvalidInput", err)
	}
}
