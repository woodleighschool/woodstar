package users

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
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
