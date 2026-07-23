package protocol

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

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

func TestMunkiHTTPServesIconHashIndex(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepository(),
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/munki/icons/_icon_hashes.plist", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != plistContentType {
		t.Fatalf("Content-Type = %q, want %q", got, plistContentType)
	}
	var hashes map[string]string
	if _, err := plist.Unmarshal(rec.Body.Bytes(), &hashes); err != nil {
		t.Fatalf("icon hashes plist: %v", err)
	}
	if len(hashes) != 0 {
		t.Fatalf("icon hashes = %v, want empty", hashes)
	}
}

func TestRegisterRoutesSelectsTransferSurface(t *testing.T) {
	t.Parallel()
	router := chi.NewRouter()
	ordinary := router.With(testRouteSurface("ordinary"))
	transfers := router.With(testRouteSurface("transfer"))
	NewServer(nil, nil, nil, nil, testLogger()).RegisterRoutes(ordinary, transfers)

	for _, tc := range []struct {
		path        string
		wantSurface string
	}{
		{path: "/munki/manifests/site_default", wantSurface: "ordinary"},
		{path: "/munki/catalogs/production", wantSurface: "ordinary"},
		{path: "/munki/icons/_icon_hashes.plist", wantSurface: "ordinary"},
		{path: "/munki/pkgs/Installer.pkg", wantSurface: "transfer"},
		{path: "/munki/icons/App.png", wantSurface: "transfer"},
		{path: "/munki/client_resources/site.zip", wantSurface: "transfer"},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil))
		if got := recorder.Header().Get("X-Route-Surface"); got != tc.wantSurface {
			t.Errorf("%s route surface = %q, want %q", tc.path, got, tc.wantSurface)
		}
	}
}

func testRouteSurface(surface string) func(http.Handler) http.Handler {
	return func(_ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Route-Surface", surface)
			w.WriteHeader(http.StatusNoContent)
		})
	}
}

func TestMunkiCatalogNoPkgOmitsInstallerFields(t *testing.T) {
	service := munki.NewRepositoryService(munki.Dependencies{
		Packages: staticPackageResolver{packages: []munkisoftware.EffectivePackage{
			{
				Actions: []munkisoftware.Action{munkisoftware.ActionManagedInstalls},
				Selector: munkisoftware.PackageSelector{
					Strategy: munkisoftware.PackageLatest,
				},
				Package: staticMunkiPackage(20, "ExternalURLApp", "1.0"),
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
	for _, key := range []string{"installer_item_location", "installer_item_hash", "installer_item_size"} {
		if _, ok := decoded[0][key]; ok {
			t.Fatalf("nopkg rendered %s: %+v", key, decoded[0])
		}
	}
}

func TestMunkiHTTPRendersLatestSoftwareIDOnceWithAllPkginfos(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepositoryWithPackages([]munkisoftware.EffectivePackage{
			{
				Actions: []munkisoftware.Action{munkisoftware.ActionOptionalInstalls},
				Selector: munkisoftware.PackageSelector{
					Strategy: munkisoftware.PackageLatest,
				},
				Package: staticMunkiPackage(20, "GoogleChrome", "148.0.0.1"),
			},
			{
				Actions: []munkisoftware.Action{munkisoftware.ActionOptionalInstalls},
				Selector: munkisoftware.PackageSelector{
					Strategy: munkisoftware.PackageLatest,
				},
				Package: staticMunkiPackage(21, "GoogleChrome", "149.0.0.1"),
			},
		}),
	)

	manifest := httptest.NewRecorder()
	manifestReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/munki/manifests/C02MUNKI", nil)
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
	if !slices.Equal(manifestBody.OptionalInstalls, []string{"GoogleChrome"}) {
		t.Fatalf("optional_installs = %v, want [GoogleChrome]", manifestBody.OptionalInstalls)
	}

	catalog := httptest.NewRecorder()
	catalogReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/munki/catalogs/woodstar", nil)
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
		if item["name"] != "GoogleChrome" {
			t.Fatalf("catalog item name = %v, want GoogleChrome: %+v", item["name"], item)
		}
	}
}

func TestMunkiHTTPRendersPinnedPackageName(t *testing.T) {
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		newStaticRepositoryWithPackages([]munkisoftware.EffectivePackage{
			{
				Actions: []munkisoftware.Action{munkisoftware.ActionManagedInstalls},
				Selector: munkisoftware.PackageSelector{
					Strategy: munkisoftware.PackageSpecific,
				},
				Package: staticMunkiPackage(20, "PinnedApp", "1.0"),
			},
		}),
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/munki/manifests/C02MUNKI", nil)
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
	if !slices.Equal(decoded.ManagedInstalls, []string{"PinnedApp--1.0"}) {
		t.Fatalf("managed_installs = %v, want [PinnedApp--1.0]", decoded.ManagedInstalls)
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/munki/manifests/C02MUNKI", nil)
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
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			req.Header.Set("Authorization", "Bearer munki-secret")

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusNotFound, rec.Body.String())
			}
		})
	}
}

func TestMunkiHTTPRedirectsPackageFileToDistributionPoint(t *testing.T) {
	repository := newStaticRepository()
	repository.packageID = 20
	sha256sum := strings.Repeat("a", 64)
	sizeBytes := int64(4096)
	repository.fileObject = storage.Object{
		ID:          42,
		Prefix:      "munki/packages",
		Filename:    "GoogleChrome.pkg",
		ContentType: "application/octet-stream",
		SHA256:      &sha256sum,
		SizeBytes:   &sizeBytes,
	}
	delivery := &fakeDeliverer{url: "https://storage.example/direct"}
	selector := &fakeSelector{
		url: "https://mdp.example/munki/pkgs/packages/20/installer/GoogleChrome.pkg?cap=grant",
		ok:  true,
	}

	router := chi.NewRouter()
	NewServer(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		repository,
		selector,
		delivery,
		testLogger(),
	).RegisterRoutes(router, router)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/munki/pkgs/packages/20/installer/GoogleChrome.pkg", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusFound, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != selector.url {
		t.Fatalf("Location = %q, want distribution point URL %q", got, selector.url)
	}
	if selector.got.PackageID != 20 || selector.got.SHA256 != sha256sum || selector.got.SizeBytes != 4096 {
		t.Fatalf("selection integrity claims = %+v", selector.got)
	}
	if selector.got.InstallerItemLocation != "packages/20/installer/GoogleChrome.pkg" {
		t.Fatalf("selection installer_item_location = %q", selector.got.InstallerItemLocation)
	}
	if delivery.gotObject.ID != 0 {
		t.Fatalf("Woodstar delivery should be skipped, got object %+v", delivery.gotObject)
	}
}

func TestMunkiHTTPDeliversIconFileWithNestedIconName(t *testing.T) {
	repository := newStaticRepository()
	repository.fileObject = storage.Object{
		ID:          7,
		Prefix:      "munki/icons",
		Filename:    "GoogleChrome.png",
		ContentType: "image/png",
	}
	delivery := &fakeDeliverer{url: "https://storage.example/icon.png?signature=test"}
	router := newMunkiContractRouter(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		repository,
		delivery,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/munki/icons/7-GoogleChrome.png", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusFound, rec.Body.String())
	}
	if repository.fileClass != "icon" ||
		repository.fileKey != "7-GoogleChrome.png" {
		t.Fatalf("file request = class %q key %q", repository.fileClass, repository.fileKey)
	}
	if delivery.gotObject.Key() != "munki/icons/7/GoogleChrome.png" {
		t.Fatalf("delivered object = %+v", delivery.gotObject)
	}
	if delivery.gotObject.ContentType != "image/png" {
		t.Fatalf("delivered content type = %q", delivery.gotObject.ContentType)
	}
}

func TestMunkiHTTPMapsVerifierErrorsToServerErrors(t *testing.T) {
	router := newMunkiContractRouter(errorVerifier{}, newStaticRepository())
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/munki/catalogs/woodstar", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func newMunkiContractRouter(
	verifier agentauth.SecretVerifier,
	repository Repository,
	delivery ...storage.Deliverer,
) chi.Router {
	var d storage.Deliverer
	if len(delivery) > 0 {
		d = delivery[0]
	}
	r := chi.NewRouter()
	NewServer(verifier, repository, &fakeSelector{}, d, testLogger()).RegisterRoutes(r, r)
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

type fakeDeliverer struct {
	url        string
	gotObject  storage.Object
	gotOptions storage.DeliveryOptions
}

func (f *fakeDeliverer) Deliver(
	w http.ResponseWriter,
	r *http.Request,
	object storage.Object,
	opts storage.DeliveryOptions,
) error {
	f.gotObject = object
	f.gotOptions = opts
	http.Redirect(w, r, f.url, http.StatusFound)
	return nil
}

type staticVerifier struct {
	agent agentauth.Agent
	token string
}

func (v staticVerifier) Verify(_ context.Context, agent agentauth.Agent, token string) (bool, error) {
	return agent == v.agent && token == v.token, nil
}

type errorVerifier struct{}

func (errorVerifier) Verify(context.Context, agentauth.Agent, string) (bool, error) {
	return false, errors.New("verifier failed")
}

type staticRepository struct {
	service      *munki.RepositoryService
	manifestName string
	fileErr      error
	fileClass    string
	fileKey      string
	packageID    int64
	fileObject   storage.Object
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

func (r *staticRepository) IconHashes(ctx context.Context) ([]byte, error) {
	return r.service.IconHashes(ctx)
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
		Object:                r.fileObject,
	}
	return installer, nil
}

func (r *staticRepository) ResolveIconFile(
	_ context.Context,
	key string,
) (storage.Object, error) {
	return r.resolve("icon", key)
}

func (r *staticRepository) ResolveClientResources(
	_ context.Context,
	name string,
) (storage.Object, error) {
	return r.resolve("client resources", name)
}

func (r *staticRepository) resolve(class, key string) (storage.Object, error) {
	r.fileClass = class
	r.fileKey = key
	if r.fileErr != nil {
		return storage.Object{}, r.fileErr
	}
	return r.fileObject, nil
}

type staticPackageResolver struct {
	packages []munkisoftware.EffectivePackage
}

func (r staticPackageResolver) EffectivePackagesForHost(
	_ context.Context,
	_ int64,
) ([]munkisoftware.EffectivePackage, error) {
	return r.packages, nil
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

func (r staticPackageResolver) ListRepositoryIconObjectIDs(context.Context) ([]int64, error) {
	ids := make([]int64, 0, len(r.packages))
	seen := make(map[int64]struct{}, len(r.packages))
	for _, pkg := range r.packages {
		if pkg.Package.Software.IconObjectID == nil {
			continue
		}
		id := *pkg.Package.Software.IconObjectID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
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
		if pkg.Package.Software.IconObjectID != nil && *pkg.Package.Software.IconObjectID == iconObjectID {
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

func staticMunkiPackage(id int64, name string, version string) packages.Package {
	return packages.Package{
		ID:            id,
		Software:      packages.PackageSoftware{ID: 1, Name: name},
		Version:       version,
		InstallerType: packages.InstallerTypeNoPkg,
		OnDemand:      true,
	}
}
