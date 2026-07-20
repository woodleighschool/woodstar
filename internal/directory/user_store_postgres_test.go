//go:build postgres

package directory

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestUsersRejectUppercaseIdentityFields(t *testing.T) {
	database, ctx := testdb.Open(t)

	tests := []struct {
		name string
		stmt string
	}{
		{
			name: "email",
			stmt: `INSERT INTO users (email, name) VALUES ('UPPER@example.test', 'Upper Email')`,
		},
		{
			name: "user principal name",
			stmt: `
INSERT INTO users (email, name, source, external_id, user_principal_name)
VALUES ('lower@example.test', 'Upper UPN', 'entra', 'upper-upn', 'UPPER@example.test')`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := database.Pool().Exec(ctx, test.stmt)
			var postgresError *pgconn.PgError
			if !errors.As(err, &postgresError) || postgresError.Code != "23514" {
				t.Fatalf("insert error = %v, want check violation", err)
			}
		})
	}
}

func TestDeleteRemovesLocalUsers(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	service := newTestUserService(store)
	if _, err := service.Create(ctx, UserCreate{
		Email:    "other-admin@example.test",
		Name:     "Other Admin",
		Role:     RoleAdmin,
		Password: "correct-password",
	}); err != nil {
		t.Fatalf("create other admin: %v", err)
	}

	user, err := service.Create(ctx, UserCreate{
		Email:    "local@example.test",
		Name:     "Local User",
		Role:     RoleAdmin,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := service.Delete(ctx, user.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	if _, err := store.GetUserByID(ctx, user.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get deleted user err = %v, want %v", err, dbutil.ErrNotFound)
	}
}

func TestDeleteSoftDeletesDirectoryUsers(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	service := newTestUserService(store)
	if _, err := service.Create(ctx, UserCreate{
		Email:    "local-admin@example.test",
		Name:     "Local Admin",
		Role:     RoleAdmin,
		Password: "correct-password",
	}); err != nil {
		t.Fatalf("create local admin: %v", err)
	}

	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	const apiKey = "test-api-key"
	var userID int64
	if err := database.Pool().QueryRow(ctx, `
INSERT INTO users (
    email,
    name,
    password_hash,
    role,
    source,
    external_id,
    api_key,
    api_key_created_at,
    user_principal_name
)
VALUES (
    'source@example.test',
    'Sourced User',
    $1,
    'admin',
    'entra',
    'entra-user-1',
    $2,
    now(),
    'source@example.test'
)
RETURNING id`, hash, apiKey).Scan(&userID); err != nil {
		t.Fatalf("insert sourced user: %v", err)
	}

	if err := service.Delete(ctx, userID); err != nil {
		t.Fatalf("soft delete directory user: %v", err)
	}

	if _, err := store.GetUserByID(ctx, userID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get deleted directory user err = %v, want %v", err, dbutil.ErrNotFound)
	}

	var source Source
	var externalID string
	var role string
	var deletedAt *time.Time
	if err := database.Pool().QueryRow(ctx, `
SELECT source::text, external_id, role::text, deleted_at
FROM users
WHERE id = $1`, userID).Scan(&source, &externalID, &role, &deletedAt); err != nil {
		t.Fatalf("load soft-deleted directory user: %v", err)
	}
	if source != SourceEntra || externalID != "entra-user-1" {
		t.Fatalf("source identity = %s/%q, want entra/entra-user-1", source, externalID)
	}
	if role != "admin" {
		t.Fatalf("role = %q, want preserved admin", role)
	}
	if deletedAt == nil {
		t.Fatal("deleted_at is nil, want soft-deleted")
	}
	if _, err := store.GetUserByAPIKey(ctx, apiKey); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get soft-deleted user by api key err = %v, want %v", err, dbutil.ErrNotFound)
	}
}

func TestListFiltersUsers(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)
	service := newTestUserService(store)

	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Groups: []ProviderGroup{
			{ExternalID: "all-users", DisplayName: "All Users", MailNickname: "all-users"},
			{ExternalID: "engineering", DisplayName: "Engineering", MailNickname: "engineering"},
		},
		Users: []ProviderUser{
			{
				ExternalID:        "u-alice",
				UserPrincipalName: "alice@example.edu",
				DisplayName:       "Alice Engineering",
				Department:        "Engineering",
				Enabled:           true,
				GroupExternalIDs:  []string{"all-users", "engineering"},
			},
			{
				ExternalID:        "u-bob",
				UserPrincipalName: "bob@example.edu",
				DisplayName:       "Bob Operations",
				Department:        "Operations",
				Enabled:           false,
				GroupExternalIDs:  []string{"all-users"},
			},
		},
	}); err != nil {
		t.Fatalf("apply entra snapshot: %v", err)
	}
	var engineeringGroupID int64
	if err := database.Pool().QueryRow(ctx, `
SELECT id
FROM directory_groups
WHERE external_id = 'engineering'`).Scan(&engineeringGroupID); err != nil {
		t.Fatalf("get engineering group id: %v", err)
	}
	local, err := service.Create(ctx, UserCreate{
		Email:    "local@example.edu",
		Name:     "Local Admin",
		Role:     RoleAdmin,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create local user: %v", err)
	}

	users, count, err := store.ListUsers(ctx, UserListParams{
		ListParams: dbutil.ListParams{Q: "engineering"},
		Source:     "entra",
	})
	if err != nil {
		t.Fatalf("list Entra engineering users: %v", err)
	}
	if count != 1 || len(users) != 1 || users[0].Email != "alice@example.edu" {
		t.Fatalf("users = %+v count=%d, want Alice only", users, count)
	}

	users, count, err = store.ListUsers(ctx, UserListParams{Source: "entra"})
	if err != nil {
		t.Fatalf("list current Entra users: %v", err)
	}
	if count != 1 || len(users) != 1 || users[0].Email != "alice@example.edu" {
		t.Fatalf("current Entra users = %+v count=%d, want Alice only", users, count)
	}

	users, count, err = store.ListUsers(ctx, UserListParams{GroupID: engineeringGroupID})
	if err != nil {
		t.Fatalf("list engineering group users: %v", err)
	}
	if count != 1 || len(users) != 1 || users[0].Email != "alice@example.edu" {
		t.Fatalf("engineering group users = %+v count=%d, want Alice only", users, count)
	}

	users, count, err = store.ListUsers(ctx, UserListParams{Role: "admin", Source: "local"})
	if err != nil {
		t.Fatalf("list local admins: %v", err)
	}
	if count != 1 || len(users) != 1 || users[0].ID != local.ID {
		t.Fatalf("local admins = %+v count=%d, want local admin", users, count)
	}
}

func TestListDepartmentsReturnsDirectoryDepartments(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database)

	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{
		GeneratedAt: time.Now().UTC(),
		Users: []ProviderUser{
			{
				ExternalID:        "u-alice",
				UserPrincipalName: "alice@example.edu",
				DisplayName:       "Alice",
				Department:        "Engineering",
				Enabled:           true,
			},
			{
				ExternalID:        "u-bob",
				UserPrincipalName: "bob@example.edu",
				DisplayName:       "Bob",
				Department:        "Operations",
				Enabled:           true,
			},
		},
	}); err != nil {
		t.Fatalf("apply entra snapshot: %v", err)
	}

	departments, count, err := store.ListDepartments(ctx, UserListParams{ListParams: dbutil.ListParams{Q: "eng"}})
	if err != nil {
		t.Fatalf("list departments: %v", err)
	}
	if count != 1 || len(departments) != 1 || departments[0].Value != "Engineering" {
		t.Fatalf("departments = %+v count=%d, want Engineering only", departments, count)
	}
}
