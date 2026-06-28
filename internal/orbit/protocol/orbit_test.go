package protocol

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
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
