package labels

import (
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestLabelMutationValidate(t *testing.T) {
	t.Parallel()
	query := "select 1;"
	unknownTableQuery := "select version from osquery_info;"
	tests := []struct {
		name    string
		in      LabelMutation
		wantErr string
	}{
		{
			name: "dynamic label with query is valid",
			in: LabelMutation{
				Name:                "Macs",
				Query:               &query,
				LabelMembershipType: LabelMembershipTypeDynamic,
			},
		},
		{
			name: "dynamic label with unknown osquery table is valid",
			in: LabelMutation{
				Name:                "Osquery info",
				Query:               &unknownTableQuery,
				LabelMembershipType: LabelMembershipTypeDynamic,
			},
		},
		{
			name: "dynamic label without query is invalid",
			in: LabelMutation{
				Name:                "No query",
				LabelMembershipType: LabelMembershipTypeDynamic,
			},
			wantErr: "query is required for dynamic labels",
		},
		{
			name: "dynamic label without name is invalid",
			in: LabelMutation{
				Query:               &query,
				LabelMembershipType: LabelMembershipTypeDynamic,
			},
			wantErr: "Name is required",
		},
		{
			name: "manual label with query is invalid",
			in: LabelMutation{
				Name:                "Manual",
				Query:               &query,
				LabelMembershipType: LabelMembershipTypeManual,
			},
			wantErr: "query is only allowed for dynamic labels",
		},
		{
			name: "derived label with criteria is valid",
			in: LabelMutation{
				Name: "Department",
				Criteria: &Criteria{
					Attribute: DerivedAttributeUserDepartment,
					Values:    []string{"Engineering"},
				},
				LabelMembershipType: LabelMembershipTypeDerived,
			},
		},
		{
			name: "derived label without criteria is invalid",
			in: LabelMutation{
				Name:                "Department",
				LabelMembershipType: LabelMembershipTypeDerived,
			},
			wantErr: "criteria is required for derived labels",
		},
		{
			name: "dynamic label with hosts is invalid",
			in: LabelMutation{
				Name:                "Dynamic hosts",
				Query:               &query,
				HostIDs:             []int64{1},
				LabelMembershipType: LabelMembershipTypeDynamic,
			},
			wantErr: "hosts are only allowed for manual labels",
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

func TestDynamicMembershipValuesRejectsDuplicateLabels(t *testing.T) {
	t.Parallel()

	_, _, err := dynamicMembershipValues([]DynamicMembership{
		{LabelID: 1, Matched: true},
		{LabelID: 1, Matched: false},
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("dynamicMembershipValues error = %v, want ErrInvalidInput", err)
	}
}
