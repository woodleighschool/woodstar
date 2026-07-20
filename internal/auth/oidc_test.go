package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/directory"
)

func TestCompleteSSORejectsMissingSessionNonce(t *testing.T) {
	sessions := scs.New()
	sessions.Store = memstore.New()
	ctx, err := sessions.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	sessions.Put(ctx, ssoStateSessionKey, "expected-state")

	service := &Service{
		sessions: sessions,
		oidc:     &oidcProvider{},
	}
	if _, err := service.CompleteSSO(ctx, "expected-state", "code"); !errors.Is(err, ErrSSONonceMismatch) {
		t.Fatalf("CompleteSSO error = %v, want %v", err, ErrSSONonceMismatch)
	}
}

func TestSSOLoginStartsPersistedUserSession(t *testing.T) {
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

	if _, err := service.completeSSOLogin(requestCtx, "ADMIN@EXAMPLE.TEST"); !errors.Is(
		err,
		ErrSSOUnknownUser,
	) {
		t.Fatalf("uppercase SSO login error = %v, want %v", err, ErrSSOUnknownUser)
	}
	loggedIn, err := service.completeSSOLogin(requestCtx, "admin@example.test")
	if err != nil {
		t.Fatalf("complete SSO login: %v", err)
	}
	if loggedIn.ID != created.ID {
		t.Fatalf("SSO user ID = %d, want %d", loggedIn.ID, created.ID)
	}
	restored, err := service.CurrentUser(requestCtx)
	if err != nil || restored.ID != created.ID {
		t.Fatalf("restored SSO user = %+v, error = %v", restored, err)
	}
}
