//go:build postgres

package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/goodies/bloby"

	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/testutil/testbloby"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

type munkiFixture struct {
	db         *pgxpool.Pool
	router     *chi.Mux
	objects    *bloby.Service
	packages   *packages.Store
	softwareID int64
}

func newMunkiFixture(t *testing.T) munkiFixture {
	t.Helper()
	db, ctx := testdb.Open(t)
	objects := testbloby.New(t, db)
	packageStore := packages.NewStore(db, objects)
	softwareStore := munkisoftware.NewStore(db, objects, packageStore)
	software, err := softwareStore.Create(ctx, munkisoftware.CreateMutation{Name: "ExampleApp"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}

	router := chi.NewRouter()
	humaAPI := humachi.New(router, testHumaConfigWithoutUtilityRoutes())
	service := munki.NewPackageService(munki.PackageServiceDependencies{
		Packages: packageStore, DesiredPackagesChanged: func() {},
	})
	deletions := munki.NewSoftwareDeletionService(softwareStore, func() {})
	registerMunkiPackages(humaAPI, humaAPI, service, objects, discardLogger())
	registerMunkiSoftware(humaAPI, softwareStore, deletions, service, objects, discardLogger())
	registerCreateClientResourcesUpload(
		humaAPI,
		objects,
		discardLogger(),
		clientResourcesBannerUploadPath,
		clientresources.BannerObjectPrefix,
		"create-test-client-resources-banner-upload",
		"Create a banner upload",
	)
	registerDeleteClientResourcesUpload(
		humaAPI,
		objects,
		discardLogger(),
		clientResourcesBannerUploadPath,
		clientresources.BannerObjectPrefix,
		"delete-test-client-resources-banner-upload",
		"Delete a banner upload",
	)
	registerCreateClientResourcesUpload(
		humaAPI,
		objects,
		discardLogger(),
		clientResourcesArchiveUploadPath,
		clientresources.ArchiveObjectPrefix,
		"create-test-client-resources-archive-upload",
		"Create an archive upload",
	)
	registerDeleteClientResourcesUpload(
		humaAPI,
		objects,
		discardLogger(),
		clientResourcesArchiveUploadPath,
		clientresources.ArchiveObjectPrefix,
		"delete-test-client-resources-archive-upload",
		"Delete an archive upload",
	)
	registerMunkiContentRoutes(router, router, router, objects, discardLogger())
	router.Handle("/storage/*", objects.TransferHandler())

	return munkiFixture{
		db:         db,
		router:     router,
		objects:    objects,
		packages:   packageStore,
		softwareID: software.ID,
	}
}

func (f munkiFixture) beginUpload(t *testing.T, path, filename string) MunkiUploadTarget {
	t.Helper()
	var request any = MunkiDirectUploadRequest{Filename: filename}
	if path == munkiPackageInstallerPath {
		request = MunkiPackageInstallerUploadRequest{Filename: filename, SizeBytes: 0}
	}
	rec := f.requestJSON(t, http.MethodPost, path, request)
	assertStatus(t, rec, http.StatusCreated, "begin upload")
	var target MunkiUploadTarget
	decodeJSON(t, rec, &target)
	if target.Upload.Strategy != bloby.StrategyDirectPut {
		t.Fatalf("upload strategy = %q, want direct-put", target.Upload.Strategy)
	}
	return target
}

func (f munkiFixture) upload(t *testing.T, target MunkiUploadTarget, body []byte) {
	t.Helper()
	uploadURL, err := url.Parse(target.Upload.Target.URL)
	if err != nil {
		t.Fatalf("parse upload URL: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), target.Upload.Target.Method, uploadURL.RequestURI(), bytes.NewReader(body))
	for name, value := range target.Upload.Target.Headers {
		req.Header.Set(name, value)
	}
	f.router.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusNoContent, "upload")
}

func (f munkiFixture) request(
	t *testing.T,
	method string,
	path string,
) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), method, path, nil)
	f.router.ServeHTTP(rec, req)
	return rec
}

func (f munkiFixture) requestJSON(
	t *testing.T,
	method string,
	path string,
	body any,
) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode request body: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(rec, req)
	return rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response: %v; body = %q", err, rec.Body.String())
	}
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int, operation string) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("%s status = %d, want %d; body = %q", operation, rec.Code, want, rec.Body.String())
	}
}

func assertProblem(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	assertStatus(t, rec, want, "invalid mutation")
	var problem struct {
		Status int `json:"status"`
	}
	decodeJSON(t, rec, &problem)
	if problem.Status != want {
		t.Fatalf("problem status = %d, want %d; body = %s", problem.Status, want, rec.Body)
	}
}
