package users

import (
	"errors"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/entra"
)

func TestDeleteRemovesLocalUsers(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)
	service := NewService(store)

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
	if _, err := store.GetByID(ctx, user.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get deleted user err = %v, want %v", err, dbutil.ErrNotFound)
	}
}

func TestDeleteDeactivatesSyncedUsers(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)
	service := NewService(store)

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
    active,
    api_key,
    api_key_created_at,
    entra_id,
    user_principal_name,
    last_synced_at
)
VALUES (
    'synced@example.test',
    'Synced User',
    $1,
    'admin',
    true,
    $2,
    now(),
    'entra-user-1',
    'synced@example.test',
    now()
)
RETURNING id`, hash, apiKey).Scan(&userID); err != nil {
		t.Fatalf("insert synced user: %v", err)
	}

	if err := service.Delete(ctx, userID); err != nil {
		t.Fatalf("delete synced user: %v", err)
	}

	user, err := store.GetByID(ctx, userID)
	if err != nil {
		t.Fatalf("get synced user: %v", err)
	}
	if !user.Synced {
		t.Fatal("synced user lost Entra ownership")
	}
	if user.Active {
		t.Fatal("synced user stayed active")
	}
	if user.Role != nil {
		t.Fatalf("role = %v, want nil", *user.Role)
	}
	if user.CanLogin {
		t.Fatal("synced user can still log in")
	}
	if _, err := store.GetByAPIKey(ctx, apiKey); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get deactivated user by api key err = %v, want %v", err, dbutil.ErrNotFound)
	}
}

func TestListFiltersUsers(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)
	entraStore := entra.NewStore(database)

	if err := entraStore.Apply(ctx, entra.Snapshot{
		GeneratedAt: time.Now().UTC(),
		Users: []entra.SnapshotUser{
			{
				ExternalID:        "u-alice",
				UserPrincipalName: "alice@example.edu",
				DisplayName:       "Alice Engineering",
				Department:        "Engineering",
				Active:            true,
			},
			{
				ExternalID:        "u-bob",
				UserPrincipalName: "bob@example.edu",
				DisplayName:       "Bob Operations",
				Department:        "Operations",
				Active:            false,
			},
		},
	}); err != nil {
		t.Fatalf("apply entra snapshot: %v", err)
	}
	local, err := store.Create(ctx, UserCreate{
		Email:    "local@example.edu",
		Name:     "Local Admin",
		Role:     RoleAdmin,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create local user: %v", err)
	}

	users, count, err := store.List(ctx, ListParams{
		ListParams: dbutil.ListParams{Q: "engineering"},
		Source:     "synced",
		Status:     "active",
	})
	if err != nil {
		t.Fatalf("list synced engineering users: %v", err)
	}
	if count != 1 || len(users) != 1 || users[0].Email != "alice@example.edu" {
		t.Fatalf("users = %+v count=%d, want Alice only", users, count)
	}

	users, count, err = store.List(ctx, ListParams{Role: "admin", Source: "local"})
	if err != nil {
		t.Fatalf("list local admins: %v", err)
	}
	if count != 1 || len(users) != 1 || users[0].ID != local.ID {
		t.Fatalf("local admins = %+v count=%d, want local admin", users, count)
	}
}

func TestListDepartmentsReturnsSyncedDepartments(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)
	entraStore := entra.NewStore(database)

	if err := entraStore.Apply(ctx, entra.Snapshot{
		GeneratedAt: time.Now().UTC(),
		Users: []entra.SnapshotUser{
			{
				ExternalID:        "u-alice",
				UserPrincipalName: "alice@example.edu",
				DisplayName:       "Alice",
				Department:        "Engineering",
				Active:            true,
			},
			{
				ExternalID:        "u-bob",
				UserPrincipalName: "bob@example.edu",
				DisplayName:       "Bob",
				Department:        "Operations",
				Active:            true,
			},
		},
	}); err != nil {
		t.Fatalf("apply entra snapshot: %v", err)
	}

	departments, count, err := store.ListDepartments(ctx, ListParams{ListParams: dbutil.ListParams{Q: "eng"}})
	if err != nil {
		t.Fatalf("list departments: %v", err)
	}
	if count != 1 || len(departments) != 1 || departments[0].Value != "Engineering" {
		t.Fatalf("departments = %+v count=%d, want Engineering only", departments, count)
	}
}
