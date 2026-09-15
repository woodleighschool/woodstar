//go:build postgres

package httpapi

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/woodleighschool/goodies/bloby"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

func TestMunkiPackageInstallerFileLifecycle(t *testing.T) {
	fixture := newMunkiFixture(t)

	t.Run("cancel pending upload", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "cancel.pkg")
		rec := fixture.request(
			t,
			http.MethodDelete,
			fmt.Sprintf("%s/%d", munkiPackageInstallerPath, target.ObjectID),
		)
		assertStatus(t, rec, http.StatusNoContent, "cancel installer")
		_, err := fixture.objects.GetByID(t.Context(), target.ObjectID)
		if !errors.Is(err, bloby.ErrNotFound) {
			t.Fatalf("get cancelled object error = %v, want ErrNotFound", err)
		}
	})

	t.Run("missing upload bytes", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "missing.pkg")
		rec := fixture.request(t, http.MethodPut, fmt.Sprintf("%s/%d", munkiPackageInstallerPath, target.ObjectID))
		assertStatus(t, rec, http.StatusBadRequest, "missing upload")
		_, err := fixture.objects.GetByID(t.Context(), target.ObjectID)
		if !errors.Is(err, bloby.ErrNotFound) {
			t.Fatalf("get cleaned missing upload error = %v, want ErrNotFound", err)
		}
	})

	t.Run("referenced object conflicts", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "claimed.pkg")
		fixture.upload(t, target, []byte("claimed installer"))
		path := fmt.Sprintf("%s/%d", munkiPackageInstallerPath, target.ObjectID)
		assertStatus(t, fixture.request(t, http.MethodPut, path), http.StatusOK, "finalize claimed installer")
		if _, err := fixture.packages.Create(t.Context(), packages.PackageCreateMutation{
			SoftwareID:        fixture.softwareID,
			Version:           "2.0",
			InstallerType:     packages.InstallerTypePkg,
			InstallerObjectID: &target.ObjectID,
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
		target := fixture.beginUpload(t, munkiIconPath, "icon.png")
		path := fmt.Sprintf("%s/%d", munkiPackageInstallerPath, target.ObjectID)
		assertStatus(t, fixture.request(t, http.MethodDelete, path), http.StatusBadRequest, "delete icon as installer")
		if _, err := fixture.objects.GetByID(t.Context(), target.ObjectID); err != nil {
			t.Fatalf("get cross-prefix object: %v", err)
		}
	})

	t.Run("multipart is rejected by file storage", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "multipart.pkg")
		path := fmt.Sprintf("%s/%d/multipart/parts/1", munkiPackageInstallerPath, target.ObjectID)
		assertStatus(
			t,
			fixture.request(t, http.MethodPost, path),
			http.StatusBadRequest,
			"sign multipart part",
		)
	})
}

func TestMunkiIconUploadLifecycle(t *testing.T) {
	fixture := newMunkiFixture(t)
	attachPath := fmt.Sprintf("/api/munki/software/%d/icon", fixture.softwareID)
	icon := []byte("\x89PNG\r\n\x1a\n")
	target := fixture.beginUpload(t, munkiIconPath, "icon.png")
	fixture.upload(t, target, icon)

	rec := fixture.requestJSON(t, http.MethodPut, attachPath, MunkiObjectMutation{ObjectID: target.ObjectID})
	assertStatus(t, rec, http.StatusOK, "attach icon")
	var view MunkiObjectView
	decodeJSON(t, rec, &view)
	wantContentURL := fmt.Sprintf("/api/munki/icons/%d/content", target.ObjectID)
	if view.ID != target.ObjectID || view.ContentType != "image/png" || view.ContentURL != wantContentURL {
		t.Fatalf("attached icon = %+v, want object %d as image/png", view, target.ObjectID)
	}

	content := fixture.request(t, http.MethodGet, view.ContentURL)
	assertStatus(t, content, http.StatusOK, "get attached icon content")
	if !bytes.Equal(content.Body.Bytes(), icon) {
		t.Fatalf("icon content = %q, want uploaded bytes %q", content.Body.Bytes(), icon)
	}
	if got := content.Header().Get("Cache-Control"); got != munkiAssetCacheControl {
		t.Fatalf("icon Cache-Control = %q, want %q", got, munkiAssetCacheControl)
	}
}

func TestMunkiUploadRejectsWrongPrefixAndInvalidIcon(t *testing.T) {
	fixture := newMunkiFixture(t)

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
		attachPath := fmt.Sprintf("/api/munki/software/%d/icon", fixture.softwareID)
		target := fixture.beginUpload(t, munkiIconPath, "not-an-icon.txt")
		fixture.upload(t, target, []byte("not an image"))
		rec := fixture.requestJSON(t, http.MethodPut, attachPath, MunkiObjectMutation{ObjectID: target.ObjectID})
		assertStatus(t, rec, http.StatusBadRequest, "invalid icon")
		_, err := fixture.objects.GetByID(t.Context(), target.ObjectID)
		if !errors.Is(err, bloby.ErrNotFound) {
			t.Fatalf("get cleaned invalid icon error = %v, want ErrNotFound", err)
		}
	})
}

func TestClientResourcesUploadsRemainPrefixScoped(t *testing.T) {
	fixture := newMunkiFixture(t)
	banner := fixture.beginUpload(t, clientResourcesBannerUploadPath, "banner.png")
	archive := fixture.beginUpload(t, clientResourcesArchiveUploadPath, "resources.zip")

	wrongArchivePath := fmt.Sprintf("%s/%d", clientResourcesArchiveUploadPath, banner.ObjectID)
	assertStatus(
		t,
		fixture.request(t, http.MethodDelete, wrongArchivePath),
		http.StatusBadRequest,
		"delete banner as archive",
	)
	if _, err := fixture.objects.GetByID(t.Context(), banner.ObjectID); err != nil {
		t.Fatalf("get cross-prefix banner: %v", err)
	}

	archivePath := fmt.Sprintf("%s/%d", clientResourcesArchiveUploadPath, archive.ObjectID)
	assertStatus(
		t,
		fixture.request(t, http.MethodDelete, archivePath),
		http.StatusNoContent,
		"delete archive upload",
	)
}
