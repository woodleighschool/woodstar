package protocol

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"howett.net/plist"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
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
		{path: "/munki/catalogs/woodstar", assert: assertCatalogPlist},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Authorization", "Bearer munki-secret")

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

func TestMunkiHTTPHonorsPlistETag(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepositoryWithPackages([]munkisoftware.EffectivePackage{
			{
				TargetID:   10,
				SoftwareID: 1,
				Actions:    []munkisoftware.Action{munkisoftware.ActionManagedInstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package:    staticMunkiPackage(20, 1, "GoogleChrome", "148.0.0.1"),
			},
		}),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/catalogs/woodstar", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag is empty")
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/munki/catalogs/woodstar", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set("If-None-Match", etag)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("cached status = %d, want %d; body = %q", rec.Code, http.StatusNotModified, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("cached body = %q, want empty", rec.Body.String())
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
					ID:                  20,
					SoftwareID:          1,
					SoftwareName:        "GoogleChrome",
					SoftwareDisplayName: "Google Chrome",
					Version:             "148.0.0.1",
					InstallerType:       packages.InstallerTypePkg,
					InstallerObjectID:   &objectID,
				},
			},
		}},
		Objects: staticObjectResolver{objects: map[int64]storage.Object{
			objectID: {ID: objectID, Prefix: packages.ObjectPrefix, Filename: "GoogleChrome.pkg"},
		}},
	})

	body, err := service.Catalog(context.Background(), "woodstar")
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

	body, err := service.Catalog(context.Background(), "woodstar")
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
	var raw map[string]any
	if _, err := plist.Unmarshal(body, &raw); err != nil {
		t.Fatalf("response is not a manifest plist: %v", err)
	}
	if _, ok := raw["display_name"]; ok {
		t.Fatalf("manifest rendered display_name: %+v", raw)
	}
	var decoded struct {
		Catalogs          []string `plist:"catalogs"`
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
	if got := decoded.Catalogs; len(got) != 1 || got[0] != "woodstar" {
		t.Fatalf("catalogs = %v, want [woodstar]", got)
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
	catalogReq := httptest.NewRequest(http.MethodGet, "/munki/catalogs/woodstar", nil)
	catalogReq.Header.Set("Authorization", "Bearer munki-secret")
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

			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestMunkiHTTPRequiresExistingManifestSerial(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepository(),
	)

	cases := []struct {
		name string
		path string
	}{
		{name: "unknown serial", path: "/munki/manifests/C02UNKNOWN"},
		{name: "site default", path: "/munki/manifests/site_default"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Authorization", "Bearer munki-secret")

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusNotFound, rec.Body.String())
			}
		})
	}
}

func TestMunkiHTTPVerifiesMunkiAgent(t *testing.T) {
	verifier := &recordingVerifier{token: "munki-secret"}
	repository := newStaticRepository()
	router := newMunkiContractRouter(verifier, repository)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/catalogs/woodstar", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if verifier.agent != agentauth.AgentMunki {
		t.Fatalf("agent = %q, want munki", verifier.agent)
	}
}

func TestMunkiHTTPRedirectsPackageFileWithBearer(t *testing.T) {
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
}

func TestMunkiHTTPRedirectsPackageFileToDistributionPoint(t *testing.T) {
	repository := newStaticRepository()
	repository.fileURL = "munki/packages/42/GoogleChrome.pkg"
	repository.packageID = 20
	repository.fileSHA = strings.Repeat("a", 64)
	repository.fileSize = 4096
	store := &fakePresigner{presignURL: "https://storage.example/direct"}
	selector := &fakeSelector{
		url: "https://mdp.example/munki/pkgs/packages/20/installer/GoogleChrome.pkg?cap=grant",
		ok:  true,
	}

	router := chi.NewRouter()
	NewServer(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		repository,
		selector,
		store,
		testLogger(),
	).RegisterRoutes(router)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/pkgs/packages/20/installer/GoogleChrome.pkg", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusFound, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != selector.url {
		t.Fatalf("Location = %q, want distribution point URL %q", got, selector.url)
	}
	if selector.got.PackageID != 20 || selector.got.SHA256 != repository.fileSHA || selector.got.SizeBytes != 4096 {
		t.Fatalf("selection integrity claims = %+v", selector.got)
	}
	if selector.got.InstallerItemLocation != "packages/20/installer/GoogleChrome.pkg" {
		t.Fatalf("selection installer_item_location = %q", selector.got.InstallerItemLocation)
	}
	if store.gotKey != "" {
		t.Fatalf("Woodstar presign should be skipped, got key %q", store.gotKey)
	}
}

func TestMunkiHTTPServesDirectWhenNoDistributionPoint(t *testing.T) {
	repository := newStaticRepository()
	repository.fileURL = "munki/packages/42/GoogleChrome.pkg"
	repository.packageID = 20
	store := &fakePresigner{presignURL: "https://storage.example/direct"}
	selector := &fakeSelector{ok: false}

	router := chi.NewRouter()
	NewServer(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		repository,
		selector,
		store,
		testLogger(),
	).RegisterRoutes(router)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/munki/pkgs/packages/20/installer/GoogleChrome.pkg", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusFound, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != store.presignURL {
		t.Fatalf("Location = %q, want Woodstar presign %q", got, store.presignURL)
	}
	if store.gotKey != "munki/packages/42/GoogleChrome.pkg" {
		t.Fatalf("presigned key = %q, want installer key", store.gotKey)
	}
}

func TestMunkiCatalogProjectsInstallerHashAndSize(t *testing.T) {
	objectID := int64(42)
	sha := strings.Repeat("a", 64)
	size := int64(4096)
	service := munki.NewRepositoryService(munki.Dependencies{
		Packages: staticPackageResolver{packages: []munkisoftware.EffectivePackage{
			{
				TargetID:   10,
				SoftwareID: 1,
				Actions:    []munkisoftware.Action{munkisoftware.ActionManagedInstalls},
				Selector:   munkisoftware.PackageSelector{Strategy: munkisoftware.PackageLatest},
				Package: packages.Package{
					ID:                  20,
					SoftwareID:          1,
					SoftwareName:        "GoogleChrome",
					SoftwareDisplayName: "Google Chrome",
					Version:             "148.0.0.1",
					InstallerType:       packages.InstallerTypePkg,
					InstallerObjectID:   &objectID,
				},
			},
		}},
		Objects: staticObjectResolver{objects: map[int64]storage.Object{
			objectID: {
				ID:        objectID,
				Prefix:    packages.ObjectPrefix,
				Filename:  "GoogleChrome.pkg",
				SHA256:    &sha,
				SizeBytes: &size,
			},
		}},
	})

	body, err := service.Catalog(context.Background(), "woodstar")
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}

	var decoded []struct {
		InstallerItemHash string `plist:"installer_item_hash"`
		InstallerItemSize int64  `plist:"installer_item_size"`
	}
	if _, err := plist.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("catalog plist: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("catalog items = %d, want 1", len(decoded))
	}
	if decoded[0].InstallerItemHash != sha {
		t.Fatalf("installer_item_hash = %q, want %q", decoded[0].InstallerItemHash, sha)
	}
	// Munki pkginfo reports installer_item_size in kilobytes.
	if decoded[0].InstallerItemSize != 4 {
		t.Fatalf("installer_item_size = %d KB, want 4", decoded[0].InstallerItemSize)
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
	req := httptest.NewRequest(http.MethodGet, "/munki/catalogs/woodstar", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")

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
	NewServer(verifier, repository, &fakeSelector{}, s, testLogger()).RegisterRoutes(r)
	return r
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

type fakeSelector struct {
	url string
	ok  bool
	got mdp.SelectionRequest
}

func (f *fakeSelector) SelectRedirect(
	_ context.Context,
	req mdp.SelectionRequest,
) (string, bool) {
	f.got = req
	return f.url, f.ok
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
	service      *munki.RepositoryService
	manifestName string
	fileURL      string
	fileErr      error
	fileClass    string
	fileKey      string
	packageID    int64
	fileSHA      string
	fileSize     int64
}

func newStaticRepository() *staticRepository {
	return newStaticRepositoryWithPackages(nil)
}

func newStaticRepositoryWithPackages(packages []munkisoftware.EffectivePackage) *staticRepository {
	return &staticRepository{
		service: munki.NewRepositoryService(munki.Dependencies{
			Hosts:    staticHostResolver{serial: "C02MUNKI"},
			Software: staticPackageResolver{packages: packages},
			Packages: staticPackageResolver{packages: packages},
		}),
	}
}

func (r *staticRepository) Manifest(ctx context.Context, name string) ([]byte, error) {
	r.manifestName = name
	return r.service.Manifest(ctx, name)
}

func (r *staticRepository) Catalog(ctx context.Context, name string) ([]byte, error) {
	return r.service.Catalog(ctx, name)
}

func (r *staticRepository) ResolvePackageFile(
	_ context.Context,
	key string,
) (munki.PackageInstaller, error) {
	r.fileClass = "package"
	r.fileKey = key
	if r.fileErr != nil {
		return munki.PackageInstaller{}, r.fileErr
	}
	installer := munki.PackageInstaller{
		PackageID:             r.packageID,
		InstallerItemLocation: key,
		Key:                   key,
		SHA256:                r.fileSHA,
		SizeBytes:             r.fileSize,
	}
	if r.fileURL != "" {
		installer.Key = r.fileURL
	}
	return installer, nil
}

func (r *staticRepository) ResolveIconFile(
	_ context.Context,
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

func (r staticPackageResolver) ListRepositoryPackages(
	_ context.Context,
) ([]packages.Package, error) {
	pkgs := make([]packages.Package, 0, len(r.packages))
	for _, pkg := range r.packages {
		pkgs = append(pkgs, pkg.Package)
	}
	return pkgs, nil
}

func (r staticPackageResolver) PackagesByID(
	_ context.Context,
	ids []int64,
) ([]packages.Package, error) {
	pkgs := make([]packages.Package, 0, len(ids))
	for _, id := range ids {
		for _, pkg := range r.packages {
			if pkg.Package.ID == id {
				pkgs = append(pkgs, pkg.Package)
				break
			}
		}
	}
	return pkgs, nil
}

func (r staticPackageResolver) RepositoryPackagesByIconObjectID(
	_ context.Context,
	iconObjectID int64,
) ([]packages.Package, error) {
	pkgs := make([]packages.Package, 0, len(r.packages))
	for _, pkg := range r.packages {
		if pkg.Package.SoftwareIconObjectID != nil && *pkg.Package.SoftwareIconObjectID == iconObjectID {
			pkgs = append(pkgs, pkg.Package)
		}
	}
	return pkgs, nil
}

type staticHostResolver struct {
	serial string
}

func (r staticHostResolver) GetByHardwareSerial(_ context.Context, serial string) (*hosts.Host, error) {
	if serial != r.serial {
		return nil, dbutil.ErrNotFound
	}
	return &hosts.Host{
		ID:          1,
		DisplayName: "Test MacBook",
		Hardware:    hosts.HostHardware{Serial: serial},
	}, nil
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
		ID:                  id,
		SoftwareID:          softwareID,
		SoftwareName:        name,
		SoftwareDisplayName: name,
		Version:             version,
		InstallerType:       packages.InstallerTypeNoPkg,
		OnDemand:            true,
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
