//go:build postgres

package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestHostSantaRulesEndpointReturnsNotFoundForMissingHost(t *testing.T) {
	db, _ := testdb.Open(t)
	ruleStore := rules.NewStore(db)
	router := santaRulesAPI(t, func(humaAPI huma.API) {
		registerHostSantaRules(humaAPI, ruleStore, discardLogger())
	})

	rec := santaRulesRequest(t, router, http.MethodGet, "/api/hosts/999999/santa/rules", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing host status = %d, want %d; body = %q", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func santaRulesAPI(t *testing.T, register func(huma.API)) *chi.Mux {
	t.Helper()

	router := chi.NewRouter()
	humaAPI := humachi.New(router, huma.DefaultConfig("test", "test"))
	register(humaAPI)
	return router
}

func santaRulesRequest(t *testing.T, router *chi.Mux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(rec, req)
	return rec
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
