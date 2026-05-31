package protocol

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/munki"
)

func TestMunkiHTTPFetchesManifestAndCatalog(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		munki.NewService(),
	)

	for _, path := range []string{"/munki/manifests/site_default", "/munki/catalogs/production"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Authorization", "Bearer munki-secret")

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); got != plistContentType {
				t.Fatalf("Content-Type = %q, want %q", got, plistContentType)
			}
			var plist struct {
				XMLName xml.Name `xml:"plist"`
			}
			if err := xml.Unmarshal(rec.Body.Bytes(), &plist); err != nil {
				t.Fatalf("response is not XML plist: %v", err)
			}
			if plist.XMLName.Local != "plist" {
				t.Fatalf("root element = %q, want plist", plist.XMLName.Local)
			}
		})
	}
}

func TestMunkiHTTPRequiresMunkiBearerSecret(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		munki.NewService(),
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
			req := httptest.NewRequest(http.MethodGet, "/munki/manifests/site_default", nil)
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}

			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestMunkiHTTPVerifiesMunkiAgent(t *testing.T) {
	verifier := &recordingVerifier{token: "munki-secret"}
	router := newMunkiContractRouter(verifier, munki.NewService())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/catalogs/production", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if verifier.agent != agentauth.AgentMunki {
		t.Fatalf("agent = %q, want munki", verifier.agent)
	}
}

func TestMunkiHTTPMapsVerifierErrorsToServerErrors(t *testing.T) {
	router := newMunkiContractRouter(errorVerifier{}, munki.NewService())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/catalogs/production", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")

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
