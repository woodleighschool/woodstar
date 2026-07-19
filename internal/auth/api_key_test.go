package auth

import (
	"errors"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/directory"
)

func TestRotateAPIKeyReplacesPreviousCredential(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := directory.NewUserService(directory.NewStore(database))
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
	service := testAuthService(t, userService, sessions, InitialAdminConfig{
		Email:    user.Email,
		Password: "configured-password",
	})

	first, err := service.RotateAPIKey(ctx, user.ID)
	if err != nil {
		t.Fatalf("rotate first api key: %v", err)
	}
	if first.APIKey == "" {
		t.Fatal("first rotated api key is empty")
	}

	second, err := service.RotateAPIKey(ctx, user.ID)
	if err != nil {
		t.Fatalf("rotate second api key: %v", err)
	}
	if second.APIKey == "" {
		t.Fatal("second rotated api key is empty")
	}
	if second.APIKey == first.APIKey {
		t.Fatal("second rotated api key did not replace the first")
	}

	got, err := service.principalByAPIKey(ctx, second.APIKey)
	if err != nil {
		t.Fatalf("authenticate with second api key: %v", err)
	}
	if got.UserID == nil || *got.UserID != user.ID {
		t.Fatalf("api key principal = %+v, want user %d", got, user.ID)
	}
	if _, err := service.principalByAPIKey(ctx, first.APIKey); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("authenticate with first api key error = %v, want %v", err, ErrNotAuthenticated)
	}
}
