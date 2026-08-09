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
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"howett.net/plist"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
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
	req.Header.Set(testSerialHeader, "C02MUNKI")

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
	NewServer(nil, nil, nil, nil, nil, nil, testLogger()).RegisterRoutes(ordinary, transfers)

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
	resolver := staticPackageResolver{packages: []munkisoftware.EffectivePackage{
		{
			Actions: []munkisoftware.Action{munkisoftware.ActionManagedInstalls},
			Selector: munkisoftware.PackageSelector{
				Strategy: munkisoftware.PackageLatest,
			},
			Package: staticMunkiPackage(20, "ExternalURLApp", "1.0"),
		},
	}}
	service := munki.NewRepositoryService(munki.Dependencies{
		Software: resolver,
		Packages: resolver,
	})

	body, err := service.Catalog(context.Background(), 42, "woodstar")
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
	manifestReq.Header.Set(testSerialHeader, "C02MUNKI")
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
	catalogReq.Header.Set(testSerialHeader, "C02MUNKI")
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
	req.Header.Set(testSerialHeader, "C02MUNKI")

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

func TestMunkiHTTPBindsEveryRepositoryRouteToAuthenticatedHost(t *testing.T) {
	routes := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "manifest", path: "/munki/manifests/C02MUNKI", wantStatus: http.StatusOK},
		{name: "catalog", path: "/munki/catalogs/woodstar", wantStatus: http.StatusOK},
		{name: "icon hashes", path: "/munki/icons/_icon_hashes.plist", wantStatus: http.StatusOK},
		{name: "package", path: "/munki/pkgs/packages/20/installer/GoogleChrome.pkg", wantStatus: http.StatusFound},
		{name: "icon", path: "/munki/icons/7-GoogleChrome.png", wantStatus: http.StatusFound},
		{name: "client resources", path: "/munki/client_resources/site_default.zip", wantStatus: http.StatusFound},
	}
	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			for _, tc := range []struct {
				name          string
				authorization string
				serial        string
				resolver      staticHostResolver
				wantStatus    int
				wantCalls     int
			}{
				{
					name:       "missing bearer authenticates before resolution",
					resolver:   staticHostResolver{err: errors.New("resolver should not run")},
					wantStatus: http.StatusUnauthorized,
				},
				{
					name:          "bad bearer authenticates before resolution",
					authorization: "Bearer bad-secret",
					resolver:      staticHostResolver{err: errors.New("resolver should not run")},
					wantStatus:    http.StatusUnauthorized,
				},
				{
					name:          "missing serial",
					authorization: "Bearer munki-secret",
					resolver:      testHostResolver(),
					wantStatus:    http.StatusNotFound,
				},
				{
					name:          "unknown serial",
					authorization: "Bearer munki-secret",
					serial:        "C02UNKNOWN",
					resolver:      testHostResolver(),
					wantStatus:    http.StatusNotFound,
				},
				{
					name:          "resolver failure",
					authorization: "Bearer munki-secret",
					serial:        "C02MUNKI",
					resolver:      staticHostResolver{err: errors.New("resolver failed")},
					wantStatus:    http.StatusInternalServerError,
				},
				{
					name:          "resolved host",
					authorization: "Bearer munki-secret",
					serial:        " C02MUNKI ",
					resolver:      testHostResolver(),
					wantStatus:    route.wantStatus,
					wantCalls:     1,
				},
			} {
				t.Run(tc.name, func(t *testing.T) {
					repository := newStaticRepository()
					router := chi.NewRouter()
					NewServer(
						staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
						tc.resolver,
						repository,
						&recordingHeartbeatRecorder{},
						&fakeSelector{},
						&fakeDeliverer{url: "https://storage.example/file"},
						testLogger(),
					).RegisterRoutes(router, router)
					recorder := httptest.NewRecorder()
					request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, route.path, nil)
					if tc.authorization != "" {
						request.Header.Set("Authorization", tc.authorization)
					}
					if tc.serial != "" {
						request.Header.Set(testSerialHeader, tc.serial)
					}

					router.ServeHTTP(recorder, request)

					if recorder.Code != tc.wantStatus {
						t.Fatalf("status = %d, want %d; body = %q", recorder.Code, tc.wantStatus, recorder.Body.String())
					}
					if repository.calls != tc.wantCalls {
						t.Fatalf("repository calls = %d, want %d", repository.calls, tc.wantCalls)
					}
					if tc.wantCalls == 1 && repository.hostID != 42 {
						t.Fatalf("repository host ID = %d, want 42", repository.hostID)
					}
				})
			}
		})
	}
}

func TestMunkiHTTPRecordsKnownHostContactBeforeRepositoryDispatch(t *testing.T) {
	routes := []struct {
		name string
		path string
	}{
		{name: "manifest", path: "/munki/manifests/C02MUNKI"},
		{name: "catalog", path: "/munki/catalogs/woodstar"},
		{name: "icon hashes", path: "/munki/icons/_icon_hashes.plist"},
		{name: "package", path: "/munki/pkgs/packages/20/installer/GoogleChrome.pkg"},
		{name: "icon", path: "/munki/icons/7-GoogleChrome.png"},
		{name: "client resources", path: "/munki/client_resources/site_default.zip"},
	}

	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			repository := newStaticRepository()
			repository.err = munki.ErrNotFound
			recorder := &recordingHeartbeatRecorder{}
			router := chi.NewRouter()
			router.Use(chimiddleware.ClientIPFromRemoteAddr)
			NewServer(
				staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
				testHostResolver(),
				repository,
				recorder,
				&fakeSelector{},
				&fakeDeliverer{},
				testLogger(),
			).RegisterRoutes(router, router)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, route.path, nil)
			req.RemoteAddr = "203.0.113.9:12345"
			req.Header.Set("Authorization", "Bearer munki-secret")
			req.Header.Set(testSerialHeader, "C02MUNKI")
			req.Header.Set("User-Agent", "ManagedInstall/7.1")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusNotFound, rec.Body.String())
			}
			if repository.calls != 1 {
				t.Fatalf("repository calls = %d, want 1", repository.calls)
			}
			if got := recorder.records; !slices.Equal(got, []heartbeatRecord{{
				hostID:  42,
				source:  heartbeats.SourceMunki,
				contact: heartbeats.Contact{RemoteIP: "203.0.113.9", UserAgent: "ManagedInstall/7.1"},
			}}) {
				t.Fatalf("records = %+v, want Munki contact", got)
			}
		})
	}
}

func TestMunkiHTTPDoesNotRecordUnauthorizedOrUnknownHosts(t *testing.T) {
	for _, tc := range []struct {
		name          string
		authorization string
		serial        string
		wantStatus    int
	}{
		{name: "missing bearer", wantStatus: http.StatusUnauthorized},
		{name: "unknown serial", authorization: "Bearer munki-secret", serial: "C02UNKNOWN", wantStatus: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &recordingHeartbeatRecorder{}
			router := chi.NewRouter()
			NewServer(
				staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
				testHostResolver(),
				newStaticRepository(),
				recorder,
				&fakeSelector{},
				&fakeDeliverer{},
				testLogger(),
			).RegisterRoutes(router, router)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/munki/manifests/C02MUNKI", nil)
			req.Header.Set("Authorization", tc.authorization)
			req.Header.Set(testSerialHeader, tc.serial)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if len(recorder.records) != 0 {
				t.Fatalf("records = %+v, want none", recorder.records)
			}
		})
	}
}

func TestMunkiHTTPMapsRecorderErrorsToServerErrors(t *testing.T) {
	repository := newStaticRepository()
	router := chi.NewRouter()
	NewServer(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		testHostResolver(),
		repository,
		&recordingHeartbeatRecorder{err: errors.New("record heartbeat")},
		&fakeSelector{},
		&fakeDeliverer{},
		testLogger(),
	).RegisterRoutes(router, router)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/munki/manifests/C02MUNKI", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set(testSerialHeader, "C02MUNKI")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if repository.calls != 0 {
		t.Fatalf("repository calls = %d, want 0", repository.calls)
	}
}

func TestMunkiHTTPRejectsConditionalRequestWithoutSerial(t *testing.T) {
	repository := newStaticRepository()
	router := chi.NewRouter()
	NewServer(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		testHostResolver(),
		repository,
		&recordingHeartbeatRecorder{},
		&fakeSelector{},
		&fakeDeliverer{url: "https://storage.example/file"},
		testLogger(),
	).RegisterRoutes(router, router)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/munki/manifests/C02MUNKI", nil)
	request.Header.Set("Authorization", "Bearer munki-secret")
	request.Header.Set("If-None-Match", "*")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %q", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if repository.calls != 0 {
		t.Fatalf("repository calls = %d, want 0", repository.calls)
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
		testHostResolver(),
		repository,
		&recordingHeartbeatRecorder{},
		selector,
		delivery,
		testLogger(),
	).RegisterRoutes(router, router)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/munki/pkgs/packages/20/installer/GoogleChrome.pkg", nil)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set(testSerialHeader, "C02MUNKI")
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

func TestMunkiHTTPDecodesPackageLocationBeforeResolution(t *testing.T) {
	repository := newStaticRepository()
	repository.packageID = 38
	sha256sum := strings.Repeat("a", 64)
	sizeBytes := int64(4096)
	repository.fileObject = storage.Object{
		ID:        81,
		Prefix:    "munki/packages",
		Filename:  "Zoom-7.1.5 (84650).pkg",
		SHA256:    &sha256sum,
		SizeBytes: &sizeBytes,
	}
	selector := &fakeSelector{
		url: "https://mdp.example/munki/pkgs/packages/38/installer/Zoom-7.1.5%20%2884650%29.pkg?cap=grant",
		ok:  true,
	}
	router := chi.NewRouter()
	NewServer(
		staticVerifier{agent: agentauth.AgentMunki, token: "munki-secret"},
		testHostResolver(),
		repository,
		&recordingHeartbeatRecorder{},
		selector,
		&fakeDeliverer{},
		testLogger(),
	).RegisterRoutes(router, router)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/munki/pkgs/packages/38/installer/Zoom-7.1.5%20(84650).pkg",
		nil,
	)
	req.Header.Set("Authorization", "Bearer munki-secret")
	req.Header.Set(testSerialHeader, "C02MUNKI")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusFound, rec.Body.String())
	}
	const want = "packages/38/installer/Zoom-7.1.5 (84650).pkg"
	if repository.fileKey != want {
		t.Fatalf("repository package location = %q, want %q", repository.fileKey, want)
	}
	if selector.got.InstallerItemLocation != want {
		t.Fatalf(
			"distribution package location = %q, want %q",
			selector.got.InstallerItemLocation,
			want,
		)
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
	req.Header.Set(testSerialHeader, "C02MUNKI")

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
	NewServer(verifier, testHostResolver(), repository, &recordingHeartbeatRecorder{}, &fakeSelector{}, d, testLogger()).RegisterRoutes(r, r)
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
	service    *munki.RepositoryService
	hostID     int64
	calls      int
	fileErr    error
	fileClass  string
	fileKey    string
	packageID  int64
	fileObject storage.Object
	err        error
}

func newStaticRepository() *staticRepository {
	return newStaticRepositoryWithPackages(nil)
}

func newStaticRepositoryWithPackages(packages []munkisoftware.EffectivePackage) *staticRepository {
	return &staticRepository{
		service: munki.NewRepositoryService(munki.Dependencies{
			Software: staticPackageResolver{packages: packages},
			Packages: staticPackageResolver{packages: packages},
		}),
	}
}

func (r *staticRepository) Manifest(ctx context.Context, hostID int64) ([]byte, error) {
	r.calls++
	r.hostID = hostID
	if r.err != nil {
		return nil, r.err
	}
	return r.service.Manifest(ctx, hostID)
}

func (r *staticRepository) Catalog(ctx context.Context, hostID int64, name string) ([]byte, error) {
	r.calls++
	r.hostID = hostID
	if r.err != nil {
		return nil, r.err
	}
	return r.service.Catalog(ctx, hostID, name)
}

func (r *staticRepository) IconHashes(ctx context.Context, hostID int64) ([]byte, error) {
	r.calls++
	r.hostID = hostID
	if r.err != nil {
		return nil, r.err
	}
	return r.service.IconHashes(ctx, hostID)
}

func (r *staticRepository) ResolvePackageFile(
	_ context.Context,
	hostID int64,
	key string,
) (munki.PackageInstaller, error) {
	r.calls++
	r.hostID = hostID
	r.fileClass = "package"
	r.fileKey = key
	if r.err != nil {
		return munki.PackageInstaller{}, r.err
	}
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
	hostID int64,
	key string,
) (storage.Object, error) {
	r.calls++
	r.hostID = hostID
	return r.resolve("icon", key)
}

func (r *staticRepository) ResolveClientResources(
	_ context.Context,
	hostID int64,
	name string,
) (storage.Object, error) {
	r.calls++
	r.hostID = hostID
	return r.resolve("client resources", name)
}

func (r *staticRepository) resolve(class, key string) (storage.Object, error) {
	r.fileClass = class
	r.fileKey = key
	if r.err != nil {
		return storage.Object{}, r.err
	}
	if r.fileErr != nil {
		return storage.Object{}, r.fileErr
	}
	return r.fileObject, nil
}

type heartbeatRecord struct {
	hostID  int64
	source  heartbeats.Source
	contact heartbeats.Contact
}

type recordingHeartbeatRecorder struct {
	records []heartbeatRecord
	err     error
}

func (r *recordingHeartbeatRecorder) Record(
	_ context.Context,
	hostID int64,
	source heartbeats.Source,
	contact heartbeats.Contact,
) error {
	r.records = append(r.records, heartbeatRecord{hostID: hostID, source: source, contact: contact})
	return r.err
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

type staticHostResolver struct {
	serial string
	hostID int64
	err    error
}

func (r staticHostResolver) GetByHardwareSerial(_ context.Context, serial string) (*hosts.Host, error) {
	if r.err != nil {
		return nil, r.err
	}
	if serial != r.serial {
		return nil, fault.ErrNotFound
	}
	return &hosts.Host{
		ID:          r.hostID,
		DisplayName: "Test MacBook",
		Hardware:    hosts.HostHardware{Serial: serial},
	}, nil
}

const testSerialHeader = "X-Woodstar-Serial-Number"

func testHostResolver() staticHostResolver {
	return staticHostResolver{serial: "C02MUNKI", hostID: 42}
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
