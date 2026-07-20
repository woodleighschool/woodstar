//go:build postgres

package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"

	apimiddleware "github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestLoginInvalidCredentialsMessage(t *testing.T) {
	database, ctx := testdb.Open(t)
	userService := directory.NewUserService(directory.NewStore(database))
	if _, err := userService.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Test Admin",
		Role:     directory.RoleAdmin,
		Password: "correct-password",
	}); err != nil {
		t.Fatalf("create test administrator: %v", err)
	}
	sessions := testSessionManager()
	authService, err := auth.NewService(userService, sessions)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}

	router := authTestRouter(
		authService,
		userService,
		sessions,
		apimiddleware.NewPasswordLoginLimiter(),
	)
	started := time.Now()
	rec := authTestLogin(router, "admin@example.test", "wrong-password")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("invalid email or password")) {
		t.Fatalf("body = %q, want invalid credential message", rec.Body.String())
	}
	if elapsed := time.Since(started); elapsed < time.Second {
		t.Fatalf("known-user credential failure took %v, want at least one second", elapsed)
	}

	started = time.Now()
	missing := authTestLogin(router, "missing@example.test", "wrong-password")
	if missing.Code != rec.Code || missing.Body.String() != rec.Body.String() {
		t.Fatalf(
			"missing-user response = %d %q, want known-user response %d %q",
			missing.Code,
			missing.Body.String(),
			rec.Code,
			rec.Body.String(),
		)
	}
	if elapsed := time.Since(started); elapsed < time.Second {
		t.Fatalf("missing-user credential failure took %v, want at least one second", elapsed)
	}
}

func TestSuccessfulPasswordLoginConsumesCapacityWithoutLimitingOIDC(t *testing.T) {
	database, ctx := testdb.Open(t)
	userService := directory.NewUserService(directory.NewStore(database))
	if _, err := userService.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Test Admin",
		Role:     directory.RoleAdmin,
		Password: "configured-password",
	}); err != nil {
		t.Fatalf("create test administrator: %v", err)
	}
	sessions := testSessionManager()
	authService, err := auth.NewService(userService, sessions)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}
	router := authTestRouter(
		authService,
		userService,
		sessions,
		rate.NewLimiter(rate.Every(time.Hour), 1),
	)

	success := authTestLogin(router, "admin@example.test", "configured-password")
	if success.Code != http.StatusOK {
		t.Fatalf("successful login status = %d, want %d; body = %q", success.Code, http.StatusOK, success.Body.String())
	}

	limited := authTestLogin(router, "admin@example.test", "configured-password")
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("second login status = %d, want %d", limited.Code, http.StatusTooManyRequests)
	}

	oidc := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/sso/start", nil)
	router.ServeHTTP(oidc, req)
	if oidc.Code != http.StatusNotFound {
		t.Fatalf("OIDC start status = %d, want %d from unconfigured OIDC", oidc.Code, http.StatusNotFound)
	}
}
