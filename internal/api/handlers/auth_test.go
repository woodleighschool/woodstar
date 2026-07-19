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

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/directory"
)

func TestLoginInvalidCredentialsMessage(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := directory.NewUserService(directory.NewStore(database))
	if _, err := userService.CreateInitialAdministrator(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Test Admin",
		Password: "correct-password",
	}); err != nil {
		t.Fatalf("complete test setup: %v", err)
	}

	router := chi.NewRouter()
	humaAPI := humachi.New(router, testHumaConfig())
	RegisterAuth(AuthHandlerDeps{
		Public:      humaAPI,
		Session:     humaAPI,
		Protected:   humaAPI,
		Router:      router,
		AuthService: auth.NewService(userService, testSessionManager()),
		Users:       userService,
		Logger:      discardLogger(),
	})

	body := strings.NewReader(`{"email":"admin@example.test","password":"wrong-password"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, sessionPath, body)
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("invalid email or password")) {
		t.Fatalf("body = %q, want invalid credential message", rec.Body.String())
	}
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
