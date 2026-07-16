package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki"
	munkiupload "github.com/woodleighschool/woodstar/internal/munki/objectupload"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestMunkiPackageInstallerFileLifecycle(t *testing.T) {
	fixture := newMunkiUploadFixture(t)

	t.Run("cancel pending upload", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "cancel.pkg")
		rec := fixture.request(
			t,
			http.MethodDelete,
			fmt.Sprintf("%s/%d", munkiPackageInstallerPath, target.ObjectID),
		)
		assertStatus(t, rec, http.StatusNoContent, "cancel installer")
		_, err := fixture.objects.GetByID(t.Context(), target.ObjectID)
		if !errors.Is(err, dbutil.ErrNotFound) {
			t.Fatalf("get cancelled object error = %v, want ErrNotFound", err)
		}
	})

	t.Run("missing upload bytes", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "missing.pkg")
		rec := fixture.request(t, http.MethodPut, fmt.Sprintf("%s/%d", munkiPackageInstallerPath, target.ObjectID))
		assertStatus(t, rec, http.StatusBadRequest, "missing upload")
		_, err := fixture.objects.GetByID(t.Context(), target.ObjectID)
		if !errors.Is(err, dbutil.ErrNotFound) {
			t.Fatalf("get cleaned missing upload error = %v, want ErrNotFound", err)
		}
	})

	t.Run("referenced object conflicts", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "claimed.pkg")
		fixture.upload(t, target, []byte("claimed installer"))
		path := fmt.Sprintf("%s/%d", munkiPackageInstallerPath, target.ObjectID)
		assertStatus(t, fixture.request(t, http.MethodPut, path), http.StatusOK, "finalize claimed installer")
		if _, err := fixture.packages.Create(t.Context(), packages.PackageCreateMutation{
			SoftwareID: fixture.softwareID,
			PackageMutation: packages.PackageMutation{
				Version:           "2.0",
				InstallerType:     packages.InstallerTypePkg,
				InstallerObjectID: &target.ObjectID,
			},
		}); err != nil {
			t.Fatalf("create package: %v", err)
		}
		assertStatus(
			t,
			fixture.request(t, http.MethodDelete, path),
			http.StatusConflict,
			"delete claimed installer",
		)
	})

	t.Run("delete rejects another object prefix", func(t *testing.T) {
		iconPath := fmt.Sprintf("/api/munki/software/%d/icon", fixture.softwareID)
		target := fixture.beginUpload(t, iconPath, "icon.png")
		path := fmt.Sprintf("%s/%d", munkiPackageInstallerPath, target.ObjectID)
		assertStatus(t, fixture.request(t, http.MethodDelete, path), http.StatusBadRequest, "delete icon as installer")
		if _, err := fixture.objects.GetByID(t.Context(), target.ObjectID); err != nil {
			t.Fatalf("get cross-prefix object: %v", err)
		}
	})

	t.Run("multipart is rejected by file storage", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "multipart.pkg")
		path := fmt.Sprintf("%s/%d/multipart", munkiPackageInstallerPath, target.ObjectID)
		assertStatus(
			t,
			fixture.request(t, http.MethodPost, path),
			http.StatusBadRequest,
			"create multipart upload",
		)
	})
}

func TestMunkiIconUploadLifecycleRemainsResourceScoped(t *testing.T) {
	fixture := newMunkiUploadFixture(t)
	path := fmt.Sprintf("/api/munki/software/%d/icon", fixture.softwareID)
	target := fixture.beginUpload(t, path, "icon.png")
	fixture.upload(t, target, []byte("\x89PNG\r\n\x1a\n"))

	rec := fixture.requestJSON(t, http.MethodPut, path, MunkiObjectMutation{ObjectID: target.ObjectID})
	assertStatus(t, rec, http.StatusOK, "attach icon")
	var view MunkiObjectView
	decodeJSON(t, rec, &view)
	if view.ID != target.ObjectID || view.ContentType != "image/png" {
		t.Fatalf("attached icon = %+v, want object %d as image/png", view, target.ObjectID)
	}
}

func TestMunkiUploadRejectsWrongPrefixAndInvalidIcon(t *testing.T) {
	fixture := newMunkiUploadFixture(t)

	t.Run("wrong object prefix", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "wrong-prefix.pkg")
		rec := fixture.requestJSON(
			t,
			http.MethodPut,
			fmt.Sprintf("/api/munki/software/%d/icon", fixture.softwareID),
			MunkiObjectMutation{ObjectID: target.ObjectID},
		)
		assertStatus(t, rec, http.StatusBadRequest, "wrong prefix")
	})

	t.Run("invalid icon content", func(t *testing.T) {
		path := fmt.Sprintf("/api/munki/software/%d/icon", fixture.softwareID)
		target := fixture.beginUpload(t, path, "not-an-icon.txt")
		fixture.upload(t, target, []byte("not an image"))
		rec := fixture.requestJSON(t, http.MethodPut, path, MunkiObjectMutation{ObjectID: target.ObjectID})
		assertStatus(t, rec, http.StatusBadRequest, "invalid icon")
		_, err := fixture.objects.GetByID(t.Context(), target.ObjectID)
		if !errors.Is(err, dbutil.ErrNotFound) {
			t.Fatalf("get cleaned invalid icon error = %v, want ErrNotFound", err)
		}
	})
}

type munkiUploadFixture struct {
	router     *chi.Mux
	objects    *storage.ObjectStore
	packages   *packages.Store
	softwareID int64
}

func newMunkiUploadFixture(t *testing.T) munkiUploadFixture {
	t.Helper()
	db, ctx := dbtest.Open(t)
	backend := newTestFileStore(t)
	objects := storage.NewObjectStore(db, backend)
	uploads := munkiupload.NewService(objects, backend)
	packageStore := packages.NewStore(db, objects, discardLogger())
	softwareStore := munkisoftware.NewStore(db, objects, packageStore)
	software, err := softwareStore.Create(ctx, munkisoftware.CreateMutation{Name: "Upload Test"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}

	router := chi.NewRouter()
	api := humachi.New(router, huma.DefaultConfig("test", "test"))
	registerPackageInstallerRoutes(api, objects, uploads, discardLogger())
	registerCreateMunkiPackage(api, munki.NewPackageService(munki.PackageServiceDependencies{
		Packages:               packageStore,
		DesiredPackagesChanged: func() {},
	}), discardLogger())
	registerIconRoutes(api, softwareStore, objects, uploads, discardLogger())
	registerCreateClientResourcesBannerUpload(api, uploads, discardLogger())
	registerDeleteClientResourcesBannerUpload(api, uploads, discardLogger())
	RegisterBlobStorage(router, backend, testCapabilityKey, discardLogger())

	return munkiUploadFixture{
		router:     router,
		objects:    objects,
		packages:   packageStore,
		softwareID: software.ID,
	}
}

func (f munkiUploadFixture) beginUpload(t *testing.T, path, filename string) MunkiUploadTarget {
	t.Helper()
	rec := f.requestJSON(t, http.MethodPost, path, MunkiUploadRequest{Filename: filename})
	assertStatus(t, rec, http.StatusCreated, "begin upload")
	var target MunkiUploadTarget
	decodeJSON(t, rec, &target)
	return target
}

func (f munkiUploadFixture) upload(t *testing.T, target MunkiUploadTarget, body []byte) {
	t.Helper()
	uploadURL, err := url.Parse(target.UploadURL)
	if err != nil {
		t.Fatalf("parse upload URL: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(target.Method, uploadURL.RequestURI(), bytes.NewReader(body))
	for name, value := range target.Headers {
		req.Header.Set(name, value)
	}
	f.router.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusNoContent, "upload")
}

func (f munkiUploadFixture) request(
	t *testing.T,
	method string,
	path string,
) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
	f.router.ServeHTTP(rec, req)
	return rec
}

func (f munkiUploadFixture) requestJSON(
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
	req := httptest.NewRequestWithContext(context.Background(), method, path, bytes.NewReader(payload))
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
