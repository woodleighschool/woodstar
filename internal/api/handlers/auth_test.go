package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

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
	authService, err := auth.NewService(userService, testSessionManager(), auth.InitialAdminConfig{})
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}

	router := authTestRouter(authService, userService)
	rec := authTestLogin(router, "192.0.2.40:1234", "admin@example.test")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("invalid email or password")) {
		t.Fatalf("body = %q, want invalid credential message", rec.Body.String())
	}

	missing := authTestLogin(router, "192.0.2.40:1234", "missing@example.test")
	if missing.Code != rec.Code || missing.Body.String() != rec.Body.String() {
		t.Fatalf(
			"missing-user response = %d %q, want known-user response %d %q",
			missing.Code,
			missing.Body.String(),
			rec.Code,
			rec.Body.String(),
		)
	}
}

func TestLoginRateLimitResponse(t *testing.T) {
	database, _ := dbtest.Open(t)
	userService := directory.NewUserService(directory.NewStore(database))
	authService, err := auth.NewService(userService, testSessionManager(), auth.InitialAdminConfig{})
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}
	router := authTestRouter(authService, userService)

	for attempt := 1; attempt <= 5; attempt++ {
		rec := authTestLogin(router, "192.0.2.41:1234", "missing@example.test")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want %d", attempt, rec.Code, http.StatusUnauthorized)
		}
	}
	rec := authTestLogin(router, "192.0.2.41:1234", "missing@example.test")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("limited status = %d, want %d; body = %q", rec.Code, http.StatusTooManyRequests, rec.Body.String())
	}
}

func authTestRouter(authService *auth.Service, userService *directory.UserService) *chi.Mux {
	router := chi.NewRouter()
	router.Use(chimiddleware.ClientIPFromRemoteAddr)
	humaAPI := humachi.New(router, testHumaConfig())
	RegisterAuth(AuthHandlerDeps{
		Public:      humaAPI,
		Session:     humaAPI,
		Protected:   humaAPI,
		Router:      router,
		AuthService: authService,
		Users:       userService,
		Logger:      discardLogger(),
	})
	return router
}

func authTestLogin(router *chi.Mux, remoteAddr, email string) *httptest.ResponseRecorder {
	body := strings.NewReader(`{"email":"` + email + `","password":"wrong-password"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, sessionPath, body)
	req.RemoteAddr = remoteAddr
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
	cfg.Components = &huma.Components{
		Schemas: huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer),
	}
	return cfg
}
