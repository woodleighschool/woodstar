package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/woodleighschool/woodstar/internal/config"
)

// TestClientIPHeaderSourceUsesTrustedHeaderOverXFF proves the header source
// reads only the configured proxy header and ignores an attacker-supplied
// X-Forwarded-For, which is the safe choice behind Cloudflare.
func TestClientIPHeaderSourceUsesTrustedHeaderOverXFF(t *testing.T) {
	cfg := config.Config{
		ClientIPSource: config.ClientIPSourceHeader,
		ClientIPHeader: "CF-Connecting-IP",
	}

	var got string
	handler := clientIPMiddleware(cfg)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = chimiddleware.GetClientIP(r.Context())
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	// Canonical form of Cloudflare's CF-Connecting-IP header.
	req.Header.Set("Cf-Connecting-Ip", "203.0.113.7")
	req.Header.Set("X-Forwarded-For", "10.9.9.9")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if got != "203.0.113.7" {
		t.Fatalf("client IP = %q, want trusted header IP 203.0.113.7", got)
	}
}
