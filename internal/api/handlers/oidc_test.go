package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/auth"
)

func TestOIDCStartReturnsNotFoundWhenDisabled(t *testing.T) {
	router := chi.NewRouter()
	service, err := auth.NewService(nil, testSessionManager())
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}
	registerOIDC(router, service, discardLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/sso/start", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestOIDCCallbackRedirectsProviderErrorsToLogin(t *testing.T) {
	router := chi.NewRouter()
	registerOIDC(router, nil, discardLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/sso/callback?error=access_denied", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if got, want := rec.Header().Get("Location"), "/login?sso_error=access_denied"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}

func TestOIDCCallbackRequiresStateAndCode(t *testing.T) {
	router := chi.NewRouter()
	registerOIDC(router, nil, discardLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/sso/callback?state=state-only", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if got, want := rec.Header().Get("Location"), "/login?sso_error=missing+state+or+code"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}
