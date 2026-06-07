package adminapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/artifacts"
	"github.com/woodleighschool/woodstar/internal/munki/artifacts/storage"
	"github.com/woodleighschool/woodstar/internal/munki/hoststate"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/webui"
)

func TestProtectedAPIRoutesRequireSession(t *testing.T) {
	server := NewServer(testDependencies(testConfig()))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/agent-secrets", nil)

	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAgentSecretsAdminAPI(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := directory.NewUserService(directory.NewStore(database))
	if _, err := userService.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Agent Secret Admin",
		Password: testUserPassword,
		Role:     directory.RoleAdmin,
	}); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	deps := testDependencies(testConfig())
	deps.Runtime.DB = database
	deps.Auth.UserService = userService
	deps.Auth.AuthService = auth.NewService(userService, deps.Runtime.SessionManager)
	deps.AgentAuth.Store = agentauth.NewStore(database)
	server := NewServer(deps)
	cookie := loginTestUser(t, deps.Auth.AuthService, deps.Runtime.SessionManager)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/api/agent-secrets",
		strings.NewReader(`{"agent":"santa","value":"created-santa-secret-value-long-32"}`),
	)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Sec-Fetch-Site", "same-origin")
	createReq.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body = %q", createRec.Code, http.StatusCreated, createRec.Body.String())
	}
	var created agentauth.AgentSecret
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode created secret: %v", err)
	}
	if created.Agent != agentauth.AgentSanta || created.Value != "created-santa-secret-value-long-32" {
		t.Fatalf("created secret = %+v, want santa value", created)
	}

	ok, err := deps.AgentAuth.Store.Verify(ctx, agentauth.AgentSanta, created.Value)
	if err != nil {
		t.Fatalf("verify created secret: %v", err)
	}
	if !ok {
		t.Fatal("created santa secret did not verify")
	}

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/agent-secrets", nil)
	listReq.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body = %q", listRec.Code, http.StatusOK, listRec.Body.String())
	}
	var listed []agentauth.AgentSecret
	if err := json.NewDecoder(listRec.Body).Decode(&listed); err != nil {
		t.Fatalf("decode listed secrets: %v", err)
	}
	if !containsAgentSecret(listed, created.ID, agentauth.AgentSanta, created.Value) {
		t.Fatalf("created secret missing from list: %+v", listed)
	}

	const updatedValue = "updated-santa-secret-value-long-32"
	updateRec := httptest.NewRecorder()
	updateReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPatch,
		fmt.Sprintf("/api/agent-secrets/%d", created.ID),
		strings.NewReader(fmt.Sprintf(`{"value":%q}`, updatedValue)),
	)
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Sec-Fetch-Site", "same-origin")
	updateReq.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body = %q", updateRec.Code, http.StatusOK, updateRec.Body.String())
	}
	var updated agentauth.AgentSecret
	if err := json.NewDecoder(updateRec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated secret: %v", err)
	}
	if updated.ID != created.ID || updated.Agent != agentauth.AgentSanta || updated.Value != updatedValue {
		t.Fatalf("updated secret = %+v, want id %d santa value %q", updated, created.ID, updatedValue)
	}
	ok, err = deps.AgentAuth.Store.Verify(ctx, agentauth.AgentSanta, created.Value)
	if err != nil {
		t.Fatalf("verify old secret after update: %v", err)
	}
	if ok {
		t.Fatal("old santa secret still verifies after update")
	}
	ok, err = deps.AgentAuth.Store.Verify(ctx, agentauth.AgentSanta, updated.Value)
	if err != nil {
		t.Fatalf("verify updated secret: %v", err)
	}
	if !ok {
		t.Fatal("updated santa secret did not verify")
	}
	created = updated

	deleteRec := httptest.NewRecorder()
	deleteReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		fmt.Sprintf("/api/agent-secrets/%d", created.ID),
		nil,
	)
	deleteReq.Header.Set("Sec-Fetch-Site", "same-origin")
	deleteReq.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf(
			"delete status = %d, want %d; body = %q",
			deleteRec.Code,
			http.StatusNoContent,
			deleteRec.Body.String(),
		)
	}

	ok, err = deps.AgentAuth.Store.Verify(ctx, agentauth.AgentSanta, created.Value)
	if err != nil {
		t.Fatalf("verify deleted secret: %v", err)
	}
	if ok {
		t.Fatal("deleted santa secret still verifies")
	}
}

func TestAgentSecretsRejectBadAgent(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := directory.NewUserService(directory.NewStore(database))
	if _, err := userService.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Agent Secret Admin",
		Password: testUserPassword,
		Role:     directory.RoleAdmin,
	}); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	deps := testDependencies(testConfig())
	deps.Runtime.DB = database
	deps.Auth.UserService = userService
	deps.Auth.AuthService = auth.NewService(userService, deps.Runtime.SessionManager)
	deps.AgentAuth.Store = agentauth.NewStore(database)
	server := NewServer(deps)
	cookie := loginTestUser(t, deps.Auth.AuthService, deps.Runtime.SessionManager)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/api/agent-secrets",
		strings.NewReader(`{"agent":"mdm","value":"invalid-agent-secret-value-long-32"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestAgentSecretsRequireAdmin(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := directory.NewUserService(directory.NewStore(database))
	if _, err := userService.Create(ctx, directory.UserCreate{
		Email:    "viewer@example.test",
		Name:     "Agent Secret Viewer",
		Password: testUserPassword,
		Role:     directory.RoleViewer,
	}); err != nil {
		t.Fatalf("create viewer user: %v", err)
	}

	deps := testDependencies(testConfig())
	deps.Runtime.DB = database
	deps.Auth.UserService = userService
	deps.Auth.AuthService = auth.NewService(userService, deps.Runtime.SessionManager)
	deps.AgentAuth.Store = agentauth.NewStore(database)
	server := NewServer(deps)
	cookie := loginTestUserWithEmail(t, deps.Auth.AuthService, deps.Runtime.SessionManager, "viewer@example.test")

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/agent-secrets", nil)
	req.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestMunkiAdminAPI(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := directory.NewUserService(directory.NewStore(database))
	if _, err := userService.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Munki Admin",
		Password: testUserPassword,
		Role:     directory.RoleAdmin,
	}); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	deps := testDependencies(testConfig())
	deps.Runtime.DB = database
	deps.Auth.UserService = userService
	deps.Auth.AuthService = auth.NewService(userService, deps.Runtime.SessionManager)
	wireMunkiTestDeps(&deps, database)
	server := NewServer(deps)
	cookie := loginTestUser(t, deps.Auth.AuthService, deps.Runtime.SessionManager)

	title := postMunkiJSON[munkisoftware.SoftwareTitle](
		t,
		server,
		cookie,
		"/api/munki/software",
		`{"name":"Google Chrome","targets":{"include":[],"exclude":[]}}`,
	)
	pkg := postMunkiJSON[packages.Package](
		t,
		server,
		cookie,
		"/api/munki/packages",
		fmt.Sprintf(
			`{"software_id":%d,"version":"148.0.0.1","installer_type":"nopkg","on_demand":true,"eligible":true}`,
			title.ID,
		),
	)
	if pkg.SoftwareName != "Google Chrome" || pkg.Version != "148.0.0.1" {
		t.Fatalf("pkg = %+v, want Google Chrome 148.0.0.1", pkg)
	}
	pkg = patchMunkiJSON[packages.Package](
		t,
		server,
		cookie,
		fmt.Sprintf("/api/munki/packages/%d", pkg.ID),
		fmt.Sprintf(
			`{"software_id":%d,"version":"148.0.0.2","installer_type":"nopkg","on_demand":true,"eligible":true}`,
			title.ID,
		),
	)
	if pkg.Version != "148.0.0.2" {
		t.Fatalf("updated pkg = %+v, want version 148.0.0.2", pkg)
	}
	excludeLabel, err := labels.NewStore(database).Create(context.Background(), labels.LabelMutation{
		Name:                "Munki API Exclude",
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create exclude label: %v", err)
	}
	allHostsID := apiTestAllHostsLabelID(t, context.Background(), labels.NewStore(database))
	updatedTitle := putMunkiJSON[struct {
		Targets munkisoftware.SoftwareTargets `json:"targets"`
	}](
		t,
		server,
		cookie,
		fmt.Sprintf("/api/munki/software/%d", title.ID),
		fmt.Sprintf(
			`{"name":"Google Chrome","description":"","category":"","developer":"","targets":{"include":[{"label_id":%d,"package":{"strategy":"specific","package_id":%d},"state":"optional_install","featured":true}],"exclude":[{"label_id":%d}]}}`,
			allHostsID,
			pkg.ID,
			excludeLabel.ID,
		),
	)
	if len(updatedTitle.Targets.Include) != 1 {
		t.Fatalf("includes = %+v, want one target", updatedTitle.Targets.Include)
	}
	include := updatedTitle.Targets.Include[0]
	if include.State != munkisoftware.SoftwareStateOptionalInstall || !include.Featured {
		t.Fatalf("include = %+v, want optional_install and featured", include)
	}
	if len(updatedTitle.Targets.Exclude) != 1 || updatedTitle.Targets.Exclude[0].LabelID != excludeLabel.ID {
		t.Fatalf("exclude labels = %+v, want [%d]", updatedTitle.Targets.Exclude, excludeLabel.ID)
	}

	detailRec := httptest.NewRecorder()
	detailReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("/api/munki/software/%d", title.ID),
		nil,
	)
	detailReq.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(detailRec, detailReq)
	if detailRec.Code != http.StatusOK {
		t.Fatalf(
			"software detail status = %d, want %d; body = %q",
			detailRec.Code,
			http.StatusOK,
			detailRec.Body.String(),
		)
	}
}

func TestMunkiArtifactUploadEndpointReturnsFinalizePayload(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := directory.NewUserService(directory.NewStore(database))
	if _, err := userService.Create(ctx, directory.UserCreate{
		Email:    "munki-upload@example.test",
		Name:     "Munki Upload",
		Password: testUserPassword,
		Role:     directory.RoleAdmin,
	}); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	deps := testDependencies(testConfig())
	deps.Runtime.DB = database
	deps.Auth.UserService = userService
	deps.Auth.AuthService = auth.NewService(userService, deps.Runtime.SessionManager)
	wireMunkiTestDeps(&deps, database)
	deps.Munki.ArtifactStorage = fakeMunkiStorage{}
	server := NewServer(deps)
	cookie := loginTestUserWithEmail(t, deps.Auth.AuthService, deps.Runtime.SessionManager, "munki-upload@example.test")
	sha := strings.Repeat("a", 64)

	upload := postMunkiJSON[struct {
		UploadURL string                     `json:"upload_url"`
		Headers   map[string]string          `json:"headers"`
		Artifact  artifacts.ArtifactMutation `json:"artifact"`
	}](
		t,
		server,
		cookie,
		"/api/munki/artifact-uploads",
		fmt.Sprintf(
			`{"kind":"icon","filename":"GoogleChrome.png","content_type":"image/png","size_bytes":123,"sha256":%q}`,
			sha,
		),
	)
	if upload.UploadURL == "" || upload.Artifact.StorageKey == "" {
		t.Fatalf("upload = %+v, want presigned URL and finalize payload", upload)
	}
	if upload.Artifact.Location != "aaaaaaaaaaaa/GoogleChrome.png" {
		t.Fatalf("artifact location = %q", upload.Artifact.Location)
	}

	body, err := json.Marshal(upload.Artifact)
	if err != nil {
		t.Fatalf("marshal artifact: %v", err)
	}
	artifact := postMunkiJSON[artifacts.Artifact](t, server, cookie, "/api/munki/artifacts", string(body))
	if artifact.Kind != artifacts.ArtifactKindIcon || artifact.SHA256 != sha {
		t.Fatalf("artifact = %+v, want finalized icon artifact", artifact)
	}
	again := postMunkiJSON[artifacts.Artifact](t, server, cookie, "/api/munki/artifacts", string(body))
	if again.ID != artifact.ID {
		t.Fatalf("repeat artifact finalize id = %d, want %d", again.ID, artifact.ID)
	}

	contentRec := httptest.NewRecorder()
	contentReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("/api/munki/artifacts/%d/content", artifact.ID),
		nil,
	)
	contentReq.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(contentRec, contentReq)
	if contentRec.Code != http.StatusFound {
		t.Fatalf("content status = %d, want %d; body = %q", contentRec.Code, http.StatusFound, contentRec.Body.String())
	}
	if location := contentRec.Header().Get("Location"); location != "https://storage.example/"+artifact.StorageKey {
		t.Fatalf("content location = %q, want fake storage URL", location)
	}
}

func TestMunkiArtifactUploadEndpointReportsUnavailableStorage(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := directory.NewUserService(directory.NewStore(database))
	if _, err := userService.Create(ctx, directory.UserCreate{
		Email:    "munki-disabled-storage@example.test",
		Name:     "Munki Disabled Storage",
		Password: testUserPassword,
		Role:     directory.RoleAdmin,
	}); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	deps := testDependencies(testConfig())
	deps.Runtime.DB = database
	deps.Auth.UserService = userService
	deps.Auth.AuthService = auth.NewService(userService, deps.Runtime.SessionManager)
	wireMunkiTestDeps(&deps, database)
	deps.Munki.ArtifactStorage = unavailableMunkiStorage{}
	server := NewServer(deps)
	cookie := loginTestUserWithEmail(
		t,
		deps.Auth.AuthService,
		deps.Runtime.SessionManager,
		"munki-disabled-storage@example.test",
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/api/munki/artifact-uploads",
		strings.NewReader(
			fmt.Sprintf(
				`{"kind":"icon","filename":"GoogleChrome.png","content_type":"image/png","size_bytes":123,"sha256":%q}`,
				strings.Repeat("a", 64),
			),
		),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Munki artifact storage is not configured")) {
		t.Fatalf("body = %q, want storage unavailable message", rec.Body.String())
	}
}

func containsAgentSecret(secrets []agentauth.AgentSecret, id int64, agent agentauth.Agent, value string) bool {
	for _, secret := range secrets {
		if secret.ID == id && secret.Agent == agent && secret.Value == value {
			return true
		}
	}
	return false
}

type fakeMunkiStorage struct{}

func (fakeMunkiStorage) PresignGet(_ context.Context, artifact artifacts.Artifact) (string, error) {
	return "https://storage.example/" + artifact.StorageKey, nil
}

func (fakeMunkiStorage) PresignPut(
	_ context.Context,
	storageKey string,
	contentType string,
	sha256 string,
) (artifacts.ArtifactUploadURL, error) {
	return artifacts.ArtifactUploadURL{
		URL:     "https://storage.example/" + storageKey,
		Headers: map[string]string{"Content-Type": contentType, "x-amz-meta-woodstar-sha256": sha256},
	}, nil
}

func (fakeMunkiStorage) Stat(_ context.Context, _ string) (artifacts.ArtifactObject, error) {
	return artifacts.ArtifactObject{ContentType: "image/png", SizeBytes: 123, SHA256: strings.Repeat("a", 64)}, nil
}

type unavailableMunkiStorage struct{}

func (unavailableMunkiStorage) PresignGet(context.Context, artifacts.Artifact) (string, error) {
	return "", storage.ErrUnavailable
}

func (unavailableMunkiStorage) PresignPut(
	context.Context,
	string,
	string,
	string,
) (artifacts.ArtifactUploadURL, error) {
	return artifacts.ArtifactUploadURL{}, storage.ErrUnavailable
}

func (unavailableMunkiStorage) Stat(context.Context, string) (artifacts.ArtifactObject, error) {
	return artifacts.ArtifactObject{}, storage.ErrUnavailable
}

type munkiTestStores struct {
	artifacts      *artifacts.Store
	hoststate      *hoststate.Store
	packages       *packages.Store
	softwareTitles *munkisoftware.Store
}

func wireMunkiTestDeps(deps *Dependencies, db *database.DB) munkiTestStores {
	artifactStore := artifacts.NewStore(db)
	packageStore := packages.NewStore(db, artifactStore)
	softwareTitleStore := munkisoftware.NewStore(db, artifactStore, packageStore)
	stores := munkiTestStores{
		artifacts:      artifactStore,
		hoststate:      hoststate.NewStore(db),
		packages:       packageStore,
		softwareTitles: softwareTitleStore,
	}
	deps.Munki.Artifacts = stores.artifacts
	deps.Munki.HostState = stores.hoststate
	deps.Munki.Packages = stores.packages
	deps.Munki.SoftwareTitles = stores.softwareTitles
	return stores
}

func postMunkiJSON[T any](t *testing.T, server *Server, cookie *http.Cookie, path string, body string) T {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST %s status = %d, want %d; body = %q", path, rec.Code, http.StatusCreated, rec.Body.String())
	}
	var decoded T
	if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode POST %s: %v", path, err)
	}
	return decoded
}

func putMunkiJSON[T any](t *testing.T, server *Server, cookie *http.Cookie, path string, body string) T {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT %s status = %d, want %d; body = %q", path, rec.Code, http.StatusOK, rec.Body.String())
	}
	var decoded T
	if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode PUT %s: %v", path, err)
	}
	return decoded
}

func patchMunkiJSON[T any](t *testing.T, server *Server, cookie *http.Cookie, path string, body string) T {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(cookie)
	server.httpServer.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH %s status = %d, want %d; body = %q", path, rec.Code, http.StatusOK, rec.Body.String())
	}
	var decoded T
	if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode PATCH %s: %v", path, err)
	}
	return decoded
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
	userService := directory.NewUserService(directory.NewStore(database))
	if _, err := userService.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Test Admin",
		Password: testUserPassword,
		Role:     directory.RoleAdmin,
	}); err != nil {
		t.Fatalf("create test user: %v", err)
	}

	manager := livequery.NewManager()
	streamHandle := manager.Start("select 1", nil)
	stopHandle := manager.Start("select 1", []int64{4})

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
		fmt.Sprintf("/api/live-queries/%d/stream", streamHandle.ID),
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
		fmt.Sprintf("/api/live-queries/%d/stop", stopHandle.ID),
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
	userService := directory.NewUserService(directory.NewStore(database))
	if _, err := userService.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Test Admin",
		Password: testUserPassword,
		Role:     directory.RoleAdmin,
	}); err != nil {
		t.Fatalf("create test user: %v", err)
	}

	deps := testDependencies(testConfig())
	deps.Runtime.DB = database
	deps.Auth.UserService = userService
	deps.Auth.AuthService = auth.NewService(userService, deps.Runtime.SessionManager)
	deps.Inventory.Hosts = hosts.NewStore(database)
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

func TestMunkiProtocolRoutesUseMunkiBearerAuth(t *testing.T) {
	database, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(database)
	if _, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware: hosts.HostHardware{
			UUID:   "munki-protocol-host-uuid",
			Serial: "C02MUNKI",
		},
		Hostname: "test-macbook",
	}); err != nil {
		t.Fatalf("create munki protocol host: %v", err)
	}

	deps := testDependencies(testConfig())
	deps.AgentAuth.Store = agentauth.NewStore(database)
	stores := wireMunkiTestDeps(&deps, database)
	deps.Munki.Repository = munki.NewService(hostStore, stores.softwareTitles)
	server := NewServer(deps)

	secret, err := deps.AgentAuth.Store.Create(ctx, agentauth.AgentSecretCreate{
		Agent: agentauth.AgentMunki,
		Value: "munki-protocol-secret-value-long-32",
	})
	if err != nil {
		t.Fatalf("create munki secret: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/munki/manifests/C02MUNKI", nil)
	req.Header.Set("Authorization", "Bearer "+secret.Value)
	req.Header.Set("Serial", "C02MUNKI")
	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/x-plist" {
		t.Fatalf("Content-Type = %q, want application/x-plist", got)
	}

	wrongAgent, err := deps.AgentAuth.Store.Create(ctx, agentauth.AgentSecretCreate{
		Agent: agentauth.AgentSanta,
		Value: "santa-protocol-secret-value-long-32",
	})
	if err != nil {
		t.Fatalf("create santa secret: %v", err)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/munki/manifests/C02MUNKI", nil)
	req.Header.Set("Authorization", "Bearer "+wrongAgent.Value)
	req.Header.Set("Serial", "C02MUNKI")
	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong-agent status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestBearerMutationAllowsNonBrowserClient(t *testing.T) {
	database, ctx := dbtest.Open(t)
	userService := directory.NewUserService(directory.NewStore(database))
	user, err := userService.Create(ctx, directory.UserCreate{
		Email:    "api@example.test",
		Name:     "API User",
		Password: testUserPassword,
		Role:     directory.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	const apiKey = "fleet-style-retrievable-key"
	if _, err := userService.SetAccountAPIKey(ctx, user.ID, apiKey); err != nil {
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
	userService := directory.NewUserService(directory.NewStore(database))
	user, err := userService.Create(ctx, directory.UserCreate{
		Email:    "admin@example.test",
		Name:     "Account User",
		Password: testUserPassword,
		Role:     directory.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	const apiKey = "fleet-style-visible-key"
	if _, err := userService.SetAccountAPIKey(ctx, user.ID, apiKey); err != nil {
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

func TestBrowserRoutesCompressesSPA(t *testing.T) {
	deps := testDependencies(testConfig())
	deps.Runtime.WebHandler = webui.NewHandler(webui.HandlerOptions{
		FS: fstest.MapFS{
			"index.html": {
				Data: []byte(
					"<!doctype html><html><head></head><body>" +
						strings.Repeat("content ", 400) +
						"</body></html>",
				),
			},
		},
		Version:   "test",
		PublicURL: deps.Runtime.Config.PublicURL,
		Logger:    slog.New(slog.DiscardHandler),
	})
	server := NewServer(deps)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/santa/events", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	server.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}

	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("read gzip response: %v", err)
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read compressed content: %v", err)
	}
	if !strings.Contains(
		string(content),
		"window.__WOODSTAR__={\"version\":\"test\",\"public_url\":\"http://localhost:8080\"};",
	) {
		t.Fatalf("decompressed body did not include runtime config: %q", content)
	}
}

const testUserPassword = "test-user-password"

func loginTestUser(t *testing.T, authService *auth.Service, sessionManager *scs.SessionManager) *http.Cookie {
	t.Helper()
	return loginTestUserWithEmail(t, authService, sessionManager, "admin@example.test")
}

func loginTestUserWithEmail(
	t *testing.T,
	authService *auth.Service,
	sessionManager *scs.SessionManager,
	email string,
) *http.Cookie {
	t.Helper()
	ctx, err := sessionManager.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if _, err := authService.Login(ctx, email, testUserPassword); err != nil {
		t.Fatalf("login test user: %v", err)
	}
	token, _, err := sessionManager.Commit(ctx)
	if err != nil {
		t.Fatalf("commit session: %v", err)
	}
	return &http.Cookie{Name: sessionManager.Cookie.Name, Value: token}
}

func apiTestAllHostsLabelID(t *testing.T, ctx context.Context, store *labels.Store) int64 {
	t.Helper()
	rows, _, err := store.List(ctx, labels.ListParams{})
	if err != nil {
		t.Fatalf("list labels: %v", err)
	}
	for _, row := range rows {
		if row.BuiltinKey != nil && *row.BuiltinKey == labels.BuiltinKeyAllHosts {
			return row.ID
		}
	}
	t.Fatalf("All Hosts label not found")
	return 0
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
	userService := directory.NewUserService(&directory.Store{})

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
