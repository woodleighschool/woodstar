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
	store := directory.NewStore(database)
	users := directory.NewUserService(store)
	if _, err := users.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Local Admin",
		Role:     directory.RoleAdmin,
		Password: "correct-password",
	}); err != nil {
		t.Fatalf("create local user: %v", err)
	}
	if err := store.ApplyProviderSnapshot(ctx, directory.SourceEntra, directory.ProviderSnapshot{
		Users: []directory.ProviderUser{{
			ExternalID:        "provider-admin",
			UserPrincipalName: "admin@example.test",
			Mail:              "admin@example.test",
			DisplayName:       "Provider Admin",
			Enabled:           true,
		}},
	}); err != nil {
		t.Fatalf("apply provider snapshot: %v", err)
	}
	created, err := users.SetRoleByEmail(ctx, "admin@example.test", directory.RoleAdmin)
	if err != nil {
		t.Fatalf("grant provider role: %v", err)
	}
	sessions := testSessionManager()
	service := testAuthService(t, users, sessions)
	requestCtx := loadTestSession(t, ctx, sessions)

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

func TestSSOLoginRejectsLocalOnlyUser(t *testing.T) {
	database, ctx := testdb.Open(t)
	users := directory.NewUserService(directory.NewStore(database))
	if _, err := users.Create(ctx, directory.UserCreate{
		Email:    "local@example.test",
		Name:     "Local User",
		Role:     directory.RoleAdmin,
		Password: "correct-password",
	}); err != nil {
		t.Fatalf("create local user: %v", err)
	}

	sessions := testSessionManager()
	service := testAuthService(t, users, sessions)
	requestCtx := loadTestSession(t, ctx, sessions)
	if _, err := service.completeSSOLogin(requestCtx, "local@example.test"); !errors.Is(
		err,
		ErrSSOUnknownUser,
	) {
		t.Fatalf("local-only SSO login error = %v, want %v", err, ErrSSOUnknownUser)
	}
}
