package directory

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestUserMutationsPreserveLastAdministrator(t *testing.T) {
	database, ctx := dbtest.Open(t)
	service := newTestUserService(NewStore(database))
	admin, err := service.Create(ctx, UserCreate{
		Email:    "admin@example.test",
		Name:     "Admin",
		Role:     RoleAdmin,
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}

	viewer := RoleViewer
	if _, err := service.Update(
		ctx,
		admin.ID,
		UserMutation{Name: admin.Name, Role: &viewer},
	); !errors.Is(
		err,
		ErrLastAdministrator,
	) {
		t.Fatalf("demote last admin error = %v, want %v", err, ErrLastAdministrator)
	}
	if err := service.Delete(ctx, admin.ID); !errors.Is(err, ErrLastAdministrator) {
		t.Fatalf("delete last admin error = %v, want %v", err, ErrLastAdministrator)
	}

	persisted, err := service.Get(ctx, admin.ID)
	if err != nil {
		t.Fatalf("get preserved admin: %v", err)
	}
	if persisted.Role == nil || *persisted.Role != RoleAdmin {
		t.Fatalf("preserved role = %v, want admin", persisted.Role)
	}
}

func TestConcurrentDemotionsPreserveOneAdministrator(t *testing.T) {
	database, ctx := dbtest.Open(t)
	service := newTestUserService(NewStore(database))
	admins := make([]*User, 2)
	for i, email := range []string{"first-admin@example.test", "second-admin@example.test"} {
		admin, err := service.Create(ctx, UserCreate{
			Email:    email,
			Name:     email,
			Role:     RoleAdmin,
			Password: "correct-password",
		})
		if err != nil {
			t.Fatalf("create admin %d: %v", i, err)
		}
		admins[i] = admin
	}

	viewer := RoleViewer
	start := make(chan struct{})
	errs := make(chan error, len(admins))
	var ready sync.WaitGroup
	ready.Add(len(admins))
	for _, admin := range admins {
		go func() {
			ready.Done()
			<-start
			_, err := service.Update(context.Background(), admin.ID, UserMutation{
				Name: admin.Name,
				Role: &viewer,
			})
			errs <- err
		}()
	}
	ready.Wait()
	close(start)

	var updated, rejected int
	for range admins {
		switch err := <-errs; {
		case err == nil:
			updated++
		case errors.Is(err, ErrLastAdministrator):
			rejected++
		default:
			t.Fatalf("concurrent demotion error: %v", err)
		}
	}
	if updated != 1 || rejected != 1 {
		t.Fatalf("demotion results: updated=%d rejected=%d, want 1 each", updated, rejected)
	}

	exists, err := service.ActiveAdministratorExists(ctx)
	if err != nil {
		t.Fatalf("check active administrator: %v", err)
	}
	if !exists {
		t.Fatal("concurrent demotions removed every administrator")
	}
}

func TestCreateHashesPassword(t *testing.T) {
	database, ctx := dbtest.Open(t)
	service := newTestUserService(NewStore(database))

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

func TestCreateInitialAdministratorRejectsCompletedSetup(t *testing.T) {
	database, ctx := dbtest.Open(t)
	service := newTestUserService(NewStore(database))
	if _, err := service.CreateInitialAdministrator(ctx, UserCreate{
		Email:    "admin@example.test",
		Name:     "Admin",
		Password: "correct-password",
	}); err != nil {
		t.Fatalf("create initial administrator: %v", err)
	}

	_, err := service.CreateInitialAdministrator(ctx, UserCreate{
		Email:    "other@example.test",
		Name:     "Other",
		Password: "correct-password",
	})
	if !errors.Is(err, ErrSetupComplete) {
		t.Fatalf("CreateInitialAdministrator error = %v, want %v", err, ErrSetupComplete)
	}
}

func TestCreateRollsBackWhenDerivedLabelsCannotRefresh(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := NewStore(database)
	service := NewUserService(store, failingDerivedLabelRefresher{})

	_, err := service.Create(ctx, UserCreate{
		Email:    "rollback@example.test",
		Name:     "Rollback User",
		Role:     RoleAdmin,
		Password: "correct-password",
	})
	if err == nil {
		t.Fatal("create succeeded despite derived label refresh failure")
	}

	var count int
	if err := database.Pool().QueryRow(
		ctx,
		`SELECT count(*) FROM users WHERE email = 'rollback@example.test'`,
	).Scan(&count); err != nil {
		t.Fatalf("count rolled-back users: %v", err)
	}
	if count != 0 {
		t.Fatalf("persisted users = %d, want 0", count)
	}
}

func TestUpdateHashesPassword(t *testing.T) {
	database, ctx := dbtest.Open(t)
	service := newTestUserService(NewStore(database))
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
	service := newTestUserService(NewStore(database))
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
	service := newTestUserService(NewStore(database))
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

type testDerivedLabelRefresher struct{}

func (testDerivedLabelRefresher) RefreshDerivedTx(context.Context, pgx.Tx) error {
	return nil
}

type failingDerivedLabelRefresher struct{}

func (failingDerivedLabelRefresher) RefreshDerivedTx(context.Context, pgx.Tx) error {
	return errors.New("refresh failed")
}

func newTestUserService(store *Store) *UserService {
	return NewUserService(store, testDerivedLabelRefresher{})
}

func newTestProviderService(store *Store) *ProviderService {
	return NewProviderService(store, testDerivedLabelRefresher{})
}
