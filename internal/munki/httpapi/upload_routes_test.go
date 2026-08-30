package httpapi

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestPackageInstallerRoutesSelectLongRunningSurface(t *testing.T) {
	t.Parallel()
	router := chi.NewRouter()
	ordinary := humachi.New(
		router.With(routeSurfaceMiddleware("ordinary")),
		testHumaConfigWithoutUtilityRoutes(),
	)
	longRunning := humachi.New(
		router.With(routeSurfaceMiddleware("long-running")),
		testHumaConfigWithoutUtilityRoutes(),
	)
	registerPackageInstallerRoutes(ordinary, longRunning, nil, discardLogger())

	for _, tc := range []struct {
		name        string
		method      string
		path        string
		wantSurface string
	}{
		{name: "create", method: http.MethodPost, path: munkiPackageInstallerPath, wantSurface: "ordinary"},
		{name: "finalize", method: http.MethodPut, path: munkiPackageInstallerPath + "/1", wantSurface: "long-running"},
		{name: "delete", method: http.MethodDelete, path: munkiPackageInstallerPath + "/1", wantSurface: "ordinary"},
		{
			name:        "sign multipart part",
			method:      http.MethodPost,
			path:        munkiPackageInstallerPath + "/1/multipart/parts/1",
			wantSurface: "ordinary",
		},
		{
			name:        "complete multipart",
			method:      http.MethodPut,
			path:        munkiPackageInstallerPath + "/1/multipart",
			wantSurface: "long-running",
		},
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
