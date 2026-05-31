package reports

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestReportMutationValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      ReportMutation
		wantErr bool
	}{
		{
			name: "valid",
			in:   ReportMutation{Name: "OS version", Query: "select * from os_version;"},
		},
		{
			name:    "missing name",
			in:      ReportMutation{Query: "select 1;"},
			wantErr: true,
		},
		{
			name:    "missing query",
			in:      ReportMutation{Name: "No query", Query: "\n\t"},
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
