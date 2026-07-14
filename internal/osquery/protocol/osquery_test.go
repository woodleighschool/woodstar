package protocol

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestOsqueryRoutesRejectMalformedAndOversizedJSON(t *testing.T) {
	router := chi.NewRouter()
	NewServer(nil, slog.New(slog.DiscardHandler)).RegisterRoutes(router)

	tests := []struct {
		name       string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "trailing JSON",
			path:       osqueryPath + "/config",
			body:       `{"node_key":"key"}{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "config over one MiB",
			path:       osqueryPath + "/config",
			body:       oversizedJSON(osqueryRequestMaxBytes),
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "distributed write over five MiB",
			path:       osqueryPath + "/distributed/write",
			body:       oversizedJSON(osqueryDistributedWriteMaxBytes),
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "log over ten MiB",
			path:       osqueryPath + "/log",
			body:       oversizedJSON(osqueryLogMaxBytes),
			wantStatus: http.StatusRequestEntityTooLarge,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestOsqueryOnlyExposesCurrentRoutes(t *testing.T) {
	router := chi.NewRouter()
	NewServer(nil, slog.New(slog.DiscardHandler)).RegisterRoutes(router)

	for _, path := range []string{
		"/api/osquery/config",
		osqueryPath + "/carve/begin",
		osqueryPath + "/carve/block",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("POST %s status = %d, want %d", path, rec.Code, http.StatusNotFound)
		}
	}
}

func oversizedJSON(limit int64) string {
	return `{"padding":"` + strings.Repeat("x", int(limit)) + `"}`
}
