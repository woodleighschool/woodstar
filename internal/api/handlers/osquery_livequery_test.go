package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
)

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
		{name: "create", method: http.MethodPost, path: "/api/live-queries", wantSurface: "ordinary"},
		{name: "stream", method: http.MethodGet, path: "/api/live-queries/1/stream", wantSurface: "streaming"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, nil))
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

func TestLiveQueryStreamReturnsNotFoundBeforeStreaming(t *testing.T) {
	router := chi.NewRouter()
	api := humachi.New(router, testHumaConfig())
	registerLiveQueries(api, api, livequery.NewManager(), nil, discardLogger())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/live-queries/404/stream", nil)
	request.Header.Set("Accept", "text/event-stream")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %q", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want a regular error response", contentType)
	}
}

func TestLiveQueryStreamReplaysCompletedResults(t *testing.T) {
	manager := livequery.NewManager()
	handle := manager.Start("select 1", []int64{4})
	manager.RecordResult(livequery.Result{
		QueryID:  handle.ID,
		HostID:   4,
		HostName: "mac-4",
		Status:   livequery.StatusSuccess,
	})
	router := chi.NewRouter()
	api := humachi.New(router, testHumaConfig())
	registerLiveQueries(api, api, manager, nil, discardLogger())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/live-queries/1/stream", nil)
	request.Header.Set("Accept", "text/event-stream")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	body := recorder.Body.String()
	resultAt := strings.Index(body, "event: result")
	completedAt := strings.Index(body, "event: completed")
	if resultAt < 0 || completedAt < 0 || resultAt > completedAt {
		t.Fatalf("SSE body = %q, want replayed result before completion", body)
	}
}
