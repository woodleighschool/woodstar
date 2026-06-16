package protocol

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"howett.net/plist"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestMunkiHTTPFetchesManifestAndCatalog(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepositoryWithPackages([]munkisoftware.EffectivePackage{
			{
				TargetID:   10,
				SoftwareID: 1,
				Actions: []munkisoftware.Action{
					munkisoftware.ActionManagedInstalls,
					munkisoftware.ActionManagedUpdates,
					munkisoftware.ActionDefaultInstalls,
				},
				Selector: munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package:  staticMunkiPackage(20, 1, "GoogleChrome", "148.0.0.1"),
			},
			{
				TargetID:   11,
				SoftwareID: 2,
				Actions:    []munkisoftware.Action{munkisoftware.ActionOptionalInstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package:    staticMunkiPackage(21, 2, "Slack", "4.50.0"),
			},
			{
				TargetID:   12,
				SoftwareID: 3,
				Actions:    []munkisoftware.Action{munkisoftware.ActionManagedUninstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package:    staticMunkiPackage(22, 3, "LegacyVPN", "1.0"),
			},
			{
				TargetID:   13,
				SoftwareID: 4,
				Actions: []munkisoftware.Action{
					munkisoftware.ActionOptionalInstalls,
					munkisoftware.ActionFeaturedItems,
				},
				Selector: munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package:  staticMunkiPackage(23, 4, "FeaturedApp", "3.2.1"),
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

func TestMunkiCatalogUsesStableInstallerItemLocation(t *testing.T) {
	objectID := int64(42)
	service := munki.NewRepositoryService(munki.Dependencies{
		Packages: staticPackageResolver{packages: []munkisoftware.EffectivePackage{
			{
				TargetID:   10,
				SoftwareID: 1,
				Actions:    []munkisoftware.Action{munkisoftware.ActionManagedInstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package: packages.Package{
					ID:                20,
					SoftwareID:        1,
					SoftwareName:      "GoogleChrome",
					Version:           "148.0.0.1",
					InstallerType:     packages.InstallerTypePkg,
					InstallerObjectID: &objectID,
				},
			},
		}},
		Objects: staticObjectResolver{objects: map[int64]storage.Object{
			objectID: {ID: objectID, Prefix: packages.ObjectPrefix, Filename: "GoogleChrome.pkg"},
		}},
	})

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
	if got := decoded[0]["installer_item_location"]; got != "packages/20/installer/GoogleChrome.pkg" {
		t.Fatalf("installer_item_location = %q, want package item location", got)
	}
	if _, ok := decoded[0]["PackageCompleteURL"]; ok {
		t.Fatalf("PackageCompleteURL should not override the client's SoftwareRepoURL: %+v", decoded[0])
	}
	if _, ok := decoded[0]["PackageURL"]; ok {
		t.Fatalf("PackageURL was rendered from stored pkginfo: %+v", decoded[0])
	}
}

func TestMunkiCatalogOmitsPackageURLsWithoutInstallerItemLocation(t *testing.T) {
	service := munki.NewRepositoryService(munki.Dependencies{
		Packages: staticPackageResolver{packages: []munkisoftware.EffectivePackage{
			{
				TargetID:   10,
				SoftwareID: 1,
				Actions:    []munkisoftware.Action{munkisoftware.ActionManagedInstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package:    staticMunkiPackage(20, 1, "ExternalURLApp", "1.0"),
			},
		}},
	})

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
		t.Fatalf("PackageCompleteURL rendered without an installer item location: %+v", decoded[0])
	}
	if _, ok := decoded[0]["PackageURL"]; ok {
		t.Fatalf("PackageURL rendered without an installer item location: %+v", decoded[0])
	}
}

func assertManifestPlist(t *testing.T, body []byte) {
	t.Helper()
	var decoded struct {
		Catalogs          []string `plist:"catalogs"`
		DisplayName       string   `plist:"display_name"`
		ManagedInstalls   []string `plist:"managed_installs"`
		ManagedUninstalls []string `plist:"managed_uninstalls"`
		ManagedUpdates    []string `plist:"managed_updates"`
		OptionalInstalls  []string `plist:"optional_installs"`
		DefaultInstalls   []string `plist:"default_installs"`
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
	if !sameStrings(decoded.ManagedInstalls, []string{"1"}) {
		t.Fatalf("managed_installs = %v, want [1]", decoded.ManagedInstalls)
	}
	if !sameStrings(decoded.OptionalInstalls, []string{"2", "4"}) {
		t.Fatalf("optional_installs = %v, want [2 4]", decoded.OptionalInstalls)
	}
	if !sameStrings(decoded.ManagedUninstalls, []string{"3"}) {
		t.Fatalf("managed_uninstalls = %v, want [3]", decoded.ManagedUninstalls)
	}
	if !sameStrings(decoded.ManagedUpdates, []string{"1"}) {
		t.Fatalf("managed_updates = %v, want [1]", decoded.ManagedUpdates)
	}
	if !sameStrings(decoded.DefaultInstalls, []string{"1"}) {
		t.Fatalf("default_installs = %v, want [1]", decoded.DefaultInstalls)
	}
	if !sameStrings(decoded.FeaturedItems, []string{"4"}) {
		t.Fatalf("featured_items = %v, want [4]", decoded.FeaturedItems)
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
	if decoded[0]["name"] != "1" ||
		decoded[0]["display_name"] != "GoogleChrome" ||
		decoded[0]["version"] != "148.0.0.1" {
		t.Fatalf("first catalog item = %+v, want package 20 / GoogleChrome 148.0.0.1", decoded[0])
	}
}

func TestMunkiHTTPRendersLatestSoftwareIDOnceWithAllPkginfos(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepositoryWithPackages([]munkisoftware.EffectivePackage{
			{
				TargetID:   10,
				SoftwareID: 1,
				Actions:    []munkisoftware.Action{munkisoftware.ActionOptionalInstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package:    staticMunkiPackage(20, 1, "GoogleChrome", "148.0.0.1"),
			},
			{
				TargetID:   10,
				SoftwareID: 1,
				Actions:    []munkisoftware.Action{munkisoftware.ActionOptionalInstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package:    staticMunkiPackage(21, 1, "GoogleChrome", "149.0.0.1"),
			},
		}),
	)

	manifest := httptest.NewRecorder()
	manifestReq := httptest.NewRequest(http.MethodGet, "/munki/manifests/C02MUNKI", nil)
	manifestReq.Header.Set("Authorization", "Bearer munki-secret")
	manifestReq.Header.Set("Serial", "C02MUNKI")
	router.ServeHTTP(manifest, manifestReq)

	if manifest.Code != http.StatusOK {
		t.Fatalf("manifest status = %d, want %d; body = %q", manifest.Code, http.StatusOK, manifest.Body.String())
	}
	var manifestBody struct {
		OptionalInstalls []string `plist:"optional_installs"`
	}
	if _, err := plist.Unmarshal(manifest.Body.Bytes(), &manifestBody); err != nil {
		t.Fatalf("manifest plist: %v", err)
	}
	if !sameStrings(manifestBody.OptionalInstalls, []string{"1"}) {
		t.Fatalf("optional_installs = %v, want [1]", manifestBody.OptionalInstalls)
	}

	catalog := httptest.NewRecorder()
	catalogReq := httptest.NewRequest(http.MethodGet, "/munki/catalogs/production", nil)
	catalogReq.Header.Set("Authorization", "Bearer munki-secret")
	catalogReq.Header.Set("Serial", "C02MUNKI")
	router.ServeHTTP(catalog, catalogReq)

	if catalog.Code != http.StatusOK {
		t.Fatalf("catalog status = %d, want %d; body = %q", catalog.Code, http.StatusOK, catalog.Body.String())
	}
	var catalogBody []map[string]any
	if _, err := plist.Unmarshal(catalog.Body.Bytes(), &catalogBody); err != nil {
		t.Fatalf("catalog plist: %v", err)
	}
	if len(catalogBody) != 2 {
		t.Fatalf("catalog items = %d, want 2", len(catalogBody))
	}
	for _, item := range catalogBody {
		if item["name"] != "1" {
			t.Fatalf("catalog item name = %v, want 1: %+v", item["name"], item)
		}
	}
}

func TestMunkiHTTPRendersFirstOverlappingEffectivePackage(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepositoryWithPackages([]munkisoftware.EffectivePackage{
			{
				TargetID:   10,
				SoftwareID: 1,
				Actions:    []munkisoftware.Action{munkisoftware.ActionManagedInstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package:    staticMunkiPackage(20, 1, "OverlapApp", "1.0"),
			},
			{
				TargetID:   11,
				SoftwareID: 1,
				Actions:    []munkisoftware.Action{munkisoftware.ActionOptionalInstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package:    staticMunkiPackage(21, 1, "OverlapApp", "1.1"),
			},
			{
				TargetID:   12,
				SoftwareID: 1,
				Actions:    []munkisoftware.Action{munkisoftware.ActionManagedUninstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package:    staticMunkiPackage(22, 1, "OverlapApp", "1.2"),
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
	if !sameStrings(decoded.ManagedInstalls, []string{"1"}) {
		t.Fatalf("managed_installs = %v, want [1]", decoded.ManagedInstalls)
	}
	if len(decoded.ManagedUninstalls) != 0 || len(decoded.OptionalInstalls) != 0 {
		t.Fatalf("manifest still has later conflicting rows: %+v", decoded)
	}
}

func TestMunkiHTTPRendersPinnedPackageName(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepositoryWithPackages([]munkisoftware.EffectivePackage{
			{
				TargetID:   10,
				SoftwareID: 1,
				Actions:    []munkisoftware.Action{munkisoftware.ActionManagedInstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageSpecific},
				Package:    staticMunkiPackage(20, 1, "PinnedApp", "1.0"),
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
		ManagedInstalls []string `plist:"managed_installs"`
	}
	if _, err := plist.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("response is not a manifest plist: %v", err)
	}
	if !sameStrings(decoded.ManagedInstalls, []string{"1--1.0"}) {
		t.Fatalf("managed_installs = %v, want [1--1.0]", decoded.ManagedInstalls)
	}
}

func TestMunkiHTTPRequiresMunkiBearerSecret(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepository(),
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
		newStaticRepository(),
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
		newStaticRepository(),
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
	repository := newStaticRepository()
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

func TestMunkiHTTPRedirectsPackageFileWithMunkiIdentity(t *testing.T) {
	repository := newStaticRepository()
	repository.fileURL = "munki/packages/42/GoogleChrome.pkg"
	store := &fakePresigner{presignURL: "https://storage.example/GoogleChrome.pkg?signature=test"}
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		repository,
		store,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/pkgs/packages/20/installer/GoogleChrome.pkg", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set("Serial", "C02MUNKI")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusFound, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != store.presignURL {
		t.Fatalf("Location = %q, want %q", got, store.presignURL)
	}
	if repository.fileClass != "package" ||
		repository.fileKey != "packages/20/installer/GoogleChrome.pkg" {
		t.Fatalf("file request = class %q key %q", repository.fileClass, repository.fileKey)
	}
	if store.gotKey != "munki/packages/42/GoogleChrome.pkg" {
		t.Fatalf("presigned key = %q", store.gotKey)
	}
	if repository.serial != "C02MUNKI" {
		t.Fatalf("serial = %q, want C02MUNKI", repository.serial)
	}
}

func TestMunkiHTTPRedirectsIconFileWithNestedIconName(t *testing.T) {
	repository := newStaticRepository()
	repository.fileURL = "munki/icons/7/GoogleChrome.png"
	store := &fakePresigner{presignURL: "https://storage.example/icon.png?signature=test"}
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		repository,
		store,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/icons/7-GoogleChrome.png", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set("Serial", "C02MUNKI")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusFound, rec.Body.String())
	}
	if repository.fileClass != "icon" ||
		repository.fileKey != "7-GoogleChrome.png" {
		t.Fatalf("file request = class %q key %q", repository.fileClass, repository.fileKey)
	}
	if store.gotKey != "munki/icons/7/GoogleChrome.png" {
		t.Fatalf("presigned key = %q", store.gotKey)
	}
}

func TestMunkiHTTPMapsVerifierErrorsToServerErrors(t *testing.T) {
	router := newMunkiContractRouter(errorVerifier{}, newStaticRepository())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/catalogs/production", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set("Serial", "C02MUNKI")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func newMunkiContractRouter(
	verifier agentauth.SecretVerifier,
	repository Repository,
	store ...storage.Presigner,
) chi.Router {
	var s storage.Presigner
	if len(store) > 0 {
		s = store[0]
	}
	r := chi.NewRouter()
	RegisterMunkiRoutes(r, verifier, repository, s, nil)
	return r
}

type fakePresigner struct {
	presignURL string
	gotKey     string
}

func (f *fakePresigner) PresignGet(
	_ context.Context,
	key string,
	_ time.Duration,
	_ storage.GetOptions,
) (string, error) {
	f.gotKey = key
	return f.presignURL, nil
}

func (f *fakePresigner) PresignPut(
	context.Context,
	string,
	time.Duration,
	storage.PutOptions,
) (storage.UploadTarget, error) {
	return storage.UploadTarget{}, nil
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
	service   *munki.RepositoryService
	want      string
	serial    string
	fileURL   string
	fileErr   error
	fileClass string
	fileKey   string
}

func newStaticRepository() *staticRepository {
	return newStaticRepositoryWithPackages(nil)
}

func newStaticRepositoryWithPackages(packages []munkisoftware.EffectivePackage) *staticRepository {
	return &staticRepository{
		service: munki.NewRepositoryService(munki.Dependencies{Packages: staticPackageResolver{packages: packages}}),
		want:    "C02MUNKI",
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

func (r *staticRepository) ResolvePackageFile(
	_ context.Context,
	_ munki.ClientHost,
	key string,
) (string, error) {
	return r.resolve("package", key)
}

func (r *staticRepository) ResolveIconFile(
	_ context.Context,
	_ munki.ClientHost,
	key string,
) (string, error) {
	return r.resolve("icon", key)
}

func (r *staticRepository) resolve(class, key string) (string, error) {
	r.fileClass = class
	r.fileKey = key
	if r.fileErr != nil {
		return "", r.fileErr
	}
	if r.fileURL != "" {
		return r.fileURL, nil
	}
	return key, nil
}

type staticPackageResolver struct {
	packages []munkisoftware.EffectivePackage
}

func (r staticPackageResolver) EffectivePackagesForHost(
	_ context.Context,
	_ int64,
) ([]munkisoftware.EffectivePackage, error) {
	return munkisoftware.ResolveEffectivePackages(r.packages), nil
}

type staticObjectResolver struct {
	objects map[int64]storage.Object
}

func (r staticObjectResolver) ListByIDs(
	_ context.Context,
	ids []int64,
) (map[int64]storage.Object, error) {
	objects := make(map[int64]storage.Object, len(ids))
	for _, id := range ids {
		if obj, ok := r.objects[id]; ok {
			objects[id] = obj
		}
	}
	return objects, nil
}

func staticMunkiPackage(id int64, softwareID int64, name string, version string) packages.Package {
	return packages.Package{
		ID:            id,
		SoftwareID:    softwareID,
		SoftwareName:  name,
		Version:       version,
		InstallerType: packages.InstallerTypeNoPkg,
		OnDemand:      true,
	}
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
