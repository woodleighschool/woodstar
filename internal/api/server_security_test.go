package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/go-chi/chi/v5"

	apimiddleware "github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/webui"
)

func TestSecurityHeadersProtectRenderedSPAForStorageBackend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		storage    storage.Config
		wantOrigin string
	}{
		{
			name: "same-origin file transfers",
			storage: storage.Config{
				Kind:          storage.KindFile,
				FileRoot:      t.TempDir(),
				BaseURL:       "https://woodstar.example",
				CapabilityKey: []byte("test capability key"),
				PresignTTL:    time.Minute,
			},
			wantOrigin: "https://woodstar.example",
		},
		{
			name: "cross-origin S3 transfers",
			storage: storage.Config{
				Kind: storage.KindS3,
				S3: storage.S3Config{
					Bucket:         "woodstar",
					Region:         "ap-southeast-2",
					PublicEndpoint: "https://uploads.example",
					AccessKey:      "test-access-key",
					SecretKey:      "test-secret-key",
					PathStyle:      true,
					PresignTTL:     time.Minute,
				},
			},
			wantOrigin: "https://uploads.example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			backend, err := storage.New(t.Context(), tt.storage)
			if err != nil {
				t.Fatalf("storage.New: %v", err)
			}

			router := chi.NewRouter()
			router.Use(apimiddleware.SecurityHeaders(backend.TransferOrigin()))
			webui.NewHandler(webui.HandlerOptions{
				FS: fstest.MapFS{
					"index.html": {Data: []byte("<!doctype html><html><head></head><body></body></html>")},
				},
				Version:   "test",
				ServerURL: "https://woodstar.example",
				Logger:    slog.New(slog.DiscardHandler),
			}).RegisterRoutes(router)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/hosts/1", nil))
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
