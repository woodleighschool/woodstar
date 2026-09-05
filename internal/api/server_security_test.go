package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-chi/chi/v5"

	apimiddleware "github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/webui"
)

func TestSecurityHeadersProtectRenderedSPAForStorageBackend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		wantOrigin string
	}{
		{name: "same-origin file transfers", wantOrigin: "https://woodstar.example"},
		{name: "cross-origin S3 transfers", wantOrigin: "https://uploads.example"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			router.Use(apimiddleware.SecurityHeaders(tt.wantOrigin))
			webui.NewHandler(webui.HandlerOptions{
				FS: fstest.MapFS{
					"index.html": {Data: []byte("<!doctype html><html><head></head><body></body></html>")},
				},
				Version:   "test",
				ServerURL: "https://woodstar.example",
				Logger:    slog.New(slog.DiscardHandler),
			}).RegisterRoutes(router)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/hosts/1", nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
			}
			csp := recorder.Header().Get("Content-Security-Policy")
			if !strings.Contains(csp, "connect-src 'self' "+tt.wantOrigin) {
				t.Fatalf("Content-Security-Policy = %q, want transfer origin in connect-src", csp)
			}
			if !strings.Contains(csp, "img-src 'self' blob: "+tt.wantOrigin) {
				t.Fatalf("Content-Security-Policy = %q, want transfer origin in img-src", csp)
			}
			if strings.Contains(recorder.Body.String(), "window.__WOODSTAR__") {
				t.Fatalf("SPA body included executable runtime config: %q", recorder.Body.String())
			}
		})
	}
}
