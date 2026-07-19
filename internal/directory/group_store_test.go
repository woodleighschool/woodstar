package directory

import (
	"context"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
)

func TestListAndGetGroups(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)
	seedGroups(t, ctx, store)

	groups, count, err := store.ListGroups(ctx, GroupListParams{Values: []string{"staff"}})
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if count != 1 || len(groups) != 1 {
		t.Fatalf("groups = %+v count=%d, want one group", groups, count)
	}
	if groups[0].Source != "entra" || groups[0].DisplayName != "Staff" || groups[0].MemberCount != 1 {
		t.Fatalf("group = %+v, want Entra Staff with one member", groups[0])
	}

	group, err := store.GetGroupByID(ctx, groups[0].ID)
	if err != nil {
		t.Fatalf("get group: %v", err)
	}
	if group.ExternalID != "staff" || group.MemberCount != 1 {
		t.Fatalf("group detail = %+v, want staff with one member", group)
	}
}

func seedGroups(t *testing.T, ctx context.Context, store *Store) {
	t.Helper()
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Groups: []ProviderGroup{
			{ExternalID: "all-users", DisplayName: "All Users", MailNickname: "all-users"},
			{ExternalID: "staff", DisplayName: "Staff", MailNickname: "staff"},
		},
		Users: []ProviderUser{
			{
				ExternalID:        "u-alice",
				UserPrincipalName: "alice@example.edu",
				DisplayName:       "Alice Engineering",
				Department:        "Engineering",
				Enabled:           true,
				GroupExternalIDs:  []string{"all-users", "staff"},
			},
			{
				ExternalID:        "u-bob",
				UserPrincipalName: "bob@example.edu",
				DisplayName:       "Bob Operations",
				Department:        "Operations",
				Enabled:           true,
				GroupExternalIDs:  []string{"all-users"},
			},
		},
	}); err != nil {
		t.Fatalf("apply entra snapshot: %v", err)
	}
}
