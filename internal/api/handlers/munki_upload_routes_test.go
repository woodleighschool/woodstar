package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
			name:        "complete multipart",
			method:      http.MethodPost,
			path:        munkiPackageInstallerPath + "/1/multipart/complete",
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
