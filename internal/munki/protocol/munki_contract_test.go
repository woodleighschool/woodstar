package protocol

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"howett.net/plist"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/munki"
)

func TestMunkiHTTPFetchesManifestAndCatalog(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepository("C02MUNKI"),
	)

	cases := []struct {
		path   string
		assert func(*testing.T, []byte)
	}{
		{path: "/munki/manifests/C02MUNKI", assert: assertManifestPlist},
		{path: "/munki/catalogs/production", assert: assertCatalogPlist},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Authorization", "Bearer munki-secret")
			req.Header.Set("Serial", "C02MUNKI")

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); got != plistContentType {
				t.Fatalf("Content-Type = %q, want %q", got, plistContentType)
			}
			tc.assert(t, rec.Body.Bytes())
		})
	}
}

func assertManifestPlist(t *testing.T, body []byte) {
	t.Helper()
	var decoded struct {
		Catalogs    []string `plist:"catalogs"`
		DisplayName string   `plist:"display_name"`
	}
	if _, err := plist.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("response is not a manifest plist: %v", err)
	}
	if got := decoded.Catalogs; len(got) != 1 || got[0] != "production" {
		t.Fatalf("catalogs = %v, want [production]", got)
	}
	if decoded.DisplayName != "Test MacBook" {
		t.Fatalf("display_name = %q, want Test MacBook", decoded.DisplayName)
	}
}

func assertCatalogPlist(t *testing.T, body []byte) {
	t.Helper()
	var decoded []any
	if _, err := plist.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("response is not a catalog plist: %v", err)
	}
	if len(decoded) != 0 {
		t.Fatalf("catalog items = %d, want 0", len(decoded))
	}
}

func TestMunkiHTTPRequiresMunkiBearerSecret(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepository("C02MUNKI"),
	)

	cases := []struct {
		name          string
		authorization string
		wantStatus    int
	}{
		{name: "missing", wantStatus: http.StatusUnauthorized},
		{name: "wrong scheme", authorization: "Basic munki-secret", wantStatus: http.StatusUnauthorized},
		{name: "wrong token", authorization: "Bearer wrong-secret", wantStatus: http.StatusUnauthorized},
		{name: "valid", authorization: "Bearer munki-secret", wantStatus: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/munki/manifests/C02MUNKI", nil)
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}
			req.Header.Set("Serial", "C02MUNKI")

			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestMunkiHTTPRequiresExistingSerial(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepository("C02MUNKI"),
	)

	cases := []struct {
		name   string
		serial string
	}{
		{name: "missing"},
		{name: "unknown", serial: "C02UNKNOWN"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/munki/manifests/C02MUNKI", nil)
			req.Header.Set("Authorization", "Bearer munki-secret")
			if tc.serial != "" {
				req.Header.Set("Serial", tc.serial)
			}

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusNotFound, rec.Body.String())
			}
		})
	}
}

func TestMunkiHTTPDoesNotUseManifestNameAsIdentity(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepository("C02MUNKI"),
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/manifests/C02OTHER", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set("Serial", "C02MUNKI")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestMunkiHTTPVerifiesMunkiAgent(t *testing.T) {
	verifier := &recordingVerifier{token: "munki-secret"}
	repository := newStaticRepository("C02MUNKI")
	router := newMunkiContractRouter(verifier, repository)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/catalogs/production", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set("Serial", "C02MUNKI")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if verifier.agent != agentauth.AgentMunki {
		t.Fatalf("agent = %q, want munki", verifier.agent)
	}
	if repository.serial != "C02MUNKI" {
		t.Fatalf("serial = %q, want C02MUNKI", repository.serial)
	}
}

func TestMunkiHTTPMapsVerifierErrorsToServerErrors(t *testing.T) {
	router := newMunkiContractRouter(errorVerifier{}, newStaticRepository("C02MUNKI"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/catalogs/production", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set("Serial", "C02MUNKI")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func newMunkiContractRouter(verifier AgentSecretVerifier, repository Repository) chi.Router {
	r := chi.NewRouter()
	RegisterMunkiRoutes(r, verifier, repository, nil)
	return r
}

type staticVerifier struct {
	agent agentauth.Agent
	token string
}

func (v staticVerifier) Verify(_ context.Context, agent agentauth.Agent, token string) (bool, error) {
	return agent == v.agent && token == v.token, nil
}

type recordingVerifier struct {
	agent agentauth.Agent
	token string
}

func (v *recordingVerifier) Verify(_ context.Context, agent agentauth.Agent, token string) (bool, error) {
	v.agent = agent
	return token == v.token, nil
}

type errorVerifier struct{}

func (errorVerifier) Verify(context.Context, agentauth.Agent, string) (bool, error) {
	return false, errors.New("verifier failed")
}

type staticRepository struct {
	service *munki.Service
	want    string
	serial  string
}

func newStaticRepository(serial string) *staticRepository {
	return &staticRepository{service: munki.NewService(nil), want: serial}
}

func (r *staticRepository) ResolveClient(_ context.Context, serial string) (munki.ClientHost, error) {
	r.serial = serial
	if serial != r.want {
		return munki.ClientHost{}, munki.ErrNotFound
	}
	return munki.ClientHost{ID: 1, Serial: serial, DisplayName: "Test MacBook"}, nil
}

func (r *staticRepository) Manifest(ctx context.Context, client munki.ClientHost, name string) ([]byte, error) {
	return r.service.Manifest(ctx, client, name)
}

func (r *staticRepository) Catalog(ctx context.Context, client munki.ClientHost, name string) ([]byte, error) {
	return r.service.Catalog(ctx, client, name)
}
