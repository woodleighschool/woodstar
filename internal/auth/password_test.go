package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/directory"
)

func TestInitialAdminLoginNeedsNoDirectoryUser(t *testing.T) {
	sessions := testSessionManager()
	service := testAuthService(t, nil, sessions, InitialAdminConfig{
		Email:    "admin@example.test",
		Password: "configured-password",
	})
	ctx := loadTestSession(t, sessions, context.Background())

	principal, err := service.Login(ctx, LoginParams{
		Email:    " Admin@Example.Test ",
		Password: "configured-password",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if principal.UserID != nil || principal.Email != "admin@example.test" ||
		principal.Role != directory.RoleAdmin {
		t.Fatalf("principal = %+v, want non-persisted administrator", principal)
	}

	restored, err := service.CurrentPrincipal(ctx)
	if err != nil {
		t.Fatalf("restore initial administrator: %v", err)
	}
	if restored.UserID != nil || restored.Email != principal.Email {
		t.Fatalf("restored principal = %+v, want %+v", restored, principal)
	}
}

func TestInitialAdminUsesLocalPasswordPolicy(t *testing.T) {
	_, err := NewService(nil, testSessionManager(), InitialAdminConfig{
		Email:    "admin@example.test",
		Password: "too-short",
	})
	if !errors.Is(err, directory.ErrWeakPassword) {
		t.Fatalf("NewService error = %v, want %v", err, directory.ErrWeakPassword)
	}
}

func TestInitialAdminShadowsSameEmailDirectoryLogin(t *testing.T) {
	database, ctx := dbtest.Open(t)
	users := directory.NewUserService(directory.NewStore(database))
	persisted, err := users.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Persisted Admin",
		Role:     directory.RoleAdmin,
		Password: "persisted-password",
	})
	if err != nil {
		t.Fatalf("create persisted administrator: %v", err)
	}

	sessions := testSessionManager()
	service := testAuthService(t, users, sessions, InitialAdminConfig{
		Email:    "admin@example.test",
		Password: "configured-password",
	})
	requestCtx := loadTestSession(t, sessions, ctx)

	if _, err := service.Login(requestCtx, LoginParams{
		Email:    "ADMIN@example.test",
		Password: "persisted-password",
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("persisted password error = %v, want %v", err, ErrInvalidCredentials)
	}
	principal, err := service.Login(requestCtx, LoginParams{
		Email:    "ADMIN@example.test",
		Password: "configured-password",
	})
	if err != nil {
		t.Fatalf("configured administrator login: %v", err)
	}
	if principal.UserID != nil {
		t.Fatalf("configured principal user ID = %v, want nil", principal.UserID)
	}

	got, err := users.Get(ctx, persisted.ID)
	if err != nil {
		t.Fatalf("get persisted administrator: %v", err)
	}
	if got.Name != persisted.Name || got.PasswordHash != persisted.PasswordHash {
		t.Fatalf("persisted administrator changed: got %+v, want %+v", got, persisted)
	}
}

func TestPersistedSessionSurvivesSameEmailInitialAdminOverlay(t *testing.T) {
	database, ctx := dbtest.Open(t)
	users := directory.NewUserService(directory.NewStore(database))
	persisted, err := users.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Persisted Admin",
		Role:     directory.RoleAdmin,
		Password: "persisted-password",
	})
	if err != nil {
		t.Fatalf("create persisted administrator: %v", err)
	}

	sessions := testSessionManager()
	withoutOverlay := testAuthService(t, users, sessions, InitialAdminConfig{})
	requestCtx := loadTestSession(t, sessions, ctx)
	if _, err := withoutOverlay.Login(requestCtx, LoginParams{
		Email:    persisted.Email,
		Password: "persisted-password",
	}); err != nil {
		t.Fatalf("persisted login: %v", err)
	}

	withOverlay := testAuthService(t, users, sessions, InitialAdminConfig{
		Email:    persisted.Email,
		Password: "configured-password",
	})
	principal, err := withOverlay.CurrentPrincipal(requestCtx)
	if err != nil {
		t.Fatalf("restore persisted session: %v", err)
	}
	if principal.UserID == nil || *principal.UserID != persisted.ID {
		t.Fatalf("restored principal = %+v, want persisted user %d", principal, persisted.ID)
	}
}

func TestRemovingInitialAdminInvalidatesItsSession(t *testing.T) {
	sessions := testSessionManager()
	configured := testAuthService(t, nil, sessions, InitialAdminConfig{
		Email:    "admin@example.test",
		Password: "configured-password",
	})
	ctx := loadTestSession(t, sessions, context.Background())
	if _, err := configured.Login(ctx, LoginParams{
		Email:    "admin@example.test",
		Password: "configured-password",
	}); err != nil {
		t.Fatalf("initial administrator login: %v", err)
	}

	disabled := testAuthService(t, nil, sessions, InitialAdminConfig{})
	if _, err := disabled.CurrentPrincipal(ctx); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("CurrentPrincipal error = %v, want %v", err, ErrNotAuthenticated)
	}
}

func TestInitialAdminSessionFollowsConfiguredPrincipalChanges(t *testing.T) {
	sessions := testSessionManager()
	configured := testAuthService(t, nil, sessions, InitialAdminConfig{
		Email:    "first@example.test",
		Password: "configured-password",
	})
	ctx := loadTestSession(t, sessions, context.Background())
	if _, err := configured.Login(ctx, LoginParams{
		Email:    "first@example.test",
		Password: "configured-password",
	}); err != nil {
		t.Fatalf("initial administrator login: %v", err)
	}

	changed := testAuthService(t, nil, sessions, InitialAdminConfig{
		Email:    "second@example.test",
		Password: "different-password",
	})
	principal, err := changed.CurrentPrincipal(ctx)
	if err != nil {
		t.Fatalf("restore changed configured principal: %v", err)
	}
	if principal.Email != "second@example.test" || principal.UserID != nil {
		t.Fatalf("restored principal = %+v, want changed deployment principal", principal)
	}
}

func TestSessionRejectsMixedPrincipalKinds(t *testing.T) {
	sessions := testSessionManager()
	service := testAuthService(t, nil, sessions, InitialAdminConfig{
		Email:    "admin@example.test",
		Password: "configured-password",
	})
	ctx := loadTestSession(t, sessions, context.Background())
	sessions.Put(ctx, sessionPrincipalKindKey, principalKindInitialAdmin)
	sessions.Put(ctx, sessionUserIDKey, int64(7))

	if _, err := service.CurrentPrincipal(ctx); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("CurrentPrincipal error = %v, want %v", err, ErrNotAuthenticated)
	}
}

func TestSessionRejectsInitialAdminWithZeroUserIDField(t *testing.T) {
	sessions := testSessionManager()
	service := testAuthService(t, nil, sessions, InitialAdminConfig{
		Email:    "admin@example.test",
		Password: "configured-password",
	})
	ctx := loadTestSession(t, sessions, context.Background())
	sessions.Put(ctx, sessionPrincipalKindKey, principalKindInitialAdmin)
	sessions.Put(ctx, sessionUserIDKey, int64(0))

	if _, err := service.CurrentPrincipal(ctx); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("CurrentPrincipal error = %v, want %v", err, ErrNotAuthenticated)
	}
}

func TestMissingLoginPerformsDummyPasswordVerification(t *testing.T) {
	database, ctx := dbtest.Open(t)
	users := directory.NewUserService(directory.NewStore(database))
	sessions := testSessionManager()
	service := testAuthService(t, users, sessions, InitialAdminConfig{})
	service.dummyHash = "not-an-argon2-hash"
	requestCtx := loadTestSession(t, sessions, ctx)

	_, err := service.Login(requestCtx, LoginParams{
		Email:    "missing@example.test",
		Password: "wrong-password",
	})
	if err == nil || !strings.HasPrefix(err.Error(), "verify password: ") {
		t.Fatalf("Login error = %v, want dummy verification error", err)
	}
}

func testAuthService(
	t *testing.T,
	users *directory.UserService,
	sessions *scs.SessionManager,
	initialConfig InitialAdminConfig,
) *Service {
	t.Helper()
	service, err := NewService(users, sessions, initialConfig)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}
	return service
}

func testSessionManager() *scs.SessionManager {
	sessions := scs.New()
	sessions.Store = memstore.New()
	return sessions
}

func loadTestSession(t *testing.T, sessions *scs.SessionManager, ctx context.Context) context.Context {
	t.Helper()
	ctx, err := sessions.Load(ctx, "")
	if err != nil {
		t.Fatalf("load test session: %v", err)
	}
	return ctx
}
