package auth

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/labels"
)

func TestSetupIgnoresUsersWithoutAdministratorAccess(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := directory.NewStore(database)
	provider := directory.NewProviderService(store, labels.NewStore(database))
	if err := provider.ApplyProviderSnapshot(ctx, directory.SourceEntra, directory.ProviderSnapshot{
		Users: []directory.ProviderUser{{
			ExternalID:        "entra-user",
			UserPrincipalName: "entra@example.test",
			DisplayName:       "Entra User",
			Enabled:           true,
		}},
	}); err != nil {
		t.Fatalf("apply Entra snapshot: %v", err)
	}

	service, sessions := testAuthService(store)
	complete, err := service.SetupComplete(ctx)
	if err != nil {
		t.Fatalf("check setup state: %v", err)
	}
	if complete {
		t.Fatal("setup is complete with only a roleless Entra user")
	}

	requestCtx := loadTestSession(t, sessions, ctx)
	user, err := service.Setup(requestCtx, SetupParams{
		Email:    "admin@example.test",
		Name:     "Initial Admin",
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("complete setup: %v", err)
	}
	if user.Role == nil || *user.Role != directory.RoleAdmin {
		t.Fatalf("setup role = %v, want admin", user.Role)
	}

	complete, err = service.SetupComplete(ctx)
	if err != nil {
		t.Fatalf("check completed setup state: %v", err)
	}
	if !complete {
		t.Fatal("setup is incomplete after creating an active administrator")
	}
}

func TestConcurrentSetupCreatesOneAdministrator(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := directory.NewStore(database)
	service, sessions := testAuthService(store)

	type result struct {
		user *directory.User
		err  error
	}
	const attempts = 2
	emails := [attempts]string{"admin-a@example.test", "admin-b@example.test"}
	requestContexts := [attempts]context.Context{
		loadTestSession(t, sessions, ctx),
		loadTestSession(t, sessions, ctx),
	}
	start := make(chan struct{})
	results := make(chan result, attempts)
	var wg sync.WaitGroup
	for i := range attempts {
		wg.Go(func() {
			<-start
			user, err := service.Setup(requestContexts[i], SetupParams{
				Email:    emails[i],
				Name:     "Initial Admin",
				Password: "correct-password",
			})
			results <- result{user: user, err: err}
		})
	}
	close(start)
	wg.Wait()
	close(results)

	var created int
	var rejected int
	for result := range results {
		switch {
		case result.err == nil:
			created++
		case errors.Is(result.err, ErrAlreadySetup):
			rejected++
		default:
			t.Fatalf("setup returned unexpected error: %v", result.err)
		}
	}
	if created != 1 || rejected != 1 {
		t.Fatalf("setup results: created=%d rejected=%d, want 1 each", created, rejected)
	}

	var administrators int
	if err := database.Pool().QueryRow(ctx, `
SELECT count(*)
FROM users
WHERE role = 'admin'
  AND deleted_at IS NULL`).Scan(&administrators); err != nil {
		t.Fatalf("count active administrators: %v", err)
	}
	if administrators != 1 {
		t.Fatalf("active administrators = %d, want 1", administrators)
	}
}

func TestSetupStaysCompleteWithoutAnActiveAdministrator(t *testing.T) {
	database, ctx := dbtest.Open(t)
	store := directory.NewStore(database)
	service, sessions := testAuthService(store)
	requestCtx := loadTestSession(t, sessions, ctx)

	user, err := service.Setup(requestCtx, SetupParams{
		Email:    "admin@example.test",
		Name:     "Initial Admin",
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("complete setup: %v", err)
	}
	if _, err := database.Pool().Exec(ctx, `
UPDATE users
SET role = NULL
WHERE id = $1`, user.ID); err != nil {
		t.Fatalf("revoke administrator access: %v", err)
	}

	complete, err := service.SetupComplete(ctx)
	if err != nil {
		t.Fatalf("check setup state: %v", err)
	}
	if !complete {
		t.Fatal("setup reopened after the final administrator was revoked")
	}
	_, err = service.Setup(requestCtx, SetupParams{
		Email:    "takeover@example.test",
		Name:     "Takeover",
		Password: "correct-password",
	})
	if !errors.Is(err, ErrAlreadySetup) {
		t.Fatalf("second Setup error = %v, want %v", err, ErrAlreadySetup)
	}
}

func testAuthService(store *directory.Store) (*Service, *scs.SessionManager) {
	sessions := scs.New()
	sessions.Store = memstore.New()
	return NewService(directory.NewUserService(store, testDerivedLabelRefresher{}), sessions), sessions
}

type testDerivedLabelRefresher struct{}

func (testDerivedLabelRefresher) RefreshDerivedTx(context.Context, pgx.Tx) error {
	return nil
}

func loadTestSession(t *testing.T, sessions *scs.SessionManager, ctx context.Context) context.Context {
	t.Helper()
	ctx, err := sessions.Load(ctx, "")
	if err != nil {
		t.Fatalf("load test session: %v", err)
	}
	return ctx
}
