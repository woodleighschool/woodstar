package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/webui"
)

func TestServerMountsComposedRoutes(t *testing.T) {
	sessions := scs.New()
	sessions.Store = memstore.New()
	logger := slog.New(slog.DiscardHandler)

	server := NewServer(ServerOptions{
		Config: config.Config{
			ClientIPSource: config.ClientIPSourceRemoteAddr,
		},
		Ready:          func(context.Context) error { return nil },
		Version:        "test",
		Logger:         logger,
		SessionManager: sessions,
		WebHandler: webui.NewHandler(webui.HandlerOptions{
			FS: fstest.MapFS{
				"index.html": {Data: []byte("<!doctype html><html><body>web</body></html>")},
			},
			Version:   "test",
			ServerURL: "https://woodstar.example",
			Logger:    logger,
		}),
		RegisterRoutes: func(routes Routes) {
			routes.App.Router.Get("/api/composed", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("composed\n"))
			})
		},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/composed", nil)
	server.httpServer.Handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || recorder.Body.String() != "composed\n" {
		t.Fatalf("response = %d %q, want composed route", recorder.Code, recorder.Body.String())
	}
}

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
