package auth

import (
	"errors"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/randtoken"
)

func TestGenerateAPIKey(t *testing.T) {
	t.Parallel()
	a, err := randtoken.Generate(apiKeyByteLen)
	if err != nil {
		t.Fatalf("randtoken.Generate returned error: %v", err)
	}
	b, err := randtoken.Generate(apiKeyByteLen)
	if err != nil {
		t.Fatalf("randtoken.Generate returned error: %v", err)
	}
	if a == b {
		t.Fatalf("two consecutive keys collided: %q", a)
	}
	if len(a) < 32 {
		t.Fatalf("key length = %d, want >= 32 (24 random bytes base64-encoded)", len(a))
	}
}

func TestRotateAndRevokeAPIKey(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := directory.NewUserService(directory.NewStore(database), labels.NewStore(database))
	user, err := userService.Create(ctx, directory.UserCreate{
		Email:    "api@example.test",
		Name:     "API User",
		Password: "correct-password",
		Role:     directory.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	sessions := scs.New()
	sessions.Store = memstore.New()
	service := NewService(userService, sessions)

	account, err := service.RotateAPIKey(ctx, user.ID)
	if err != nil {
		t.Fatalf("rotate api key: %v", err)
	}
	if account.APIKey == "" {
		t.Fatal("rotated api key is empty")
	}
	got, err := userService.GetByAPIKey(ctx, account.APIKey)
	if err != nil {
		t.Fatalf("get user by rotated api key: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("api key user id = %d, want %d", got.ID, user.ID)
	}

	revoked, err := service.RevokeAPIKey(ctx, user.ID)
	if err != nil {
		t.Fatalf("revoke api key: %v", err)
	}
	if revoked.APIKey != "" {
		t.Fatalf("revoked api key = %q, want empty", revoked.APIKey)
	}
	if _, err := userService.GetByAPIKey(ctx, account.APIKey); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get revoked api key err = %v, want %v", err, dbutil.ErrNotFound)
	}
}
