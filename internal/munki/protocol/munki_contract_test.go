package protocol

import (
	"context"
	"encoding/json"
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
		newStaticRepositoryWithReleases("C02MUNKI", []munki.EffectiveRelease{
			{
				AssignmentID: 10,
				Intent:       munki.IntentEnsureInstalled,
				Release: munki.Release{
					ID:      20,
					Name:    "GoogleChrome",
					Version: "148.0.0.1",
					Pkginfo: json.RawMessage(
						`{"name":"GoogleChrome","version":"148.0.0.1","installer_type":"nopkg"}`,
					),
				},
			},
			{
				AssignmentID: 11,
				Intent:       munki.IntentOptional,
				Release: munki.Release{
					ID:      21,
					Name:    "Slack",
					Version: "4.50.0",
					Pkginfo: json.RawMessage(
						`{"name":"Slack","version":"4.50.0","installer_type":"nopkg"}`,
					),
				},
			},
			{
				AssignmentID: 12,
				Intent:       munki.IntentEnsureAbsent,
				Release: munki.Release{
					ID:      22,
					Name:    "LegacyVPN",
					Version: "1.0",
					Pkginfo: json.RawMessage(
						`{"name":"LegacyVPN","version":"1.0","installer_type":"nopkg"}`,
					),
				},
			},
			{
				AssignmentID: 13,
				Intent:       munki.IntentFeatured,
				Release: munki.Release{
					ID:      23,
					Name:    "SelfServiceApp",
					Version: "3.2.1",
					Pkginfo: json.RawMessage(
						`{"name":"SelfServiceApp","version":"3.2.1","installer_type":"nopkg"}`,
					),
				},
			},
		}),
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

func TestMunkiCatalogUsesStableArtifactURL(t *testing.T) {
	artifactID := int64(42)
	service := munki.NewService(
		nil,
		staticReleaseResolver{releases: []munki.EffectiveRelease{
			{
				AssignmentID: 10,
				Intent:       munki.IntentEnsureInstalled,
				Release: munki.Release{
					ID:      20,
					Name:    "GoogleChrome",
					Version: "148.0.0.1",
					Pkginfo: json.RawMessage(
						`{"name":"GoogleChrome","version":"148.0.0.1","PackageCompleteURL":"https://s3.example/raw?expires=1","PackageURL":"https://packages.example"}`,
					),
					InstallerArtifactID:       &artifactID,
					InstallerArtifactLocation: "apps/GoogleChrome.pkg",
				},
			},
		}},
		munki.WithPublicURL("https://woodstar.example"),
	)

	body, err := service.Catalog(context.Background(), munki.ClientHost{ID: 1, Serial: "C02MUNKI"}, "production")
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}

	var decoded []map[string]any
	if _, err := plist.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("catalog plist: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("catalog items = %d, want 1", len(decoded))
	}
	if got := decoded[0]["installer_item_location"]; got != "apps/GoogleChrome.pkg" {
		t.Fatalf("installer_item_location = %q, want artifact location", got)
	}
	if got := decoded[0]["PackageCompleteURL"]; got != "https://woodstar.example/munki/pkgs/apps/GoogleChrome.pkg" {
		t.Fatalf("PackageCompleteURL = %q", got)
	}
	if _, ok := decoded[0]["PackageURL"]; ok {
		t.Fatalf("PackageURL was rendered from stored pkginfo: %+v", decoded[0])
	}
}

func TestMunkiCatalogStripsCallerPackageURLs(t *testing.T) {
	service := munki.NewService(
		nil,
		staticReleaseResolver{releases: []munki.EffectiveRelease{
			{
				AssignmentID: 10,
				Intent:       munki.IntentEnsureInstalled,
				Release: munki.Release{
					ID:      20,
					Name:    "ExternalURLApp",
					Version: "1.0",
					Pkginfo: json.RawMessage(
						`{"name":"ExternalURLApp","version":"1.0","PackageCompleteURL":"https://s3.example/raw?expires=1","PackageURL":"https://packages.example"}`,
					),
				},
			},
		}},
		munki.WithPublicURL("https://woodstar.example"),
	)

	body, err := service.Catalog(context.Background(), munki.ClientHost{ID: 1, Serial: "C02MUNKI"}, "production")
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}

	var decoded []map[string]any
	if _, err := plist.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("catalog plist: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("catalog items = %d, want 1", len(decoded))
	}
	if _, ok := decoded[0]["PackageCompleteURL"]; ok {
		t.Fatalf("PackageCompleteURL was rendered from stored pkginfo: %+v", decoded[0])
	}
	if _, ok := decoded[0]["PackageURL"]; ok {
		t.Fatalf("PackageURL was rendered from stored pkginfo: %+v", decoded[0])
	}
}

func assertManifestPlist(t *testing.T, body []byte) {
	t.Helper()
	var decoded struct {
		Catalogs          []string `plist:"catalogs"`
		DisplayName       string   `plist:"display_name"`
		ManagedInstalls   []string `plist:"managed_installs"`
		ManagedUninstalls []string `plist:"managed_uninstalls"`
		OptionalInstalls  []string `plist:"optional_installs"`
		FeaturedItems     []string `plist:"featured_items"`
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
	if !sameStrings(decoded.ManagedInstalls, []string{"GoogleChrome"}) {
		t.Fatalf("managed_installs = %v, want [GoogleChrome]", decoded.ManagedInstalls)
	}
	if !sameStrings(decoded.OptionalInstalls, []string{"Slack", "SelfServiceApp"}) {
		t.Fatalf("optional_installs = %v, want [Slack SelfServiceApp]", decoded.OptionalInstalls)
	}
	if !sameStrings(decoded.ManagedUninstalls, []string{"LegacyVPN"}) {
		t.Fatalf("managed_uninstalls = %v, want [LegacyVPN]", decoded.ManagedUninstalls)
	}
	if !sameStrings(decoded.FeaturedItems, []string{"SelfServiceApp"}) {
		t.Fatalf("featured_items = %v, want [SelfServiceApp]", decoded.FeaturedItems)
	}
}

func assertCatalogPlist(t *testing.T, body []byte) {
	t.Helper()
	var decoded []map[string]any
	if _, err := plist.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("response is not a catalog plist: %v", err)
	}
	if len(decoded) != 4 {
		t.Fatalf("catalog items = %d, want 4", len(decoded))
	}
	if decoded[0]["name"] != "GoogleChrome" || decoded[0]["version"] != "148.0.0.1" {
		t.Fatalf("first catalog item = %+v, want GoogleChrome 148.0.0.1", decoded[0])
	}
}

func TestMunkiHTTPCollapsesOverlappingAssignments(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepositoryWithReleases("C02MUNKI", []munki.EffectiveRelease{
			{
				AssignmentID: 10,
				Intent:       munki.IntentEnsureInstalled,
				Release: munki.Release{
					ID:      20,
					Name:    "OverlapApp",
					Version: "1.0",
					Pkginfo: json.RawMessage(
						`{"name":"OverlapApp","version":"1.0","installer_type":"nopkg"}`,
					),
				},
			},
			{
				AssignmentID: 11,
				Intent:       munki.IntentOptional,
				Release: munki.Release{
					ID:      21,
					Name:    "OverlapApp",
					Version: "1.1",
					Pkginfo: json.RawMessage(
						`{"name":"OverlapApp","version":"1.1","installer_type":"nopkg"}`,
					),
				},
			},
			{
				AssignmentID: 12,
				Intent:       munki.IntentEnsureAbsent,
				Release: munki.Release{
					ID:      22,
					Name:    "OverlapApp",
					Version: "1.2",
					Pkginfo: json.RawMessage(
						`{"name":"OverlapApp","version":"1.2","installer_type":"nopkg"}`,
					),
				},
			},
		}),
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/manifests/C02MUNKI", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set("Serial", "C02MUNKI")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	var decoded struct {
		ManagedInstalls   []string `plist:"managed_installs"`
		ManagedUninstalls []string `plist:"managed_uninstalls"`
		OptionalInstalls  []string `plist:"optional_installs"`
	}
	if _, err := plist.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("response is not a manifest plist: %v", err)
	}
	if !sameStrings(decoded.ManagedUninstalls, []string{"OverlapApp"}) {
		t.Fatalf("managed_uninstalls = %v, want [OverlapApp]", decoded.ManagedUninstalls)
	}
	if len(decoded.ManagedInstalls) != 0 || len(decoded.OptionalInstalls) != 0 {
		t.Fatalf("manifest still has conflicting installs: %+v", decoded)
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

func TestMunkiHTTPUsesSerialHeaderNotManifestName(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepository("C02MUNKI"),
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/manifests/site_default", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set("Serial", "C02MUNKI")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
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

func TestMunkiHTTPRedirectsArtifactWithMunkiIdentity(t *testing.T) {
	repository := newStaticRepository("C02MUNKI")
	repository.artifactURL = "https://storage.example/GoogleChrome.pkg?signature=test"
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		repository,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/pkgs/apps/GoogleChrome.pkg", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set("Serial", "C02MUNKI")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusFound, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != repository.artifactURL {
		t.Fatalf("Location = %q, want %q", got, repository.artifactURL)
	}
	if repository.artifactKind != munki.ArtifactKindPackage || repository.artifactLocation != "apps/GoogleChrome.pkg" {
		t.Fatalf("artifact request = kind %q location %q", repository.artifactKind, repository.artifactLocation)
	}
	if repository.serial != "C02MUNKI" {
		t.Fatalf("serial = %q, want C02MUNKI", repository.serial)
	}
}

func TestMunkiHTTPMapsMissingArtifactStorageToUnavailable(t *testing.T) {
	repository := newStaticRepository("C02MUNKI")
	repository.artifactErr = munki.ErrStorageUnavailable
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		repository,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/pkgs/apps/GoogleChrome.pkg", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set("Serial", "C02MUNKI")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
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
	service          *munki.Service
	want             string
	serial           string
	artifactURL      string
	artifactErr      error
	artifactKind     munki.ArtifactKind
	artifactLocation string
}

func newStaticRepository(serial string) *staticRepository {
	return newStaticRepositoryWithReleases(serial, nil)
}

func newStaticRepositoryWithReleases(serial string, releases []munki.EffectiveRelease) *staticRepository {
	return &staticRepository{
		service: munki.NewService(nil, staticReleaseResolver{releases: releases}),
		want:    serial,
	}
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

func (r *staticRepository) ArtifactRedirect(
	_ context.Context,
	_ munki.ClientHost,
	kind munki.ArtifactKind,
	location string,
) (string, error) {
	r.artifactKind = kind
	r.artifactLocation = location
	if r.artifactErr != nil {
		return "", r.artifactErr
	}
	if r.artifactURL == "" {
		return "", munki.ErrNotFound
	}
	return r.artifactURL, nil
}

type staticReleaseResolver struct {
	releases []munki.EffectiveRelease
}

func (r staticReleaseResolver) EffectiveReleasesForHost(_ context.Context, _ int64) ([]munki.EffectiveRelease, error) {
	return r.releases, nil
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
