package transport

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/db/sqlc"
	"github.com/woodleighschool/woodstar/internal/models"
	queryinfra "github.com/woodleighschool/woodstar/internal/queries"
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

func TestLiveQueryStreamRequiresSession(t *testing.T) {
	server := NewServer(testDependencies(testConfig()))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/live-queries/1/stream", nil)

	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLiveQueryStreamUsesBrowserSession(t *testing.T) {
	deps := testDependencies(testConfig())
	store := newTestUserStore(t)
	deps.AuthService = auth.NewService(store, deps.SessionManager)
	deps.LiveQueryManager = queryinfra.NewLiveQueryManager(queryinfra.NewHub(), time.Minute)
	server := NewServer(deps)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/live-queries/1/stream", nil)
	req.AddCookie(loginTestUser(t, deps.AuthService, deps.SessionManager))

	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	if body := rec.Body.String(); !strings.Contains(body, "event: completed") {
		t.Fatalf("body = %q, want completed event", body)
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

const testUserPassword = "test-user-password"

type testUserStore struct {
	user models.User
}

func newTestUserStore(t *testing.T) *testUserStore {
	t.Helper()

	hash, err := auth.HashPassword(testUserPassword)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return &testUserStore{
		user: models.User{User: sqlc.User{
			ID:           7,
			Email:        "admin@example.test",
			Name:         "Test Admin",
			PasswordHash: hash,
			Role:         models.RoleAdmin,
		}},
	}
}

func (s *testUserStore) Exists(context.Context) (bool, error) {
	return true, nil
}

func (s *testUserStore) Create(context.Context, models.CreateUserParams) (*models.User, error) {
	return &s.user, nil
}

func (s *testUserStore) GetByEmail(context.Context, string) (*models.User, error) {
	return &s.user, nil
}

func (s *testUserStore) GetByID(_ context.Context, id int64) (*models.User, error) {
	if id != s.user.ID {
		return nil, models.ErrNotFound
	}
	return &s.user, nil
}

func (s *testUserStore) List(context.Context) ([]models.User, error) {
	return []models.User{s.user}, nil
}

func (s *testUserStore) Update(context.Context, int64, models.UpdateUserParams) (*models.User, error) {
	return &s.user, nil
}

func (s *testUserStore) SoftDelete(context.Context, int64) error {
	return nil
}

func (s *testUserStore) CountAdmins(context.Context) (int, error) {
	return 1, nil
}

func loginTestUser(t *testing.T, authService *auth.Service, sessionManager *scs.SessionManager) *http.Cookie {
	t.Helper()

	ctx, err := sessionManager.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if _, err := authService.Login(ctx, "admin@example.test", testUserPassword); err != nil {
		t.Fatalf("login test user: %v", err)
	}
	token, _, err := sessionManager.Commit(ctx)
	if err != nil {
		t.Fatalf("commit session: %v", err)
	}
	return &http.Cookie{Name: sessionManager.Cookie.Name, Value: token}
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
		Logger:         slog.New(slog.DiscardHandler),
		AuthService:    auth.NewService(users, sessionManager),
		SessionManager: sessionManager,
		HostStore:      hosts,
		DeviceMappings: deviceMappings,
		SecretStore:    secrets,
		SoftwareStore:  software,
	}
}
