//go:build postgres

package auth

import (
	"errors"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestRotateAPIKeyReplacesAndRevokesBearerCredential(t *testing.T) {
	database, ctx := testdb.Open(t)
	userService := directory.NewUserService(directory.NewStore(database, labels.NewStore(database)))
	user, err := userService.Create(ctx, directory.UserCreate{
		Email:    "api-user@example.invalid",
		Name:     "API User",
		Password: "correct horse battery staple",
		Role:     directory.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	sessions := scs.New()
	sessions.Store = memstore.New()
	service := testAuthService(t, userService, sessions)

	first, err := service.RotateAPIKey(ctx, user.ID)
	if err != nil {
		t.Fatalf("rotate first API key: %v", err)
	}
	if first.APIKey == "" || first.APIKeyCreatedAt == nil {
		t.Fatalf("first API key = %q created at = %v", first.APIKey, first.APIKeyCreatedAt)
	}

	second, err := service.RotateAPIKey(ctx, user.ID)
	if err != nil {
		t.Fatalf("rotate second API key: %v", err)
	}
	if second.APIKey == "" || second.APIKey == first.APIKey {
		t.Fatalf("second API key = %q, first = %q", second.APIKey, first.APIKey)
	}

	got, err := service.Authenticate(ctx, "Bearer "+second.APIKey)
	if err != nil {
		t.Fatalf("authenticate with second API key: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("authenticated user = %+v", got)
	}
	if _, err := service.Authenticate(ctx, "Bearer "+first.APIKey); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("authenticate with first API key error = %v, want %v", err, ErrNotAuthenticated)
	}

	revoked, err := service.RevokeAPIKey(ctx, user.ID)
	if err != nil {
		t.Fatalf("revoke API key: %v", err)
	}
	if revoked.APIKey != "" || revoked.APIKeyCreatedAt != nil {
		t.Fatalf("revoked API key = %q created at = %v", revoked.APIKey, revoked.APIKeyCreatedAt)
	}
	if _, err := service.Authenticate(ctx, "Bearer "+second.APIKey); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("authenticate with revoked API key error = %v, want %v", err, ErrNotAuthenticated)
	}
}
