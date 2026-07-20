//go:build postgres

package handlers

import (
	"context"
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
	router := santaRulesAPI(t, func(api huma.API) {
		registerHostSantaRules(api, ruleStore, discardLogger())
	})

	rec := santaRulesRequest(t, router, http.MethodGet, "/api/hosts/999999/santa/rules", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing host status = %d, want %d; body = %q", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func santaRulesAPI(t *testing.T, register func(huma.API)) *chi.Mux {
	t.Helper()

	router := chi.NewRouter()
	api := humachi.New(router, huma.DefaultConfig("test", "test"))
	register(api)
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
