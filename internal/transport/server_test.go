package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/models"
)

func TestVersionEndpointPublic(t *testing.T) {
	server := NewServer(testDependencies(testConfig()))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/version", nil)

	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestProtectedAPIRoutesRequireSession(t *testing.T) {
	server := NewServer(testDependencies(testConfig()))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/orbit/enroll-secrets", nil)

	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAgentRoutesBypassBrowserAuth(t *testing.T) {
	server := NewServer(testDependencies(testConfig()))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/osquery/carve/begin", nil)

	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func testConfig() config.Config {
	return config.Config{
		PublicURL:     "http://localhost:8080",
		SessionSecret: strings.Repeat("s", 32),
	}
}

func testDependencies(cfg config.Config) Dependencies {
	users := models.NewUserStore(nil)
	hosts := models.NewHostStore(nil)
	deviceMappings := models.NewDeviceMappingStore(nil)
	secrets := models.NewSecretStore(nil)
	software := models.NewSoftwareStore(nil)

	sessionManager := scs.New()
	sessionManager.Store = memstore.New()

	return Dependencies{
		Config:         cfg,
		Version:        "test",
		AuthService:    auth.NewService(users, sessionManager),
		SessionManager: sessionManager,
		HostStore:      hosts,
		DeviceMappings: deviceMappings,
		SecretStore:    secrets,
		SoftwareStore:  software,
	}
}
