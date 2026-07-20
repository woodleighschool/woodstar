package handlers

import (
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
	"github.com/woodleighschool/woodstar/internal/directory"
)

func TestLoginRateLimitPrecedesRequestValidation(t *testing.T) {
	sessions := testSessionManager()
	authService, err := auth.NewService(nil, sessions)
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
