package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"golang.org/x/time/rate"

	apimiddleware "github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/directory"
)

func TestLoginInvalidCredentialsMessage(t *testing.T) {
	database, ctx := dbtest.Open(t)
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
	authService, err := auth.NewService(userService, sessions, auth.InitialAdminConfig{})
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

func TestLoginRateLimitPrecedesRequestValidation(t *testing.T) {
	sessions := testSessionManager()
	authService, err := auth.NewService(nil, sessions, auth.InitialAdminConfig{
		Email:    "admin@example.test",
		Password: "configured-password",
	})
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}
	router := authTestRouter(
		authService,
		nil,
		sessions,
		rate.NewLimiter(rate.Every(time.Hour), 1),
	)

	malformed := authTestRequest(router, `{}`)
	if malformed.Code != http.StatusUnprocessableEntity {
		t.Fatalf("admitted malformed status = %d, want %d", malformed.Code, http.StatusUnprocessableEntity)
	}

	rec := authTestLogin(router, "different@example.test", "wrong-password")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("limited status = %d, want %d; body = %q", rec.Code, http.StatusTooManyRequests, rec.Body.String())
	}
	retryAfter, err := strconv.Atoi(rec.Header().Get("Retry-After"))
	if err != nil || retryAfter < 1 {
		t.Fatalf("Retry-After = %q, want positive integer seconds", rec.Header().Get("Retry-After"))
	}
}

func TestSuccessfulPasswordLoginConsumesCapacityWithoutLimitingOIDC(t *testing.T) {
	sessions := testSessionManager()
	authService, err := auth.NewService(nil, sessions, auth.InitialAdminConfig{
		Email:    "admin@example.test",
		Password: "configured-password",
	})
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}
	router := authTestRouter(
		authService,
		nil,
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

func authTestRouter(
	authService *auth.Service,
	userService *directory.UserService,
	sessions *scs.SessionManager,
	loginLimiter *rate.Limiter,
) *chi.Mux {
	router := chi.NewRouter()
	config := testHumaConfig()
	passwordLoginRouter := router.With(
		apimiddleware.LimitPasswordLogin(loginLimiter),
		sessions.LoadAndSave,
	)
	ordinaryRouter := router.With(sessions.LoadAndSave)
	passwordLoginAPI := humachi.New(passwordLoginRouter, config)
	humaAPI := humachi.New(ordinaryRouter, config)
	RegisterAuth(AuthHandlerDeps{
		PasswordLogin: passwordLoginAPI,
		Session:       humaAPI,
		Protected:     humaAPI,
		Router:        ordinaryRouter,
		AuthService:   authService,
		Users:         userService,
		Logger:        discardLogger(),
	})
	return router
}

func authTestLogin(router *chi.Mux, email, password string) *httptest.ResponseRecorder {
	return authTestRequest(
		router,
		`{"email":"`+email+`","password":"`+password+`"}`,
	)
}

func authTestRequest(router *chi.Mux, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		sessionPath,
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}

func testSessionManager() *scs.SessionManager {
	sm := scs.New()
	sm.Store = memstore.New()
	return sm
}

func testHumaConfig() huma.Config {
	cfg := huma.DefaultConfig("test", "test")
	cfg.OpenAPIPath = ""
	cfg.DocsPath = ""
	cfg.SchemasPath = ""
	cfg.Components = &huma.Components{
		Schemas: huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer),
	}
	return cfg
}
