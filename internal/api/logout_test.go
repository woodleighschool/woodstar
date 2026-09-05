package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/config"
)

func TestLogoutSurfaceLoadsSessionAndRejectsCrossOrigin(t *testing.T) {
	sessions := scs.New()
	sessions.Store = memstore.NewWithCleanupInterval(0)
	ctx, err := sessions.Load(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	sessions.Put(ctx, "sentinel", "present")
	token, _, err := sessions.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	handler := routes(ServerOptions{
		Config:         config.Config{ClientIPSource: config.ClientIPSourceRemoteAddr},
		Logger:         slog.New(slog.DiscardHandler),
		SessionManager: sessions,
		RegisterRoutes: func(routes Routes) {
			huma.Register(routes.App.Logout, huma.Operation{
				OperationID: "delete-session", Method: http.MethodDelete, Path: "/api/session",
				DefaultStatus: http.StatusNoContent,
			}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
				if sessions.GetString(ctx, "sentinel") != "present" {
					t.Error("logout session was not loaded")
				}
				return nil, sessions.Destroy(ctx)
			})
		},
	})
	for _, tt := range []struct {
		origin string
		status int
	}{
		{"https://untrusted.example.invalid", http.StatusForbidden},
		{"https://app.example.invalid", http.StatusNoContent},
	} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "https://app.example.invalid/api/session", nil)
		req.Header.Set("Cookie", sessions.Cookie.Name+"="+token)
		req.Header.Set("Origin", tt.origin)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tt.status {
			t.Fatalf("logout from %s = %d %s, want %d", tt.origin, rec.Code, rec.Body.String(), tt.status)
		}
	}
	if _, exists, err := sessions.Store.Find(token); err != nil || exists {
		t.Fatalf("session after logout: exists = %t, error = %v", exists, err)
	}
}
