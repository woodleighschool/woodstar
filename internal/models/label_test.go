package models

import (
	"strings"
	"testing"
)

func TestCleanLabelCreate(t *testing.T) {
	t.Parallel()
	query := " select 1; "
	staticQuery := "select 1;"

	tests := []struct {
		name    string
		in      LabelCreate
		want    LabelCreate
		wantErr string
	}{
		{
			name: "dynamic label with query is valid",
			in: LabelCreate{
				Name:  " Macs ",
				Query: &query,
			},
			want: LabelCreate{
				Name:           "Macs",
				Query:          new("select 1;"),
				Kind:           LabelKindRegular,
				MembershipType: LabelMembershipTypeDynamic,
			},
		},
		{
			name: "dynamic label without query is invalid",
			in: LabelCreate{
				Name:           "No query",
				MembershipType: LabelMembershipTypeDynamic,
			},
			wantErr: "query is required for dynamic labels",
		},
		{
			name: "manual label with query is invalid",
			in: LabelCreate{
				Name:           "Manual",
				Query:          &staticQuery,
				MembershipType: LabelMembershipTypeManual,
			},
			wantErr: "query is only allowed for dynamic labels",
		},
		{
			name: "host_vitals label with query is invalid",
			in: LabelCreate{
				Name:           "Department",
				Query:          &staticQuery,
				MembershipType: LabelMembershipTypeHostVitals,
			},
			wantErr: "query is only allowed for dynamic labels",
		},
		{
			name: "missing name is invalid",
			in: LabelCreate{
				Query: &query,
			},
			wantErr: "name is required",
		},
		{
			name: "unknown kind is invalid",
			in: LabelCreate{
				Name:           "Bad kind",
				Query:          &query,
				Kind:           "magic",
				MembershipType: LabelMembershipTypeDynamic,
			},
			wantErr: "unknown label kind",
		},
		{
			name: "builtin cannot be created through admin path",
			in: LabelCreate{
				Name:           "Builtin",
				Query:          &query,
				Kind:           LabelKindBuiltin,
				MembershipType: LabelMembershipTypeDynamic,
			},
			wantErr: "builtin labels cannot be created",
		},
		{
			name: "unknown membership type is invalid",
			in: LabelCreate{
				Name:           "Bad membership",
				Query:          &query,
				MembershipType: "maybe",
			},
			wantErr: "unknown label membership type",
		},
		{
			name: "manual label without query is valid",
			in: LabelCreate{
				Name:           "Pinned",
				MembershipType: LabelMembershipTypeManual,
				Platform:       new(" darwin "),
			},
			want: LabelCreate{
				Name:           "Pinned",
				Kind:           LabelKindRegular,
				MembershipType: LabelMembershipTypeManual,
				Platform:       new("darwin"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := cleanLabelCreate(tt.in)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("cleanLabelCreate error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("cleanLabelCreate returned error: %v", err)
			}
			assertLabelCreate(t, got, tt.want)
		})
	}
}

func TestCleanLabelUpdate(t *testing.T) {
	query := " select 1; "
	got, err := cleanLabelUpdate(LabelUpdate{
		Name:           " Macs ",
		Query:          &query,
		MembershipType: LabelMembershipTypeDynamic,
		Platform:       new(" darwin "),
	})
	if err != nil {
		t.Fatalf("cleanLabelUpdate returned error: %v", err)
	}
	if got.Name != "Macs" {
		t.Fatalf("Name = %q, want %q", got.Name, "Macs")
	}
	if got.MembershipType != LabelMembershipTypeDynamic {
		t.Fatalf("MembershipType = %q, want %q", got.MembershipType, LabelMembershipTypeDynamic)
	}
	assertStringPtr(t, "Query", got.Query, new("select 1;"))
	assertStringPtr(t, "Platform", got.Platform, new("darwin"))
}

func assertLabelCreate(t *testing.T, got LabelCreate, want LabelCreate) {
	t.Helper()
	if got.Name != want.Name {
		t.Fatalf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.Description != want.Description {
		t.Fatalf("Description = %q, want %q", got.Description, want.Description)
	}
	if got.Kind != want.Kind {
		t.Fatalf("Kind = %q, want %q", got.Kind, want.Kind)
	}
	if got.MembershipType != want.MembershipType {
		t.Fatalf("MembershipType = %q, want %q", got.MembershipType, want.MembershipType)
	}
	assertStringPtr(t, "Query", got.Query, want.Query)
	assertStringPtr(t, "Platform", got.Platform, want.Platform)
}

func assertStringPtr(t *testing.T, name string, got *string, want *string) {
	t.Helper()
	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("%s = %v, want %v", name, got, want)
	case *got != *want:
		t.Fatalf("%s = %q, want %q", name, *got, *want)
	}
}
