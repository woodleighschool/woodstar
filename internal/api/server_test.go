package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/config"
)

func TestBaseURLScopesAPI(t *testing.T) {
	server := NewServer(ServerDependencies{
		Config: config.Config{
			BaseURL:       "https://woodstar.example.edu/woodstar",
			SessionSecret: strings.Repeat("s", 32),
		},
		Version: "test",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/woodstar/api/version", nil)

	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestProtectedAPIRoutesRequireSession(t *testing.T) {
	server := NewServer(ServerDependencies{
		Config: config.Config{
			BaseURL:       "http://localhost:8080",
			SessionSecret: strings.Repeat("s", 32),
		},
		Version: "test",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/orbit/enroll-secrets", nil)

	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
