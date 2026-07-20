package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRequestLogPathUsesRoutePattern(t *testing.T) {
	router := chi.NewRouter()
	router.Get("/api/latest/fleet/device/{token}/ping", func(_ http.ResponseWriter, r *http.Request) {
		if got, want := requestLogPath(r), "/api/latest/fleet/device/{token}/ping"; got != want {
			t.Fatalf("request log path = %q, want %q", got, want)
		}
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/latest/fleet/device/secret-token/ping", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
}

func TestRequestLogPathRedactsUnmatchedDeviceToken(t *testing.T) {
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/latest/fleet/device/secret-token/unknown", nil)
	if got, want := requestLogPath(request), "/api/latest/fleet/device/{token}/unknown"; got != want {
		t.Fatalf("request log path = %q, want %q", got, want)
	}
}
