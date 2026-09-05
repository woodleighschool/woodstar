//go:build postgres

package directory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/woodleighschool/goodies/auth/authn"

	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestLoginUsesPersistedCredentials(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database, labels.NewStore(database))
	users := NewUserService(store)
	created, err := users.Create(ctx, UserCreate{
		Email: "admin@example.invalid", Name: "Persisted Admin", Password: "correct-password", Role: RoleAdmin,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	sessions, requestCtx := loadAuthnStoreTestSession(t, ctx)
	service := newAuthnStoreTestService(t, store, sessions)
	if _, err := service.Login(requestCtx, authn.LoginParams{Email: created.Email, Password: "wrong-password"}); !errors.Is(err, authn.ErrInvalidCredentials) {
		t.Fatalf("wrong password error = %v, want authn.ErrInvalidCredentials", err)
	}

	principal, err := service.Login(requestCtx, authn.LoginParams{Email: created.Email, Password: "correct-password"})
	if err != nil {
		t.Fatalf("authenticate password: %v", err)
	}
	if principal.ID != created.ID {
		t.Fatalf("principal ID = %d, want %d", principal.ID, created.ID)
	}
}

func TestRotateAPIKeyReplacesAndRevokesBearerCredential(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database, labels.NewStore(database))
	users := NewUserService(store)
	user, err := users.Create(ctx, UserCreate{
		Email: "api-user@example.invalid", Name: "API User", Password: "correct horse battery staple", Role: RoleAdmin,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	service := newAuthnStoreTestService(t, store, newAuthnStoreTestSession())

	if err := service.RotateAPIKey(ctx, user.ID); err != nil {
		t.Fatalf("rotate first API key: %v", err)
	}
	first, err := users.GetAccount(ctx, user.ID)
	if err != nil || first.APIKey == "" || first.APIKeyCreatedAt == nil {
		t.Fatalf("first account = %+v, error = %v", first, err)
	}

	if err := service.RotateAPIKey(ctx, user.ID); err != nil {
		t.Fatalf("rotate second API key: %v", err)
	}
	second, err := users.GetAccount(ctx, user.ID)
	if err != nil || second.APIKey == "" || second.APIKey == first.APIKey {
		t.Fatalf("second account = %+v, first key = %q, error = %v", second, first.APIKey, err)
	}

	principal, err := service.Authenticate(ctx, "Bearer "+second.APIKey)
	if err != nil || principal.ID != user.ID {
		t.Fatalf("Authenticate() principal = %+v, error = %v", principal, err)
	}
	if _, err := service.Authenticate(ctx, "Bearer "+first.APIKey); !errors.Is(err, authn.ErrNotAuthenticated) {
		t.Fatalf("old API key error = %v, want authn.ErrNotAuthenticated", err)
	}

	if err := service.RevokeAPIKey(ctx, user.ID); err != nil {
		t.Fatalf("revoke API key: %v", err)
	}
	if _, err := service.Authenticate(ctx, "Bearer "+second.APIKey); !errors.Is(err, authn.ErrNotAuthenticated) {
		t.Fatalf("revoked API key error = %v, want authn.ErrNotAuthenticated", err)
	}
}

func TestSSOResolvesProvisionedPrincipal(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database, labels.NewStore(database))
	if err := store.ApplyProviderSnapshot(ctx, SourceEntra, ProviderSnapshot{Users: []ProviderUser{{
		ExternalID: "provider-user", UserPrincipalName: "user@example.invalid", Mail: "user@example.invalid", DisplayName: "Provider User", Enabled: true,
	}}}); err != nil {
		t.Fatalf("apply provider snapshot: %v", err)
	}

	principal, err := NewAuthnStore(store).GetSSOPrincipalByEmail(ctx, "user@example.invalid")
	if err != nil || principal.Email != "user@example.invalid" {
		t.Fatalf("complete SSO login principal = %+v, error = %v", principal, err)
	}
}

func TestSSORejectsLocalOnlyPrincipal(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewStore(database, labels.NewStore(database))
	if _, err := NewUserService(store).Create(ctx, UserCreate{Email: "local@example.invalid", Password: "correct-password", Role: RoleAdmin}); err != nil {
		t.Fatalf("create local user: %v", err)
	}

	if _, err := NewAuthnStore(store).GetSSOPrincipalByEmail(ctx, "local@example.invalid"); !errors.Is(err, authn.ErrPrincipalNotFound) {
		t.Fatalf("local-only SSO error = %v, want authn.ErrPrincipalNotFound", err)
	}
}

func TestAuthnStoreMapsMissingIdentities(t *testing.T) {
	database, ctx := testdb.Open(t)
	store := NewAuthnStore(NewStore(database, labels.NewStore(database)))
	const missingID int64 = 9223372036854775807
	checks := map[string]func() error{
		"session":  func() error { _, err := store.GetPrincipal(ctx, missingID); return err },
		"password": func() error { _, err := store.GetPasswordIdentityByEmail(ctx, "missing@example.invalid"); return err },
		"sso":      func() error { _, err := store.GetSSOPrincipalByEmail(ctx, "missing@example.invalid"); return err },
		"api key":  func() error { _, err := store.GetPrincipalByAPIKey(ctx, "missing-key"); return err },
		"rotate":   func() error { return store.SetAPIKey(ctx, missingID, "synthetic-key") },
		"revoke":   func() error { return store.ClearAPIKey(ctx, missingID) },
	}
	for name, check := range checks {
		t.Run(name, func(t *testing.T) {
			if err := check(); !errors.Is(err, authn.ErrPrincipalNotFound) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func newAuthnStoreTestService(t *testing.T, store *Store, sessions *scs.SessionManager) *authn.Service {
	t.Helper()
	service, err := authn.New(t.Context(), NewAuthnStore(store), sessions, authn.Config{})
	if err != nil {
		t.Fatalf("create authn service: %v", err)
	}
	return service
}

func loadAuthnStoreTestSession(t *testing.T, ctx context.Context) (*scs.SessionManager, context.Context) {
	t.Helper()
	sessions := newAuthnStoreTestSession()
	loaded, err := sessions.Load(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	return sessions, loaded
}

func newAuthnStoreTestSession() *scs.SessionManager {
	return &scs.SessionManager{
		Store: memstore.NewWithCleanupInterval(0), Codec: scs.GobCodec{}, Lifetime: time.Hour,
		Cookie: scs.SessionCookie{Name: "session", Path: "/", HttpOnly: true},
	}
}
