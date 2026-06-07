package directory

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestCreateHashesPassword(t *testing.T) {
	database, ctx := dbtest.Open(t)
	service := NewUserService(NewStore(database))

	user, err := service.Create(ctx, UserCreate{
		Email:    "local@example.test",
		Name:     "Local User",
		Role:     RoleAdmin,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if user.PasswordHash == "" {
		t.Fatal("password hash is empty")
	}
	if user.PasswordHash == "correct-password" {
		t.Fatal("password hash stored raw password")
	}
	valid, err := VerifyPassword("correct-password", user.PasswordHash)
	if err != nil {
		t.Fatalf("verify password hash: %v", err)
	}
	if !valid {
		t.Fatal("password hash does not verify original password")
	}
}

func TestUpdateHashesPassword(t *testing.T) {
	database, ctx := dbtest.Open(t)
	service := NewUserService(NewStore(database))
	role := RoleAdmin
	user, err := service.Create(ctx, UserCreate{
		Email:    "local@example.test",
		Name:     "Local User",
		Role:     role,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	newPassword := "better-password"
	updated, err := service.Update(ctx, user.ID, UserMutation{
		Name:     "Local User",
		Role:     &role,
		Password: &newPassword,
	})
	if err != nil {
		t.Fatalf("update user: %v", err)
	}

	if updated.PasswordHash == user.PasswordHash {
		t.Fatal("password hash did not change")
	}
	valid, err := VerifyPassword(newPassword, updated.PasswordHash)
	if err != nil {
		t.Fatalf("verify updated password hash: %v", err)
	}
	if !valid {
		t.Fatal("updated password hash does not verify new password")
	}
}

func TestUpdateAccountHashesPassword(t *testing.T) {
	database, ctx := dbtest.Open(t)
	service := NewUserService(NewStore(database))
	user, err := service.Create(ctx, UserCreate{
		Email:    "local@example.test",
		Name:     "Local User",
		Role:     RoleAdmin,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	newPassword := "account-password"
	account, err := service.UpdateAccount(ctx, user.ID, AccountMutation{
		Name:     "Local User",
		Password: &newPassword,
	})
	if err != nil {
		t.Fatalf("update account: %v", err)
	}

	if account.User.PasswordHash == user.PasswordHash {
		t.Fatal("password hash did not change")
	}
	valid, err := VerifyPassword(newPassword, account.User.PasswordHash)
	if err != nil {
		t.Fatalf("verify account password hash: %v", err)
	}
	if !valid {
		t.Fatal("account password hash does not verify new password")
	}
}

func TestSetAndClearAccountAPIKey(t *testing.T) {
	database, ctx := dbtest.Open(t)
	service := NewUserService(NewStore(database))
	user, err := service.Create(ctx, UserCreate{
		Email:    "local@example.test",
		Name:     "Local User",
		Role:     RoleAdmin,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	const apiKey = "test-account-api-key"
	account, err := service.SetAccountAPIKey(ctx, user.ID, apiKey)
	if err != nil {
		t.Fatalf("set account api key: %v", err)
	}
	if account.APIKey != apiKey {
		t.Fatalf("api key = %q, want %q", account.APIKey, apiKey)
	}
	if account.APIKeyCreatedAt == nil {
		t.Fatal("api key created at is nil")
	}
	got, err := service.GetByAPIKey(ctx, apiKey)
	if err != nil {
		t.Fatalf("get by api key: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("api key user id = %d, want %d", got.ID, user.ID)
	}

	cleared, err := service.ClearAccountAPIKey(ctx, user.ID)
	if err != nil {
		t.Fatalf("clear account api key: %v", err)
	}
	if cleared.APIKey != "" {
		t.Fatalf("cleared api key = %q, want empty", cleared.APIKey)
	}
	if cleared.APIKeyCreatedAt != nil {
		t.Fatalf("cleared api key created at = %v, want nil", cleared.APIKeyCreatedAt)
	}
	if _, err := service.GetByAPIKey(ctx, apiKey); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get cleared api key err = %v, want %v", err, dbutil.ErrNotFound)
	}
}
