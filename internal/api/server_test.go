package api

import (
	"bytes"
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

	"github.com/woodleighschool/woodstar/internal/agents/livequery"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/users"
)

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
	database, ctx := dbtest.Open(t)
	userStore := users.NewStore(database)
	userService := users.NewService(userStore)
	if _, err := userService.Create(ctx, users.CreateParams{
		Email:    "admin@example.test",
		Name:     "Test Admin",
		Password: testUserPassword,
		Role:     users.RoleAdmin,
	}); err != nil {
		t.Fatalf("create test user: %v", err)
	}

	deps := testDependencies(testConfig())
	deps.UserService = userService
	deps.AuthService = auth.NewService(userService, deps.SessionManager)
	deps.LiveQueryManager = livequery.NewManager()
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

func TestLiveQueryStopUsesBrowserSession(t *testing.T) {
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
	deps.DB = database
	deps.UserService = userService
	deps.AuthService = auth.NewService(userService, deps.SessionManager)
	deps.LiveQueryManager = manager
	server := NewServer(deps)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		fmt.Sprintf("/api/live-queries/%d/stop", handle.ID),
		nil,
	)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(loginTestUser(t, deps.AuthService, deps.SessionManager))

	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusNoContent, rec.Body.String())
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
	deps.DB = database
	deps.UserService = userService
	deps.AuthService = auth.NewService(userService, deps.SessionManager)
	deps.HostStore = hosts.NewStore(database)
	server := NewServer(deps)

	sessionCookie := loginTestUser(t, deps.AuthService, deps.SessionManager)

	postRec := httptest.NewRecorder()
	postReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/api/live-queries/targets/count",
		strings.NewReader(`{"selected":{"hosts":[],"labels":[]}}`),
	)
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Origin", "http://evil.example")
	postReq.AddCookie(sessionCookie)

	server.routes().ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusForbidden {
		t.Fatalf(
			"cross-origin status = %d, want %d; body = %q",
			postRec.Code,
			http.StatusForbidden,
			postRec.Body.String(),
		)
	}
	if postRec.Body.String() != "forbidden origin" {
		t.Fatalf("cross-origin body = %q, want forbidden origin", postRec.Body.String())
	}

	postRec = httptest.NewRecorder()
	postReq = httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"http://localhost:8080/api/live-queries/targets/count",
		strings.NewReader(`{"selected":{"hosts":[],"labels":[]}}`),
	)
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Origin", "http://localhost:5173")
	postReq.Header.Set("Sec-Fetch-Site", "same-origin")
	postReq.AddCookie(sessionCookie)

	server.routes().ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf(
			"same-origin fetch metadata status = %d, want %d; body = %q",
			postRec.Code,
			http.StatusOK,
			postRec.Body.String(),
		)
	}

	postRec = httptest.NewRecorder()
	postReq = httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"http://localhost:8080/api/live-queries/targets/count",
		strings.NewReader(`{"selected":{"hosts":[],"labels":[]}}`),
	)
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Origin", "http://localhost:8080")
	postReq.AddCookie(sessionCookie)

	server.routes().ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("same-origin status = %d, want %d; body = %q", postRec.Code, http.StatusOK, postRec.Body.String())
	}
}

func TestLiveQueryTargetCountReturnsStatusMetrics(t *testing.T) {
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
	const apiKey = "fleet-style-target-count-key"
	if _, err := userService.SetAPIKey(ctx, user.ID, apiKey); err != nil {
		t.Fatalf("set api key: %v", err)
	}

	hostStore := hosts.NewStore(database)
	labelStore := labels.NewStore(database)
	onlineHost, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.EnrollParams{
		HardwareUUID: "test-api-target-count-online",
		OrbitNodeKey: "orbit-key-api-target-count-online",
	})
	if err != nil {
		t.Fatalf("enroll online host: %v", err)
	}
	offlineHost, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.EnrollParams{
		HardwareUUID: "test-api-target-count-offline",
		OrbitNodeKey: "orbit-key-api-target-count-offline",
	})
	if err != nil {
		t.Fatalf("enroll offline host: %v", err)
	}
	if _, err := database.Pool().Exec(ctx,
		`UPDATE hosts
		 SET last_seen_at = CASE id
		     WHEN $1 THEN now() - interval '1 minute'
		     WHEN $2 THEN now() - interval '10 minutes'
		 END
		 WHERE id = ANY($3::bigint[])`,
		onlineHost.ID,
		offlineHost.ID,
		[]int64{onlineHost.ID, offlineHost.ID},
	); err != nil {
		t.Fatalf("set host seen times: %v", err)
	}
	label, err := labelStore.Create(ctx, labels.LabelCreate{
		Name:                "API Target Count Test",
		LabelType:           labels.LabelTypeRegular,
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	if err := labelStore.SetMembership(ctx, label.ID, offlineHost.ID, true); err != nil {
		t.Fatalf("set label membership: %v", err)
	}

	deps := testDependencies(testConfig())
	deps.DB = database
	deps.UserService = userService
	deps.AuthService = auth.NewService(userService, deps.SessionManager)
	deps.HostStore = hostStore
	server := NewServer(deps)

	body, err := json.Marshal(struct {
		Selected struct {
			Hosts  []int64 `json:"hosts"`
			Labels []int64 `json:"labels"`
		} `json:"selected"`
	}{
		Selected: struct {
			Hosts  []int64 `json:"hosts"`
			Labels []int64 `json:"labels"`
		}{
			Hosts:  []int64{onlineHost.ID, onlineHost.ID},
			Labels: []int64{label.ID},
		},
	})
	if err != nil {
		t.Fatalf("encode request body: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/api/live-queries/targets/count",
		bytes.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got struct {
		TargetsCount           int `json:"targets_count"`
		TargetsOnline          int `json:"targets_online"`
		TargetsOffline         int `json:"targets_offline"`
		TargetsMissingInAction int `json:"targets_missing_in_action"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if got.TargetsCount != 2 || got.TargetsOnline != 1 || got.TargetsOffline != 1 || got.TargetsMissingInAction != 0 {
		t.Fatalf("target counts = %+v, want 2 total, 1 online, 1 offline, 0 missing", got)
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
	deps.DB = database
	deps.UserService = userService
	deps.AuthService = auth.NewService(userService, deps.SessionManager)
	server := NewServer(deps)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/account/api-key", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	server.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestAccountReadOwnsRetrievableAPIKey(t *testing.T) {
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
	deps.DB = database
	deps.UserService = userService
	deps.AuthService = auth.NewService(userService, deps.SessionManager)
	server := NewServer(deps)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/account", nil)
	req.AddCookie(loginTestUser(t, deps.AuthService, deps.SessionManager))

	server.routes().ServeHTTP(rec, req)

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

func TestNewServerBuildsHTTPServer(t *testing.T) {
	server := NewServer(testDependencies(config.Config{
		Host:          "127.0.0.1",
		Port:          9090,
		PublicURL:     "http://localhost:9090",
		SessionSecret: strings.Repeat("s", 32),
	}))

	if server.httpServer.Addr != "127.0.0.1:9090" {
		t.Fatalf("Addr = %q, want 127.0.0.1:9090", server.httpServer.Addr)
	}
	if server.httpServer.Handler == nil {
		t.Fatal("Handler is nil")
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
		Config:         cfg,
		Version:        "test",
		Logger:         slog.New(slog.DiscardHandler),
		AuthService:    auth.NewService(userService, sessionManager),
		UserService:    userService,
		SessionManager: sessionManager,
	}
}
