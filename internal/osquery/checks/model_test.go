package checks

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

func TestCheckMutationValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      CheckMutation
		wantErr bool
	}{
		{
			name: "valid",
			in:   CheckMutation{Name: "Gatekeeper disabled", Query: "select 1;"},
		},
		{
			name:    "missing name",
			in:      CheckMutation{Query: "select 1;"},
			wantErr: true,
		},
		{
			name:    "missing query",
			in:      CheckMutation{Name: "No query"},
			wantErr: true,
		},
		{
			name: "duplicate include label",
			in: CheckMutation{
				Name:  "Duplicate include",
				Query: "select 1;",
				Targets: CheckTargets{
					Include: []targeting.LabelRef{{LabelID: 1}, {LabelID: 1}},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate exclude label",
			in: CheckMutation{
				Name:  "Duplicate exclude",
				Query: "select 1;",
				Targets: CheckTargets{
					Exclude: []targeting.LabelRef{{LabelID: 2}, {LabelID: 2}},
				},
			},
			wantErr: true,
		},
		{
			name: "include exclude overlap",
			in: CheckMutation{
				Name:  "Overlap",
				Query: "select 1;",
				Targets: CheckTargets{
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
				if !errors.Is(err, dbutil.ErrInvalidInput) {
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
