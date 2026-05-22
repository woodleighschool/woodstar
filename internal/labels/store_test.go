package labels

import (
	"context"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/platforms"
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
				Name:      " Macs ",
				Query:     &query,
				Platforms: []platforms.Platform{" darwin ", "DARWIN"},
			},
			want: LabelCreate{
				Name:                "Macs",
				Query:               new("select 1;"),
				LabelType:           LabelTypeRegular,
				LabelMembershipType: LabelMembershipTypeDynamic,
				Platforms:           []platforms.Platform{platforms.PlatformDarwin},
			},
		},
		{
			name: "dynamic label without query is invalid",
			in: LabelCreate{
				Name:                "No query",
				LabelMembershipType: LabelMembershipTypeDynamic,
				Platforms:           allPlatforms(),
			},
			wantErr: "query is required for dynamic labels",
		},
		{
			name: "manual label with query is invalid",
			in: LabelCreate{
				Name:                "Manual",
				Query:               &staticQuery,
				LabelMembershipType: LabelMembershipTypeManual,
				Platforms:           allPlatforms(),
			},
			wantErr: "query is only allowed for dynamic labels",
		},
		{
			name: "derived label with query is invalid",
			in: LabelCreate{
				Name:                "Department",
				Query:               &staticQuery,
				LabelMembershipType: LabelMembershipTypeDerived,
				Platforms:           allPlatforms(),
			},
			wantErr: "query is only allowed for dynamic labels",
		},
		{
			name: "missing name is invalid",
			in: LabelCreate{
				Query:     &query,
				Platforms: allPlatforms(),
			},
			wantErr: "name is required",
		},
		{
			name: "unknown label type is invalid",
			in: LabelCreate{
				Name:                "Bad label type",
				Query:               &query,
				LabelType:           "magic",
				LabelMembershipType: LabelMembershipTypeDynamic,
				Platforms:           allPlatforms(),
			},
			wantErr: "unknown label type",
		},
		{
			name: "builtin cannot be created through admin path",
			in: LabelCreate{
				Name:                "Builtin",
				Query:               &query,
				LabelType:           LabelTypeBuiltin,
				LabelMembershipType: LabelMembershipTypeDynamic,
				Platforms:           allPlatforms(),
			},
			wantErr: "builtin labels cannot be created",
		},
		{
			name: "unknown membership type is invalid",
			in: LabelCreate{
				Name:                "Bad membership",
				Query:               &query,
				LabelMembershipType: "maybe",
				Platforms:           allPlatforms(),
			},
			wantErr: "unknown label membership type",
		},
		{
			name: "manual label without query is valid",
			in: LabelCreate{
				Name:                "Pinned",
				LabelMembershipType: LabelMembershipTypeManual,
				Platforms:           []platforms.Platform{" darwin "},
			},
			want: LabelCreate{
				Name:                "Pinned",
				LabelType:           LabelTypeRegular,
				LabelMembershipType: LabelMembershipTypeManual,
				Platforms:           []platforms.Platform{platforms.PlatformDarwin},
			},
		},
		{
			name: "empty platforms are invalid",
			in: LabelCreate{
				Name:  "No targets",
				Query: &query,
			},
			wantErr: "platforms are required",
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
		Name:                " Macs ",
		Query:               &query,
		LabelMembershipType: LabelMembershipTypeDynamic,
		Platforms:           []platforms.Platform{" darwin "},
	})
	if err != nil {
		t.Fatalf("cleanLabelUpdate returned error: %v", err)
	}
	if got.Name != "Macs" {
		t.Fatalf("Name = %q, want %q", got.Name, "Macs")
	}
	if got.LabelMembershipType != LabelMembershipTypeDynamic {
		t.Fatalf("LabelMembershipType = %q, want %q", got.LabelMembershipType, LabelMembershipTypeDynamic)
	}
	assertStringPtr(t, "Query", got.Query, new("select 1;"))
	assertPlatforms(t, "Platforms", got.Platforms, []platforms.Platform{platforms.PlatformDarwin})
}

func TestLabelListFiltersByPlatformTargetSet(t *testing.T) {
	store, ctx := newIntegrationLabelStore(t)
	query := "select 1;"
	if _, err := store.Create(ctx, LabelCreate{
		Name:                "All targets label",
		Query:               &query,
		LabelMembershipType: LabelMembershipTypeDynamic,
		Platforms:           allPlatforms(),
	}); err != nil {
		t.Fatalf("create all-target label: %v", err)
	}
	if _, err := store.Create(ctx, LabelCreate{
		Name:                "Windows only label",
		Query:               &query,
		LabelMembershipType: LabelMembershipTypeDynamic,
		Platforms:           []platforms.Platform{platforms.PlatformWindows},
	}); err != nil {
		t.Fatalf("create windows label: %v", err)
	}

	got, count, err := store.List(ctx, LabelListParams{LabelType: LabelTypeRegular, Platform: "darwin"})
	if err != nil {
		t.Fatalf("list labels: %v", err)
	}
	if count != 1 || len(got) != 1 || got[0].Name != "All targets label" {
		t.Fatalf("List(platform=darwin) returned count=%d rows=%+v, want only all-target label", count, got)
	}
}

func assertLabelCreate(t *testing.T, got LabelCreate, want LabelCreate) {
	t.Helper()
	if got.Name != want.Name {
		t.Fatalf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.Description != want.Description {
		t.Fatalf("Description = %q, want %q", got.Description, want.Description)
	}
	if got.LabelType != want.LabelType {
		t.Fatalf("LabelType = %q, want %q", got.LabelType, want.LabelType)
	}
	if got.LabelMembershipType != want.LabelMembershipType {
		t.Fatalf("LabelMembershipType = %q, want %q", got.LabelMembershipType, want.LabelMembershipType)
	}
	assertStringPtr(t, "Query", got.Query, want.Query)
	assertPlatforms(t, "Platforms", got.Platforms, want.Platforms)
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

func assertPlatforms(t *testing.T, name string, got []platforms.Platform, want []platforms.Platform) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %#v, want %#v", name, got, want)
		}
	}
}

func allPlatforms() []platforms.Platform {
	return []platforms.Platform{platforms.PlatformDarwin, platforms.PlatformWindows, platforms.PlatformLinux}
}

func newIntegrationLabelStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	database, ctx := dbtest.Open(t)
	return NewStore(database), ctx
}
