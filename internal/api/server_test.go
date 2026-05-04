package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/models"
	"github.com/woodleighschool/woodstar/internal/orbit"
	"github.com/woodleighschool/woodstar/internal/osquery"
)

func TestBaseURLScopesAPI(t *testing.T) {
	server := NewServer(testDependencies(config.Config{
		BaseURL:       "https://woodstar.example.edu/woodstar",
		SessionSecret: strings.Repeat("s", 32),
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/woodstar/api/version", nil)

	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func testDependencies(cfg config.Config) ServerDependencies {
	users := models.NewUserStore(nil)
	sessions := models.NewSessionStore(nil)
	hosts := models.NewHostStore(nil)
	deviceMappings := models.NewDeviceMappingStore(nil)
	secrets := models.NewSecretStore(nil)
	software := models.NewSoftwareStore(nil)

	return ServerDependencies{
		Config: config.Config{
			BaseURL:       cfg.BaseURL,
			SessionSecret: cfg.SessionSecret,
		},
		Version:        "test",
		AuthService:    auth.NewService(users, sessions, SessionTTL, cfg.SessionSecret),
		HostStore:      hosts,
		DeviceMappings: deviceMappings,
		SecretStore:    secrets,
		SoftwareStore:  software,
		OrbitService:   orbit.NewService(hosts, secrets, deviceMappings),
		OsqueryService: osquery.NewService(hosts, software, secrets),
	}
}

func TestProtectedAPIRoutesRequireSession(t *testing.T) {
	server := NewServer(testDependencies(config.Config{
		BaseURL:       "http://localhost:8080",
		SessionSecret: strings.Repeat("s", 32),
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/orbit/enroll-secrets", nil)

	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// Agent endpoints must not gate on the admin session cookie. Without a real
// database the handler cannot complete enrollment, but the route should at
// least not return 401 — that would indicate it was accidentally placed inside
// the admin auth middleware.
func TestOrbitEnrollBypassesSessionAuth(t *testing.T) {
	server := NewServer(testDependencies(config.Config{
		BaseURL:       "http://localhost:8080",
		SessionSecret: strings.Repeat("s", 32),
	}))

	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"enroll_secret":"x","hardware_uuid":"u"}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/fleet/orbit/enroll", body)
	req.Header.Set("Content-Type", "application/json")

	server.routes().ServeHTTP(rec, req)

	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("orbit enroll returned 401; route is incorrectly behind session auth")
	}
}
