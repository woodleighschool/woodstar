//go:build postgres

package directory

import (
	"errors"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestProviderIdentityLinksByCanonicalEmailNotUPN(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	service := newTestUserService(store)

	upnOwner, err := service.Create(ctx, UserCreate{
		Email:    "upn-owner@example.test",
		Name:     "UPN Owner",
		Role:     RoleViewer,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create UPN owner: %v", err)
	}
	mailOwner, err := service.Create(ctx, UserCreate{
		Email:    "mail-owner@example.test",
		Name:     "Mail Owner",
		Role:     RoleViewer,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create mail owner: %v", err)
	}

	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		Users: []ProviderUser{{
			ExternalID:        "provider-identity",
			Mail:              mailOwner.Email,
			UserPrincipalName: upnOwner.Email,
			DisplayName:       "Provider Identity",
			Enabled:           true,
		}},
	}); err != nil {
		t.Fatalf("apply provider snapshot: %v", err)
	}

	var linkedUserID int64
	if err := database.Pool().QueryRow(ctx, `
SELECT user_id
FROM directory_user_links
WHERE source = 'entra' AND external_id = 'provider-identity'`).Scan(&linkedUserID); err != nil {
		t.Fatalf("load provider link: %v", err)
	}
	if linkedUserID != mailOwner.ID {
		t.Fatalf("linked user ID = %d, want canonical email owner %d", linkedUserID, mailOwner.ID)
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

	if _, err := service.GetSSOByEmail(ctx, "alternate@example.test"); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("SSO lookup by UPN error = %v, want %v", err, dbutil.ErrNotFound)
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
	if err := database.Pool().QueryRow(ctx, `
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
	if err := database.Pool().QueryRow(
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
	if _, err := database.Pool().Exec(ctx, `
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
	if err := database.Pool().QueryRow(
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
	if err := provider.ApplyProviderSnapshot(ctx, SourceEntra, second); err != nil {
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
	if err := provider.ApplyProviderSnapshot(ctx, SourceEntra, third); err != nil {
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
	database, ctx := testdb.Open(t)
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

func TestApplyProviderSnapshotPreservesExistingLocalUser(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)

	var localID int64
	if err := store.db.Pool().QueryRow(ctx, `
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
	if err := store.db.Pool().QueryRow(ctx, `
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
	if err := store.db.Pool().QueryRow(ctx, `
		SELECT user_id FROM directory_user_links
		WHERE source = 'entra' AND external_id = 'entra-admin'
	`).Scan(&providerID); err != nil {
		t.Fatalf("load provider identity link: %v", err)
	}
	if providerID != localID {
		t.Fatalf("provider identity user = %d, want local user %d", providerID, localID)
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
	var linkCount int
	if err := store.db.Pool().QueryRow(
		ctx,
		`SELECT count(*) FROM directory_user_links WHERE user_id = $1`,
		localID,
	).Scan(&linkCount); err != nil {
		t.Fatalf("count removed provider identity links: %v", err)
	}
	if linkCount != 0 {
		t.Fatalf("provider identity links = %d, want 0 after removal", linkCount)
	}
}
