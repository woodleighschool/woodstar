//go:build postgres

package auth

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestSSOLoginStartsPersistedUserSession(t *testing.T) {
	database, ctx := testdb.Open(t)
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
