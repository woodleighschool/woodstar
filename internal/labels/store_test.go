package labels

import (
	"context"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/scope"
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
				Platforms:           []scope.Platform{scope.PlatformDarwin},
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
				Query:               &query,
				LabelMembershipType: LabelMembershipTypeManual,
				Platforms:           allPlatforms(),
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
				Platforms:           allPlatforms(),
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

func TestLabelListFiltersByPlatformTargetSet(t *testing.T) {
	store, ctx := newIntegrationLabelStore(t)
	query := "select 1;"
	if _, err := store.Create(ctx, LabelCreate{
		Name:                "All targets label",
		LabelType:           LabelTypeRegular,
		Query:               &query,
		LabelMembershipType: LabelMembershipTypeDynamic,
		Platforms:           allPlatforms(),
	}); err != nil {
		t.Fatalf("create all-target label: %v", err)
	}
	if _, err := store.Create(ctx, LabelCreate{
		Name:                "Windows only label",
		LabelType:           LabelTypeRegular,
		Query:               &query,
		LabelMembershipType: LabelMembershipTypeDynamic,
		Platforms:           []scope.Platform{scope.PlatformWindows},
	}); err != nil {
		t.Fatalf("create windows label: %v", err)
	}

	got, count, err := store.List(ctx, ListParams{LabelType: LabelTypeRegular, Platform: "darwin"})
	if err != nil {
		t.Fatalf("list labels: %v", err)
	}
	if count != 1 || len(got) != 1 || got[0].Name != "All targets label" {
		t.Fatalf("List(platform=darwin) returned count=%d rows=%+v, want only all-target label", count, got)
	}
}

func allPlatforms() []scope.Platform {
	return []scope.Platform{scope.PlatformDarwin, scope.PlatformWindows, scope.PlatformLinux}
}

func newIntegrationLabelStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	database, ctx := dbtest.Open(t)
	return NewStore(database), ctx
}
