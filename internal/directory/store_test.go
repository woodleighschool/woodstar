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

	var userCount int
	if err := store.db.Pool().QueryRow(ctx, `SELECT count(*) FROM directory_users`).Scan(&userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 2 {
		t.Fatalf("user count = %d, want 2", userCount)
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

	var upn, displayName, department string
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT user_principal_name, display_name, COALESCE(department, '')
		FROM directory_users
	`).Scan(&upn, &displayName, &department); err != nil {
		t.Fatalf("get user after second snapshot: %v", err)
	}
	if upn != "alice@example.com" {
		t.Fatalf("user after second snapshot = %q, want only alice", upn)
	}
	if displayName != "Alice (updated)" || department != "Operations" {
		t.Fatalf("alice display_name/department = %q/%q, want updated Operations", displayName, department)
	}

	var groupExternalID string
	if err := store.db.Pool().
		QueryRow(ctx, `SELECT external_id FROM directory_groups`).
		Scan(&groupExternalID); err != nil {
		t.Fatalf("get remaining group: %v", err)
	}
	if groupExternalID != "g-ops" {
		t.Fatalf("remaining group = %q, want g-ops", groupExternalID)
	}
}
