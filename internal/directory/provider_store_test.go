package directory

import (
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
)

func TestApplyProviderSnapshotReconcilesUsersAndGroups(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)

	first := ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Groups: []ProviderGroup{
			{ExternalID: "g-eng", DisplayName: "Engineering"},
			{ExternalID: "g-ops", DisplayName: "Operations"},
		},
		Users: []ProviderUser{
			{
				ExternalID:        "u-alice",
				UserPrincipalName: "alice@example.com",
				DisplayName:       "Alice",
				Department:        "Engineering",
				Enabled:           true,
				GroupExternalIDs:  []string{"g-eng", "g-ops"},
			},
			{
				ExternalID:        "u-bob",
				UserPrincipalName: "bob@example.com",
				DisplayName:       "Bob",
				Department:        "Operations",
				Enabled:           true,
				GroupExternalIDs:  []string{"g-ops"},
			},
		},
	}
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, first); err != nil {
		t.Fatalf("apply first snapshot: %v", err)
	}

	var userCount int
	if err := store.db.Pool().
		QueryRow(ctx, `SELECT count(*) FROM users WHERE source = 'entra'`).
		Scan(&userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 2 {
		t.Fatalf("user count = %d, want 2", userCount)
	}

	// Second snapshot misses Bob and removes the ops group; Alice moves to ops only.
	second := ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Groups: []ProviderGroup{
			{ExternalID: "g-ops", DisplayName: "Operations"},
		},
		Users: []ProviderUser{
			{
				ExternalID:        "u-alice",
				UserPrincipalName: "alice@example.com",
				DisplayName:       "Alice (updated)",
				Department:        "Operations",
				Enabled:           true,
				GroupExternalIDs:  []string{"g-ops"},
			},
		},
	}
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, second); err != nil {
		t.Fatalf("apply second snapshot: %v", err)
	}

	var upn, name, department string
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT user_principal_name, name, COALESCE(department, '')
		FROM users
		WHERE source = 'entra' AND external_id = 'u-alice'
	`).Scan(&upn, &name, &department); err != nil {
		t.Fatalf("get user after second snapshot: %v", err)
	}
	if upn != "alice@example.com" {
		t.Fatalf("user after second snapshot = %q, want alice", upn)
	}
	if name != "Alice (updated)" || department != "Operations" {
		t.Fatalf("alice name/department = %q/%q, want updated Operations", name, department)
	}
	var bobDeletedAt *time.Time
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT deleted_at
		FROM users
		WHERE source = 'entra' AND external_id = 'u-bob'
	`).Scan(&bobDeletedAt); err != nil {
		t.Fatalf("get bob after second snapshot: %v", err)
	}
	if bobDeletedAt == nil {
		t.Fatal("bob deleted_at is nil, want soft-deleted after missing from snapshot")
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

	third := ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Groups: []ProviderGroup{
			{ExternalID: "g-ops", DisplayName: "Operations"},
		},
		Users: []ProviderUser{
			{
				ExternalID:        "u-bob",
				UserPrincipalName: "bob@example.edu",
				DisplayName:       "Bob Returned",
				Department:        "Operations",
				Enabled:           true,
				GroupExternalIDs:  []string{"g-ops"},
			},
		},
	}
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, third); err != nil {
		t.Fatalf("apply third snapshot: %v", err)
	}
	bobDeletedAt = &time.Time{}
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT deleted_at
		FROM users
		WHERE source = 'entra' AND external_id = 'u-bob'
	`).Scan(&bobDeletedAt); err != nil {
		t.Fatalf("get bob after third snapshot: %v", err)
	}
	if bobDeletedAt != nil {
		t.Fatalf("bob deleted_at = %v, want nil after returning to Entra", bobDeletedAt)
	}
}

func TestApplyProviderSnapshotReusesDeletedEntraUserWhenRecreatedWithNewExternalID(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)

	var userID int64
	if err := store.db.Pool().QueryRow(ctx, `
		INSERT INTO users (
			email,
			name,
			role,
			source,
			external_id,
			user_principal_name,
			deleted_at
		)
		VALUES (
			'recreated@example.edu',
			'Recreated User',
			'viewer',
			'entra',
			'old-object-id',
			'recreated@example.edu',
			now()
		)
		RETURNING id
	`).Scan(&userID); err != nil {
		t.Fatalf("insert deleted entra user: %v", err)
	}

	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Users: []ProviderUser{
			{
				ExternalID:        "new-object-id",
				UserPrincipalName: "recreated@example.edu",
				Mail:              "recreated@example.edu",
				DisplayName:       "Recreated User",
				Enabled:           true,
			},
		},
	}); err != nil {
		t.Fatalf("apply recreated user snapshot: %v", err)
	}

	var gotID int64
	var externalID string
	var deletedAt *time.Time
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT id, external_id, deleted_at
		FROM users
		WHERE email = 'recreated@example.edu'
	`).Scan(&gotID, &externalID, &deletedAt); err != nil {
		t.Fatalf("load recreated user: %v", err)
	}
	if gotID != userID {
		t.Fatalf("user id = %d, want reused id %d", gotID, userID)
	}
	if externalID != "new-object-id" {
		t.Fatalf("external_id = %q, want new-object-id", externalID)
	}
	if deletedAt != nil {
		t.Fatalf("deleted_at = %v, want nil", deletedAt)
	}
}

func TestApplyProviderSnapshotAttachesExistingLocalUser(t *testing.T) {
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

	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Users: []ProviderUser{
			{
				ExternalID:        "entra-admin",
				UserPrincipalName: "admin@example.edu",
				Mail:              "admin@example.edu",
				DisplayName:       "Directory Admin",
				Enabled:           true,
			},
		},
	}); err != nil {
		t.Fatalf("apply snapshot: %v", err)
	}

	var id int64
	var role string
	var source, externalID string
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT id, role::text, source::text, external_id
		FROM users
		WHERE email = 'admin@example.edu'
	`).Scan(&id, &role, &source, &externalID); err != nil {
		t.Fatalf("load attached user: %v", err)
	}
	if id != localID {
		t.Fatalf("attached id = %d, want existing local id %d", id, localID)
	}
	if role != "admin" {
		t.Fatalf("role = %q, want preserved admin", role)
	}
	if source != "entra" {
		t.Fatalf("source = %q, want entra", source)
	}
	if externalID != "entra-admin" {
		t.Fatalf("external_id = %q, want entra-admin", externalID)
	}
}
