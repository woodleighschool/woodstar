package protocol

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestPingReportsSupportedCapabilities(t *testing.T) {
	router := chi.NewRouter()
	NewServer(nil, slog.New(slog.DiscardHandler)).RegisterRoutes(router)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/api/fleet/orbit/ping", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	got := rec.Header().Get(capabilitiesHeader)
	if got != orbitCapabilitiesValue {
		t.Fatalf("capabilities = %q, want %q", got, orbitCapabilitiesValue)
	}
}

func TestOrbitRoutesRejectMalformedAndOversizedJSON(t *testing.T) {
	router := chi.NewRouter()
	NewServer(nil, slog.New(slog.DiscardHandler)).RegisterRoutes(router)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "trailing JSON",
			body:       `{"orbit_node_key":"key"}{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "oversized body",
			body:       `{"padding":"` + strings.Repeat("x", orbitRequestMaxBytes) + `"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/fleet/orbit/config", strings.NewReader(tt.body))
			router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}
