package httpapi

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestCreateLiveQueryRejectsBlankSQL(t *testing.T) {
	router := chi.NewRouter()
	humaAPI := humachi.New(router, testHumaConfig())
	registerLiveQueries(humaAPI, humaAPI, nil, nil, discardLogger())

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/osquery/live-queries",
		strings.NewReader(`{"sql":" \n "}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %q", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestLiveQueryRoutesSelectStreamingSurface(t *testing.T) {
	t.Parallel()
	router := chi.NewRouter()
	ordinary := humachi.New(
		router.With(routeSurfaceMiddleware("ordinary")),
		testHumaConfigWithoutUtilityRoutes(),
	)
	streaming := humachi.New(
		router.With(routeSurfaceMiddleware("streaming")),
		testHumaConfigWithoutUtilityRoutes(),
	)
	registerLiveQueries(ordinary, streaming, nil, nil, discardLogger())

	for _, tc := range []struct {
		name        string
		method      string
		path        string
		wantSurface string
	}{
		{name: "create", method: http.MethodPost, path: "/api/osquery/live-queries", wantSurface: "ordinary"},
		{name: "delete", method: http.MethodDelete, path: "/api/osquery/live-queries/1", wantSurface: "ordinary"},
		{name: "stream", method: http.MethodGet, path: "/api/osquery/live-queries/1/stream", wantSurface: "streaming"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, nil))
			if got := recorder.Header().Get("X-Route-Surface"); got != tc.wantSurface {
				t.Fatalf("route surface = %q, want %q", got, tc.wantSurface)
			}
		})
	}
}

func routeSurfaceMiddleware(surface string) func(http.Handler) http.Handler {
	return func(_ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Route-Surface", surface)
			w.WriteHeader(http.StatusNoContent)
		})
	}
}

func testHumaConfigWithoutUtilityRoutes() huma.Config {
	cfg := testHumaConfig()
	cfg.OpenAPIPath = ""
	cfg.DocsPath = ""
	cfg.SchemasPath = ""
	return cfg
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
