package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/users"
)

func TestProtectedAPIRoutesRequireSession(t *testing.T) {
	server := NewServer(testDependencies(testConfig()))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/enroll-secrets", nil)

	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLiveQueryStreamRequiresSession(t *testing.T) {
	server := NewServer(testDependencies(testConfig()))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/live-queries/1/stream", nil)

	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLiveQueryEndpointsUseBrowserSession(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := users.NewService(users.NewStore(database))
	if _, err := userService.Create(ctx, users.CreateParams{
		Email:    "admin@example.test",
		Name:     "Test Admin",
		Password: testUserPassword,
		Role:     users.RoleAdmin,
	}); err != nil {
		t.Fatalf("create test user: %v", err)
	}

	manager := livequery.NewManager()
	handle := manager.Start("select 1", []int64{4})

	deps := testDependencies(testConfig())
	deps.Runtime.DB = database
	deps.Auth.UserService = userService
	deps.Auth.AuthService = auth.NewService(userService, deps.Runtime.SessionManager)
	deps.Osquery.LiveQueries = manager
	server := NewServer(deps)
	cookie := loginTestUser(t, deps.Auth.AuthService, deps.Runtime.SessionManager)

	streamRec := httptest.NewRecorder()
	streamReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("/api/live-queries/%d/stream", handle.ID),
		nil,
	)
	streamReq.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(streamRec, streamReq)

	if streamRec.Code != http.StatusOK {
		t.Fatalf("stream status = %d, want %d", streamRec.Code, http.StatusOK)
	}
	if got := streamRec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}

	stopRec := httptest.NewRecorder()
	stopReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		fmt.Sprintf("/api/live-queries/%d/stop", handle.ID),
		nil,
	)
	stopReq.Header.Set("Sec-Fetch-Site", "same-origin")
	stopReq.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(stopRec, stopReq)

	if stopRec.Code != http.StatusNoContent {
		t.Fatalf("stop status = %d, want %d; body = %q", stopRec.Code, http.StatusNoContent, stopRec.Body.String())
	}
	if work := manager.PendingForHost(4); len(work) != 0 {
		t.Fatalf("work after stop = %#v, want none", work)
	}
}

func TestBrowserMutationRequiresTrustedOrigin(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := users.NewService(users.NewStore(database))
	if _, err := userService.Create(ctx, users.CreateParams{
		Email:    "admin@example.test",
		Name:     "Test Admin",
		Password: testUserPassword,
		Role:     users.RoleAdmin,
	}); err != nil {
		t.Fatalf("create test user: %v", err)
	}

	deps := testDependencies(testConfig())
	deps.Runtime.DB = database
	deps.Auth.UserService = userService
	deps.Auth.AuthService = auth.NewService(userService, deps.Runtime.SessionManager)
	deps.Hosts.Store = hosts.NewStore(database)
	server := NewServer(deps)
	sessionCookie := loginTestUser(t, deps.Auth.AuthService, deps.Runtime.SessionManager)

	cases := []struct {
		name           string
		origin         string
		secFetchSite   string
		wantStatusCode int
	}{
		{name: "cross-origin rejected", origin: "http://evil.example", wantStatusCode: http.StatusForbidden},
		{
			name:           "fetch-metadata same-origin accepted",
			origin:         "http://localhost:5173",
			secFetchSite:   "same-origin",
			wantStatusCode: http.StatusOK,
		},
		{name: "matching origin accepted", origin: "http://localhost:8080", wantStatusCode: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(
				context.Background(),
				http.MethodPost,
				"http://localhost:8080/api/live-queries/targets/count",
				strings.NewReader(`{"selected":{"hosts":[],"labels":[]}}`),
			)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", tc.origin)
			if tc.secFetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tc.secFetchSite)
			}
			req.AddCookie(sessionCookie)

			server.httpServer.Handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatusCode {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, tc.wantStatusCode, rec.Body.String())
			}
			if tc.wantStatusCode == http.StatusForbidden && rec.Body.String() != "forbidden origin" {
				t.Fatalf("body = %q, want %q", rec.Body.String(), "forbidden origin")
			}
		})
	}
}

func TestOrbitProtocolRoutesBypassBrowserAuth(t *testing.T) {
	server := NewServer(testDependencies(testConfig()))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/osquery/carve/begin", nil)

	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestBearerMutationAllowsNonBrowserClient(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := users.NewService(users.NewStore(database))
	user, err := userService.Create(ctx, users.CreateParams{
		Email:    "api@example.test",
		Name:     "API User",
		Password: testUserPassword,
		Role:     users.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	const apiKey = "fleet-style-retrievable-key"
	if _, err := userService.SetAPIKey(ctx, user.ID, apiKey); err != nil {
		t.Fatalf("set api key: %v", err)
	}

	deps := testDependencies(testConfig())
	deps.Runtime.DB = database
	deps.Auth.UserService = userService
	deps.Auth.AuthService = auth.NewService(userService, deps.Runtime.SessionManager)
	server := NewServer(deps)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/account/api-key", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestAccountReadReturnsRetrievableAPIKeyOnlyToSelf(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := users.NewService(users.NewStore(database))
	user, err := userService.Create(ctx, users.CreateParams{
		Email:    "admin@example.test",
		Name:     "Account User",
		Password: testUserPassword,
		Role:     users.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	const apiKey = "fleet-style-visible-key"
	if _, err := userService.SetAPIKey(ctx, user.ID, apiKey); err != nil {
		t.Fatalf("set api key: %v", err)
	}

	deps := testDependencies(testConfig())
	deps.Runtime.DB = database
	deps.Auth.UserService = userService
	deps.Auth.AuthService = auth.NewService(userService, deps.Runtime.SessionManager)
	server := NewServer(deps)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/account", nil)
	req.AddCookie(loginTestUser(t, deps.Auth.AuthService, deps.Runtime.SessionManager))

	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body struct {
		APIKey string         `json:"api_key"`
		User   map[string]any `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if body.APIKey != apiKey {
		t.Fatalf("api_key = %q, want %q", body.APIKey, apiKey)
	}
	if _, ok := body.User["api_key"]; ok {
		t.Fatalf("account user leaked api_key: %#v", body.User)
	}
}

const testUserPassword = "test-user-password"

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
	sessionManager := scs.New()
	sessionManager.Store = memstore.New()
	userService := users.NewService(&users.Store{})

	return Dependencies{
		Runtime: RuntimeDependencies{
			Config:         cfg,
			Version:        "test",
			Logger:         slog.New(slog.DiscardHandler),
			SessionManager: sessionManager,
		},
		Auth: AuthDependencies{
			AuthService: auth.NewService(userService, sessionManager),
			UserService: userService,
		},
	}
}
