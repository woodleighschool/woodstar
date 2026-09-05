//go:build postgres

package account

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/woodleighschool/goodies/auth/authn"
	authhttp "github.com/woodleighschool/goodies/auth/http"
	authhuma "github.com/woodleighschool/goodies/auth/huma"

	"github.com/woodleighschool/woodstar/internal/api"
)

func authTestRouter(
	deps Dependencies,
	sessions *scs.SessionManager,
) *chi.Mux {
	router := chi.NewRouter()
	config := testHumaConfig()
	passwordLoginRouter := router.With(
		deps.Authn.LimitPasswordLogin,
		sessions.LoadAndSave,
	)
	ordinaryRouter := router.With(sessions.LoadAndSave)
	passwordLoginAPI := humachi.New(passwordLoginRouter, config)
	ordinaryAPI := humachi.New(ordinaryRouter, config)
	requestAuth := deps.Authn
	sessionAPI := huma.NewGroup(ordinaryAPI)
	sessionAPI.UseMiddleware(authhuma.OptionalAuth(ordinaryAPI, requestAuth, deps.Logger))
	protectedAPI := huma.NewGroup(ordinaryAPI)
	protectedAPI.UseMiddleware(authhuma.RequireAuth(ordinaryAPI, requestAuth, deps.Logger))
	RegisterAPI(api.AppRoutes{
		PasswordLogin: passwordLoginAPI,
		Session:       sessionAPI,
		Logout:        ordinaryAPI,
		Protected:     protectedAPI,
		Router:        ordinaryRouter,
	}, deps)
	ordinaryRouter.With(authhttp.RequireAuth(requestAuth, deps.Logger)).Get("/content", func(w http.ResponseWriter, r *http.Request) {
		if _, err := authn.RequirePrincipal(r.Context()); err != nil {
			http.Error(w, "principal missing", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
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
		"/api/session",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}

func testSessionManager() *scs.SessionManager {
	return &scs.SessionManager{
		Store: memstore.NewWithCleanupInterval(0), Codec: scs.GobCodec{}, Lifetime: time.Hour,
		Cookie: scs.SessionCookie{Name: "session", Path: "/", HttpOnly: true},
	}
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

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
