package groups

import (
	"context"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/entra"
)

func TestListAndGetGroups(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)
	seedGroups(t, database, ctx)

	groups, count, err := store.List(ctx, ListParams{Values: []string{"staff"}})
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if count != 1 || len(groups) != 1 {
		t.Fatalf("groups = %+v count=%d, want one group", groups, count)
	}
	if groups[0].DisplayName != "Staff" || groups[0].MemberCount != 1 {
		t.Fatalf("group = %+v, want Staff with one member", groups[0])
	}

	group, err := store.GetByID(ctx, groups[0].ID)
	if err != nil {
		t.Fatalf("get group: %v", err)
	}
	if group.ExternalID != "staff" || group.MemberCount != 1 {
		t.Fatalf("group detail = %+v, want staff with one member", group)
	}
}

func seedGroups(t *testing.T, database *database.DB, ctx context.Context) {
	t.Helper()
	entraStore := entra.NewStore(database)
	if err := entraStore.Apply(ctx, entra.Snapshot{
		GeneratedAt: time.Now().UTC(),
		Groups: []entra.SnapshotGroup{
			{ExternalID: "all-users", DisplayName: "All Users", MailNickname: "all-users"},
			{ExternalID: "staff", DisplayName: "Staff", MailNickname: "staff"},
		},
		Users: []entra.SnapshotUser{
			{
				ExternalID:        "u-alice",
				UserPrincipalName: "alice@example.edu",
				DisplayName:       "Alice Engineering",
				Department:        "Engineering",
				Active:            true,
				GroupExternalIDs:  []string{"all-users", "staff"},
			},
			{
				ExternalID:        "u-bob",
				UserPrincipalName: "bob@example.edu",
				DisplayName:       "Bob Operations",
				Department:        "Operations",
				Active:            true,
				GroupExternalIDs:  []string{"all-users"},
			},
		},
	}); err != nil {
		t.Fatalf("apply entra snapshot: %v", err)
	}
}
