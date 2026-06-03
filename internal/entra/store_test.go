package entra

import (
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
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
	if err := store.db.Pool().QueryRow(ctx, `SELECT count(*) FROM users WHERE entra_id IS NOT NULL`).Scan(&userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 2 {
		t.Fatalf("user count = %d, want 2", userCount)
	}

	// Second snapshot misses Bob and removes the ops group; Alice moves to ops only.
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

	var upn, name, department string
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT user_principal_name, name, COALESCE(department, '')
		FROM users
		WHERE entra_id = 'u-alice'
	`).Scan(&upn, &name, &department); err != nil {
		t.Fatalf("get user after second snapshot: %v", err)
	}
	if upn != "alice@example.com" {
		t.Fatalf("user after second snapshot = %q, want alice", upn)
	}
	if name != "Alice (updated)" || department != "Operations" {
		t.Fatalf("alice name/department = %q/%q, want updated Operations", name, department)
	}
	var bobActive bool
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT active
		FROM users
		WHERE entra_id = 'u-bob'
	`).Scan(&bobActive); err != nil {
		t.Fatalf("get bob after second snapshot: %v", err)
	}
	if bobActive {
		t.Fatal("bob active = true, want inactive after missing from snapshot")
	}

	var groupExternalID string
	if err := store.db.Pool().
		QueryRow(ctx, `SELECT external_id FROM entra_groups`).
		Scan(&groupExternalID); err != nil {
		t.Fatalf("get remaining group: %v", err)
	}
	if groupExternalID != "g-ops" {
		t.Fatalf("remaining group = %q, want g-ops", groupExternalID)
	}
}

func TestStoreApplyAttachesExistingLocalUser(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)

	var localID int64
	if err := store.db.Pool().QueryRow(ctx, `
		INSERT INTO users (email, name, role)
		VALUES ('admin@example.edu', 'Local Admin', 'admin')
		RETURNING id
	`).Scan(&localID); err != nil {
		t.Fatalf("insert local user: %v", err)
	}

	if err := store.Apply(ctx, Snapshot{
		GeneratedAt: time.Now().UTC(),
		Users: []SnapshotUser{
			{
				ExternalID:        "entra-admin",
				UserPrincipalName: "admin@example.edu",
				Mail:              "admin@example.edu",
				DisplayName:       "Synced Admin",
				Active:            true,
			},
		},
	}); err != nil {
		t.Fatalf("apply snapshot: %v", err)
	}

	var id int64
	var role string
	var entraID string
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT id, role::text, entra_id
		FROM users
		WHERE email = 'admin@example.edu'
	`).Scan(&id, &role, &entraID); err != nil {
		t.Fatalf("load attached user: %v", err)
	}
	if id != localID {
		t.Fatalf("attached id = %d, want existing local id %d", id, localID)
	}
	if role != "admin" {
		t.Fatalf("role = %q, want preserved admin", role)
	}
	if entraID != "entra-admin" {
		t.Fatalf("entra_id = %q, want entra-admin", entraID)
	}
}

func TestStoreListsEntraSelectors(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)

	if err := store.Apply(ctx, Snapshot{
		GeneratedAt: time.Now().UTC(),
		Groups: []SnapshotGroup{
			{ExternalID: "azure-all-users", DisplayName: "Azure All Users", MailNickname: "all-users"},
			{ExternalID: "staff", DisplayName: "Staff", MailNickname: "staff"},
		},
		Users: []SnapshotUser{
			{
				ExternalID:        "ahyde",
				UserPrincipalName: "ahyde@example.edu",
				Mail:              "ahyde@example.edu",
				DisplayName:       "Alex Hyde",
				Department:        "Engineering",
				Active:            true,
				GroupExternalIDs:  []string{"azure-all-users"},
			},
			{
				ExternalID:        "ops",
				UserPrincipalName: "ops@example.edu",
				DisplayName:       "Ops User",
				Department:        "Operations",
				Active:            true,
				GroupExternalIDs:  []string{"staff"},
			},
		},
	}); err != nil {
		t.Fatalf("apply snapshot: %v", err)
	}

	users, count, err := store.ListUsers(ctx, ListParams{ListParams: dbutil.ListParams{Q: "ahyde"}})
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if count != 1 || len(users) != 1 || users[0].EntraID != "ahyde" {
		t.Fatalf("users = %+v count=%d, want ahyde only", users, count)
	}

	groups, count, err := store.ListGroups(ctx, ListParams{Values: []string{"azure-all-users"}})
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if count != 1 || len(groups) != 1 || groups[0].DisplayName != "Azure All Users" {
		t.Fatalf("groups = %+v count=%d, want Azure All Users only", groups, count)
	}

	departments, count, err := store.ListDepartments(ctx, ListParams{ListParams: dbutil.ListParams{Q: "eng"}})
	if err != nil {
		t.Fatalf("list departments: %v", err)
	}
	if count != 1 || len(departments) != 1 || departments[0].Value != "Engineering" {
		t.Fatalf("departments = %+v count=%d, want Engineering only", departments, count)
	}
}
