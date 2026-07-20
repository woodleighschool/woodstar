package protocol

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutesSelectsWebSocketSurface(t *testing.T) {
	t.Parallel()
	router := chi.NewRouter()
	ordinary := router.With(routeSurface("ordinary"))
	websocket := router.With(routeSurface("websocket"))
	(&Server{}).RegisterRoutes(ordinary, websocket)

	for _, tc := range []struct {
		path        string
		wantSurface string
	}{
		{path: "/api/munki/distribution/packages/1/download-url", wantSurface: "ordinary"},
		{path: "/api/munki/distribution/connect", wantSurface: "websocket"},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil))
		if got := recorder.Header().Get("X-Route-Surface"); got != tc.wantSurface {
			t.Errorf("%s route surface = %q, want %q", tc.path, got, tc.wantSurface)
		}
	}
}

func routeSurface(surface string) func(http.Handler) http.Handler {
	return func(_ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Route-Surface", surface)
			w.WriteHeader(http.StatusNoContent)
		})
	}
}
