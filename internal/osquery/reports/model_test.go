package reports

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/targeting"
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
			in:      ReportMutation{Name: "No query"},
			wantErr: true,
		},
		{
			name:    "negative schedule interval",
			in:      ReportMutation{Name: "Bad schedule", Query: "select 1;", ScheduleInterval: -1},
			wantErr: true,
		},
		{
			name: "duplicate include label",
			in: ReportMutation{
				Name:  "Duplicate include",
				Query: "select 1;",
				Targets: ReportTargets{
					Include: []targeting.LabelRef{{LabelID: 1}, {LabelID: 1}},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate exclude label",
			in: ReportMutation{
				Name:  "Duplicate exclude",
				Query: "select 1;",
				Targets: ReportTargets{
					Exclude: []targeting.LabelRef{{LabelID: 2}, {LabelID: 2}},
				},
			},
			wantErr: true,
		},
		{
			name: "include exclude overlap",
			in: ReportMutation{
				Name:  "Overlap",
				Query: "select 1;",
				Targets: ReportTargets{
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
