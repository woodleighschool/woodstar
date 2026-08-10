//go:build postgres

package directory

import (
	"errors"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestProviderSnapshotKeepsLocalAndEntraUsersSeparate(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	service := newTestUserService(store)

	local, err := service.Create(ctx, UserCreate{
		Email:    "shared@example.test",
		Name:     "Local User",
		Role:     RoleViewer,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create local user: %v", err)
	}

	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		Users: []ProviderUser{{
			ExternalID:        "provider-identity",
			Mail:              local.Email,
			UserPrincipalName: "provider-upn@example.test",
			DisplayName:       "Provider Identity",
			Enabled:           true,
		}},
	}); err != nil {
		t.Fatalf("apply provider snapshot: %v", err)
	}

	var providerID int64
	var providerRole *string
	if err := database.QueryRow(ctx, `
SELECT id, role::text
FROM users
WHERE source = 'entra' AND external_id = 'provider-identity'`).Scan(
		&providerID,
		&providerRole,
	); err != nil {
		t.Fatalf("load provider user: %v", err)
	}
	if providerID == local.ID {
		t.Fatalf("provider user reused local user id %d", local.ID)
	}
	if providerRole != nil {
		t.Fatalf("provider role = %q, want nil", *providerRole)
	}
	if _, err := service.GetSSOByEmail(ctx, local.Email); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("SSO lookup before provider grant error = %v, want %v", err, fault.ErrNotFound)
	}

	granted, err := service.SetRoleByEmail(ctx, local.Email, RoleViewer)
	if err != nil {
		t.Fatalf("grant provider role by email: %v", err)
	}
	if granted.ID != providerID {
		t.Fatalf("granted user id = %d, want provider user %d", granted.ID, providerID)
	}
	if sso, err := service.GetSSOByEmail(ctx, local.Email); err != nil || sso.ID != providerID {
		t.Fatalf("SSO user after provider grant = %+v, %v", sso, err)
	}
	if login, err := service.GetLoginByEmail(ctx, local.Email); err != nil || login.ID != local.ID {
		t.Fatalf("password login user = %+v, %v", login, err)
	}
}

func TestSSOLookupDoesNotUseUPNAsAlternateAccountIdentifier(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	service := newTestUserService(store)

	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		Users: []ProviderUser{{
			ExternalID:        "provider-user",
			Mail:              "canonical@example.test",
			UserPrincipalName: "alternate@example.test",
			DisplayName:       "Provider User",
			Enabled:           true,
		}},
	}); err != nil {
		t.Fatalf("apply provider snapshot: %v", err)
	}
	if _, err := service.SetRoleByEmail(ctx, "canonical@example.test", RoleViewer); err != nil {
		t.Fatalf("grant app role: %v", err)
	}

	if _, err := service.GetSSOByEmail(ctx, "alternate@example.test"); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("SSO lookup by UPN error = %v, want %v", err, fault.ErrNotFound)
	}
	user, err := service.GetSSOByEmail(ctx, "canonical@example.test")
	if err != nil {
		t.Fatalf("SSO lookup by canonical email: %v", err)
	}
	if user.Email != "canonical@example.test" {
		t.Fatalf("SSO user email = %q, want canonical email", user.Email)
	}
}

func TestApplyProviderSnapshotRevokesLastProviderAdministrator(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	service := newTestUserService(store)

	provider := store
	if err := provider.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		Users: []ProviderUser{{
			ExternalID:        "admin-object-id",
			UserPrincipalName: "admin@example.test",
			DisplayName:       "Directory Admin",
			Enabled:           true,
		}},
	}); err != nil {
		t.Fatalf("seed provider user: %v", err)
	}
	var adminID int64
	if err := database.QueryRow(ctx, `
UPDATE users
SET role = 'admin'
WHERE external_id = 'admin-object-id'
RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("grant administrator role: %v", err)
	}

	if err := provider.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{}); err != nil {
		t.Fatalf("remove last provider administrator: %v", err)
	}
	if _, err := service.Get(ctx, adminID); err == nil {
		t.Fatal("revoked provider administrator remains active")
	}
	var deletedAt *time.Time
	if err := database.QueryRow(
		ctx,
		`SELECT deleted_at FROM users WHERE id = $1`,
		adminID,
	).Scan(&deletedAt); err != nil {
		t.Fatalf("load revoked provider administrator: %v", err)
	}
	if deletedAt == nil {
		t.Fatal("revoked provider administrator deleted_at is nil")
	}
}

func TestApplyProviderSnapshotRollsBackWhenDerivedLabelsCannotRefresh(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	if _, err := database.Exec(ctx, `
INSERT INTO labels (name, criteria, label_type, label_membership_type)
VALUES ('Invalid derived label', '{"attribute":"invalid","values":["value"]}', 'regular', 'derived')`); err != nil {
		t.Fatalf("insert invalid derived label: %v", err)
	}
	provider := store

	err := provider.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		Users: []ProviderUser{{
			ExternalID:        "rollback-user",
			UserPrincipalName: "rollback@example.test",
			DisplayName:       "Rollback User",
			Enabled:           true,
		}},
	})
	if err == nil {
		t.Fatal("provider snapshot succeeded despite derived label refresh failure")
	}

	var count int
	if err := database.QueryRow(
		ctx,
		`SELECT count(*) FROM users WHERE external_id = 'rollback-user'`,
	).Scan(&count); err != nil {
		t.Fatalf("count rolled-back users: %v", err)
	}
	if count != 0 {
		t.Fatalf("persisted users = %d, want 0", count)
	}
}

func TestApplyProviderSnapshotReconcilesUsersAndGroups(t *testing.T) {
	database, ctx := testdb.Open(t)
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
	provider := store
	if err := provider.ApplyProviderSnapshot(ctx, SourceEntra, first); err != nil {
		t.Fatalf("apply first snapshot: %v", err)
	}

	var userCount int
	if err := store.pool.
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
	if err := provider.ApplyProviderSnapshot(ctx, SourceEntra, second); err != nil {
		t.Fatalf("apply second snapshot: %v", err)
	}

	var upn, name, department string
	if err := store.pool.QueryRow(ctx, `
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
	if err := store.pool.QueryRow(ctx, `
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
	if err := store.pool.
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
	if err := provider.ApplyProviderSnapshot(ctx, SourceEntra, third); err != nil {
		t.Fatalf("apply third snapshot: %v", err)
	}
	bobDeletedAt = &time.Time{}
	if err := store.pool.QueryRow(ctx, `
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

func TestApplyProviderSnapshotKeepsReplacedEntraObjectsDistinct(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)

	user := ProviderUser{
		ExternalID:        "old-object-id",
		UserPrincipalName: "recreated@example.edu",
		Mail:              "recreated@example.edu",
		DisplayName:       "Recreated User",
		Enabled:           true,
	}
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Users:       []ProviderUser{user},
	}); err != nil {
		t.Fatalf("apply original user snapshot: %v", err)
	}

	var oldUserID int64
	if err := store.pool.QueryRow(ctx, `
UPDATE users
SET role = 'viewer'
WHERE source = 'entra' AND external_id = 'old-object-id'
RETURNING id`).Scan(&oldUserID); err != nil {
		t.Fatalf("grant original user role: %v", err)
	}

	user.ExternalID = "new-object-id"
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Users:       []ProviderUser{user},
	}); err != nil {
		t.Fatalf("apply replacement user snapshot: %v", err)
	}

	var oldDeletedAt *time.Time
	var oldRole string
	if err := store.pool.QueryRow(ctx, `
		SELECT role::text, deleted_at
		FROM users WHERE id = $1
	`, oldUserID).Scan(&oldRole, &oldDeletedAt); err != nil {
		t.Fatalf("load original user: %v", err)
	}
	if oldRole != "viewer" {
		t.Fatalf("original user role = %q, want viewer", oldRole)
	}
	if oldDeletedAt == nil {
		t.Fatal("original user remains active after its object left the snapshot")
	}

	var newUserID int64
	var newRole *string
	var newDeletedAt *time.Time
	if err := store.pool.QueryRow(ctx, `
		SELECT id, role::text, deleted_at
		FROM users
		WHERE source = 'entra' AND external_id = 'new-object-id'
	`).Scan(&newUserID, &newRole, &newDeletedAt); err != nil {
		t.Fatalf("load replacement user: %v", err)
	}
	if newUserID == oldUserID {
		t.Fatalf("replacement user reused original user id %d", oldUserID)
	}
	if newRole != nil {
		t.Fatalf("replacement user role = %q, want nil", *newRole)
	}
	if newDeletedAt != nil {
		t.Fatalf("replacement user deleted_at = %v, want nil", newDeletedAt)
	}
}

func TestApplyProviderSnapshotReconcilesEmailSwapByExternalID(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	service := newTestUserService(store)

	users := []ProviderUser{
		{
			ExternalID:        "object-a",
			UserPrincipalName: "a@example.edu",
			Mail:              "a@example.edu",
			DisplayName:       "Object A",
			Enabled:           true,
		},
		{
			ExternalID:        "object-b",
			UserPrincipalName: "b@example.edu",
			Mail:              "b@example.edu",
			DisplayName:       "Object B",
			Enabled:           true,
		},
	}
	for _, user := range users {
		if _, err := service.Create(ctx, UserCreate{
			Email:    user.Mail,
			Name:     "Local " + user.DisplayName,
			Role:     RoleViewer,
			Password: "correct-password",
		}); err != nil {
			t.Fatalf("create local user for %s: %v", user.ExternalID, err)
		}
	}
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Users:       users,
	}); err != nil {
		t.Fatalf("apply original snapshot: %v", err)
	}

	users[0].Mail, users[1].Mail = users[1].Mail, users[0].Mail
	users[0].UserPrincipalName, users[1].UserPrincipalName =
		users[1].UserPrincipalName, users[0].UserPrincipalName
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Users:       users,
	}); err != nil {
		t.Fatalf("apply swapped snapshot: %v", err)
	}

	for _, user := range users {
		var email, upn string
		var deletedAt *time.Time
		if err := store.pool.QueryRow(ctx, `
			SELECT email, user_principal_name, deleted_at
			FROM users
			WHERE source = 'entra' AND external_id = $1
		`, user.ExternalID).Scan(&email, &upn, &deletedAt); err != nil {
			t.Fatalf("load %s: %v", user.ExternalID, err)
		}
		if email != user.Mail {
			t.Fatalf("%s email = %q, want %q", user.ExternalID, email, user.Mail)
		}
		if upn != user.UserPrincipalName {
			t.Fatalf("%s UPN = %q, want %q", user.ExternalID, upn, user.UserPrincipalName)
		}
		if deletedAt != nil {
			t.Fatalf("%s deleted_at = %v, want nil", user.ExternalID, deletedAt)
		}
	}
}

func TestApplyProviderSnapshotPreservesExistingLocalUser(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)

	var localID int64
	if err := store.pool.QueryRow(ctx, `
		INSERT INTO users (email, name, password_hash, role)
		VALUES ('admin@example.edu', 'Local Admin', 'password-hash', 'admin')
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

	var role, source string
	var externalID *string
	if err := store.pool.QueryRow(ctx, `
		SELECT role::text, source::text, external_id
		FROM users WHERE id = $1
	`, localID).Scan(&role, &source, &externalID); err != nil {
		t.Fatalf("load local user: %v", err)
	}
	if role != "admin" {
		t.Fatalf("role = %q, want preserved admin", role)
	}
	if source != "local" {
		t.Fatalf("local source = %q, want local", source)
	}
	if externalID != nil {
		t.Fatalf("local external_id = %q, want nil", *externalID)
	}
	login, err := store.GetLoginUserByEmail(ctx, "admin@example.edu")
	if err != nil {
		t.Fatalf("load local password login: %v", err)
	}
	if login.ID != localID {
		t.Fatalf("password login user = %d, want local user %d", login.ID, localID)
	}

	var providerID int64
	var providerRole *string
	if err := store.pool.QueryRow(ctx, `
		SELECT id, role::text FROM users
		WHERE source = 'entra' AND external_id = 'entra-admin'
	`).Scan(&providerID, &providerRole); err != nil {
		t.Fatalf("load provider user: %v", err)
	}
	if providerID == localID {
		t.Fatalf("provider user reused local user id %d", localID)
	}
	if providerRole != nil {
		t.Fatalf("provider role = %q, want nil", *providerRole)
	}

	if err := store.ApplyProviderSnapshot(
		ctx,
		SourceEntra,
		ProviderSnapshot{},
	); err != nil {
		t.Fatalf("remove linked provider identity: %v", err)
	}
	login, err = store.GetLoginUserByEmail(ctx, "admin@example.edu")
	if err != nil || login.ID != localID {
		t.Fatalf("password login after provider removal = %+v, %v", login, err)
	}
	var providerDeletedAt *time.Time
	if err := store.pool.QueryRow(ctx, `
		SELECT deleted_at FROM users WHERE id = $1
	`, providerID).Scan(&providerDeletedAt); err != nil {
		t.Fatalf("load removed provider user: %v", err)
	}
	if providerDeletedAt == nil {
		t.Fatal("provider user remains active after provider removal")
	}
}
