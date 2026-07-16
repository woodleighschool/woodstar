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

func TestMunkiUploadRoutes(t *testing.T) {
	fixture := newMunkiUploadFixture(t)

	t.Run("package installer", func(t *testing.T) {
		path := fmt.Sprintf("/api/munki/packages/%d/installer", fixture.packageID)
		target := fixture.beginUpload(t, path, "Installer.pkg")
		fixture.upload(t, target, []byte{0x00, 0x01, 0x02, 0x03})

		rec := fixture.requestJSON(
			t,
			http.MethodPut,
			path,
			MunkiObjectMutation{ObjectID: target.ObjectID},
		)
		assertStatus(t, rec, http.StatusOK, "attach package")
		var view MunkiObjectView
		decodeJSON(t, rec, &view)
		if view.ID != target.ObjectID || view.ContentType != "application/octet-stream" {
			t.Fatalf(
				"attached package = %+v, want object %d as application/octet-stream",
				view,
				target.ObjectID,
			)
		}
	})

	t.Run("software icon", func(t *testing.T) {
		path := fmt.Sprintf("/api/munki/software/%d/icon", fixture.softwareID)
		target := fixture.beginUpload(t, path, "icon.png")
		fixture.upload(t, target, []byte("\x89PNG\r\n\x1a\n"))

		rec := fixture.requestJSON(
			t,
			http.MethodPut,
			path,
			MunkiObjectMutation{ObjectID: target.ObjectID},
		)
		assertStatus(t, rec, http.StatusOK, "attach icon")
		var view MunkiObjectView
		decodeJSON(t, rec, &view)
		if view.ID != target.ObjectID || view.ContentType != "image/png" {
			t.Fatalf("attached icon = %+v, want object %d as image/png", view, target.ObjectID)
		}
	})

	t.Run("missing upload bytes", func(t *testing.T) {
		path := fmt.Sprintf("/api/munki/packages/%d/installer", fixture.packageID)
		target := fixture.beginUpload(t, path, "missing.pkg")
		rec := fixture.requestJSON(
			t,
			http.MethodPut,
			path,
			MunkiObjectMutation{ObjectID: target.ObjectID},
		)
		assertStatus(t, rec, http.StatusBadRequest, "missing upload")
		_, err := fixture.objects.GetByID(t.Context(), target.ObjectID)
		if !errors.Is(err, dbutil.ErrNotFound) {
			t.Fatalf("get cleaned missing upload error = %v, want %v", err, dbutil.ErrNotFound)
		}
	})

	t.Run("wrong object prefix", func(t *testing.T) {
		packagePath := fmt.Sprintf("/api/munki/packages/%d/installer", fixture.packageID)
		target := fixture.beginUpload(t, packagePath, "wrong-prefix.pkg")
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
		rec := fixture.requestJSON(
			t,
			http.MethodPut,
			path,
			MunkiObjectMutation{ObjectID: target.ObjectID},
		)
		assertStatus(t, rec, http.StatusBadRequest, "invalid icon")
		_, err := fixture.objects.GetByID(t.Context(), target.ObjectID)
		if !errors.Is(err, dbutil.ErrNotFound) {
			t.Fatalf("get cleaned invalid icon error = %v, want %v", err, dbutil.ErrNotFound)
		}
	})
}

type munkiUploadFixture struct {
	router     *chi.Mux
	objects    *storage.ObjectStore
	softwareID int64
	packageID  int64
}

func newMunkiUploadFixture(t *testing.T) munkiUploadFixture {
	t.Helper()
	db, ctx := dbtest.Open(t)
	backend := newTestFileStore(t)
	objects := storage.NewObjectStore(db, backend)
	uploads := munkiupload.NewService(objects, backend)
	packageStore := packages.NewStore(db, objects)
	softwareStore := munkisoftware.NewStore(db, objects, packageStore)
	software, err := softwareStore.Create(ctx, munkisoftware.CreateMutation{Name: "Upload Test"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	pkg, err := packageStore.Create(ctx, packages.PackageCreateMutation{
		SoftwareID: software.ID,
		PackageMutation: packages.PackageMutation{
			Version:       "1.0",
			InstallerType: packages.InstallerTypePkg,
			Eligible:      true,
		},
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	packageService := munki.NewPackageService(munki.PackageServiceDependencies{
		Packages:               packageStore,
		DesiredPackagesChanged: func() {},
	})

	router := chi.NewRouter()
	api := humachi.New(router, huma.DefaultConfig("test", "test"))
	registerObjectRoutes(api, packageService, objects, uploads, discardLogger())
	registerIconRoutes(api, softwareStore, objects, uploads, discardLogger())
	RegisterBlobStorage(router, backend, testCapabilityKey, discardLogger())

	return munkiUploadFixture{
		router:     router,
		objects:    objects,
		softwareID: software.ID,
		packageID:  pkg.ID,
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

func assertStatus(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	want int,
	operation string,
) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf(
			"%s status = %d, want %d; body = %q",
			operation,
			rec.Code,
			want,
			rec.Body.String(),
		)
	}
}
