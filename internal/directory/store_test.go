package directory

import (
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
)

func TestStoreApplyReconcilesUsersAndGroups(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)

	first := Snapshot{
		GeneratedAt: time.Now().UTC(),
		Groups: []SnapshotGroup{
			{ExternalID: "g-eng", DisplayName: "Engineering"},
			{ExternalID: "g-ops", DisplayName: "Operations"},
		},
		Users: []SnapshotUser{
			{
				ExternalID:        "u-alice",
				UserPrincipalName: "alice@example.com",
				DisplayName:       "Alice",
				Department:        "Engineering",
				Active:            true,
				GroupExternalIDs:  []string{"g-eng", "g-ops"},
			},
			{
				ExternalID:        "u-bob",
				UserPrincipalName: "bob@example.com",
				DisplayName:       "Bob",
				Department:        "Operations",
				Active:            true,
				GroupExternalIDs:  []string{"g-ops"},
			},
		},
	}
	if err := store.Apply(ctx, first); err != nil {
		t.Fatalf("apply first snapshot: %v", err)
	}

	users, err := store.ListUsers(ctx)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("user count = %d, want 2", len(users))
	}

	// Second snapshot drops Bob and removes the ops group; Alice moves to ops only.
	second := Snapshot{
		GeneratedAt: time.Now().UTC(),
		Groups: []SnapshotGroup{
			{ExternalID: "g-ops", DisplayName: "Operations"},
		},
		Users: []SnapshotUser{
			{
				ExternalID:        "u-alice",
				UserPrincipalName: "alice@example.com",
				DisplayName:       "Alice (updated)",
				Department:        "Operations",
				Active:            true,
				GroupExternalIDs:  []string{"g-ops"},
			},
		},
	}
	if err := store.Apply(ctx, second); err != nil {
		t.Fatalf("apply second snapshot: %v", err)
	}

	users, err = store.ListUsers(ctx)
	if err != nil {
		t.Fatalf("list users after second: %v", err)
	}
	if len(users) != 1 || users[0].UserPrincipalName != "alice@example.com" {
		t.Fatalf("users after second snapshot = %+v, want only alice", users)
	}
	if users[0].DisplayName != "Alice (updated)" || users[0].Department != "Operations" {
		t.Fatalf("alice not updated: %+v", users[0])
	}

	groups, err := store.ListGroups(ctx)
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(groups) != 1 || groups[0].ExternalID != "g-ops" {
		t.Fatalf("groups = %+v, want only g-ops", groups)
	}
}
