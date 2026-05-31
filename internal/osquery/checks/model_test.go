package checks

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
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
			in:      CheckMutation{Name: "No query", Query: " "},
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
