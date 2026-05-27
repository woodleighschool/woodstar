package labels

import (
	"strings"
	"testing"
)

func TestLabelCreateValidate(t *testing.T) {
	t.Parallel()
	query := "select 1;"
	tests := []struct {
		name    string
		in      LabelCreate
		wantErr string
	}{
		{
			name: "dynamic label with query is valid",
			in: LabelCreate{
				Name:                "Macs",
				Query:               &query,
				LabelMembershipType: LabelMembershipTypeDynamic,
			},
		},
		{
			name: "dynamic label without query is invalid",
			in: LabelCreate{
				Name:                "No query",
				LabelMembershipType: LabelMembershipTypeDynamic,
			},
			wantErr: "query is required for dynamic labels",
		},
		{
			name: "manual label with query is invalid",
			in: LabelCreate{
				Name:                "Manual",
				Query:               &query,
				LabelMembershipType: LabelMembershipTypeManual,
			},
			wantErr: "query is only allowed for dynamic labels",
		},
		{
			name: "builtin cannot be created through admin path",
			in: LabelCreate{
				Name:                "Builtin",
				Query:               &query,
				LabelType:           LabelTypeBuiltin,
				LabelMembershipType: LabelMembershipTypeDynamic,
			},
			wantErr: "builtin labels cannot be created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.in.Validate()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Validate error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate returned error: %v", err)
			}
		})
	}
}
