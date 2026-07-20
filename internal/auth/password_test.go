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

func TestPersistedUserLoginStartsSession(t *testing.T) {
	database, ctx := dbtest.Open(t)
	users := directory.NewUserService(directory.NewStore(database))
	created, err := users.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Persisted Admin",
		Role:     directory.RoleAdmin,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	sessions := testSessionManager()
	service := testAuthService(t, users, sessions)
	requestCtx := loadTestSession(t, sessions, ctx)
	if _, err := service.Login(requestCtx, LoginParams{
		Email:    "ADMIN@EXAMPLE.TEST",
		Password: "correct-password",
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("uppercase login error = %v, want %v", err, ErrInvalidCredentials)
	}
	loggedIn, err := service.Login(requestCtx, LoginParams{
		Email:    " admin@example.test ",
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loggedIn.ID != created.ID {
		t.Fatalf("logged-in user ID = %d, want %d", loggedIn.ID, created.ID)
	}

	restored, err := service.CurrentUser(requestCtx)
	if err != nil {
		t.Fatalf("restore session user: %v", err)
	}
	if restored.ID != created.ID || restored.Email != created.Email {
		t.Fatalf("restored user = %+v, want %+v", restored, created)
	}
}

func TestSessionReloadsPersistedUser(t *testing.T) {
	database, ctx := dbtest.Open(t)
	users := directory.NewUserService(directory.NewStore(database))
	created, err := users.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Original Name",
		Role:     directory.RoleAdmin,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	sessions := testSessionManager()
	service := testAuthService(t, users, sessions)
	requestCtx := loadTestSession(t, sessions, ctx)
	if _, err := service.Login(requestCtx, LoginParams{
		Email:    created.Email,
		Password: "correct-password",
	}); err != nil {
		t.Fatalf("login: %v", err)
	}

	viewer := directory.RoleViewer
	if _, err := users.Update(ctx, created.ID, directory.UserMutation{
		Name: "Updated Name",
		Role: &viewer,
	}); err != nil {
		t.Fatalf("update user: %v", err)
	}
	restored, err := service.CurrentUser(requestCtx)
	if err != nil {
		t.Fatalf("restore updated session user: %v", err)
	}
	if restored.ID != created.ID || restored.Name != "Updated Name" ||
		restored.Role == nil || *restored.Role != directory.RoleViewer {
		t.Fatalf("restored user = %+v, want updated persisted user", restored)
	}
}

func TestDeletedUserSessionIsRejected(t *testing.T) {
	database, ctx := dbtest.Open(t)
	users := directory.NewUserService(directory.NewStore(database))
	created, err := users.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Persisted Admin",
		Role:     directory.RoleAdmin,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	sessions := testSessionManager()
	service := testAuthService(t, users, sessions)
	requestCtx := loadTestSession(t, sessions, ctx)
	if _, err := service.Login(requestCtx, LoginParams{
		Email:    created.Email,
		Password: "correct-password",
	}); err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := users.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	if _, err := service.CurrentUser(requestCtx); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("CurrentUser error = %v, want %v", err, ErrNotAuthenticated)
	}
}

func TestMissingLoginPerformsDummyPasswordVerification(t *testing.T) {
	database, ctx := dbtest.Open(t)
	users := directory.NewUserService(directory.NewStore(database))
	sessions := testSessionManager()
	service := testAuthService(t, users, sessions)
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
) *Service {
	t.Helper()
	service, err := NewService(users, sessions)
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
