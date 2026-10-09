//go:build postgres

package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/woodleighschool/goodies/bloby"
)

func TestMunkiPackageInstallerFileLifecycle(t *testing.T) {
	fixture := newMunkiFixture(t)
	installerPath := func(objectID int64) string {
		return fmt.Sprintf("%s/%d", munkiPackageInstallerPath, objectID)
	}
	createPackage := func(t *testing.T, objectID int64) int {
		t.Helper()
		return fixture.requestJSON(t, http.MethodPost, munkiPackagePath, json.RawMessage(fmt.Sprintf(
			`{"software_id":%d,"version":"1.0","installer_type":"pkg","installer_object_id":%d}`,
			fixture.softwareID, objectID,
		))).Code
	}
	assertReleased := func(t *testing.T, objectID int64) {
		t.Helper()
		if _, err := fixture.objects.GetByID(t.Context(), objectID); !errors.Is(err, bloby.ErrNotFound) {
			t.Fatalf("get released upload error = %v, want ErrNotFound", err)
		}
	}

	t.Run("cancel pending upload", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "cancel.pkg", []byte("cancelled installer"))
		rec := fixture.request(t, http.MethodDelete, installerPath(target.ObjectID))
		assertStatus(t, rec, http.StatusNoContent, "cancel installer")
		assertReleased(t, target.ObjectID)
	})

	t.Run("finalize publishes the declared content", func(t *testing.T) {
		body := []byte("published installer")
		target := fixture.stage(t, munkiPackageInstallerPath, "published.pkg", body)
		content, err := bloby.Digest(bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		// A repeated request returns the same published object.
		for range 2 {
			rec := fixture.request(t, http.MethodPut, installerPath(target.ObjectID))
			assertStatus(t, rec, http.StatusOK, "finalize installer")
			var view MunkiObjectView
			decodeJSON(t, rec, &view)
			if view.ID != target.ObjectID || view.SizeBytes != content.SizeBytes || view.SHA256 != content.SHA256 {
				t.Fatalf("finalized installer = %+v, want object %d with declared %+v", view, target.ObjectID, content)
			}
		}
		downloaded := fixture.request(t, http.MethodGet, contentURL(munkiPackageInstallerPath, target.ObjectID))
		assertStatus(t, downloaded, http.StatusOK, "get installer content")
		if !bytes.Equal(downloaded.Body.Bytes(), body) {
			t.Fatalf("installer content = %q, want %q", downloaded.Body.Bytes(), body)
		}
	})

	t.Run("finalize before upload releases the object", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "missing.pkg", []byte("never sent"))
		rec := fixture.request(t, http.MethodPut, installerPath(target.ObjectID))
		assertProblem(t, rec, http.StatusBadRequest)
		assertReleased(t, target.ObjectID)
	})

	t.Run("storage refuses undeclared content", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "undeclared.pkg", []byte("declared installer"))
		rec := fixture.put(t, target, []byte("tampered installer"))
		assertStatus(t, rec, http.StatusBadRequest, "upload undeclared content")
		object, err := fixture.objects.GetByID(t.Context(), target.ObjectID)
		if err != nil || object.Available() {
			t.Fatalf("refused upload = %+v, %v; want a pending object", object, err)
		}
		assertStatus(
			t,
			fixture.request(t, http.MethodGet, contentURL(munkiPackageInstallerPath, target.ObjectID)),
			http.StatusNotFound,
			"get refused content",
		)
		assertProblem(t, fixture.request(t, http.MethodPut, installerPath(target.ObjectID)), http.StatusBadRequest)
		assertReleased(t, target.ObjectID)
	})

	t.Run("pending installer cannot be attached", func(t *testing.T) {
		target := fixture.stage(t, munkiPackageInstallerPath, "pending.pkg", []byte("pending installer"))
		if status := createPackage(t, target.ObjectID); status != http.StatusBadRequest {
			t.Fatalf("attach pending installer status = %d, want %d", status, http.StatusBadRequest)
		}
		assertStatus(
			t,
			fixture.request(t, http.MethodPut, installerPath(target.ObjectID)),
			http.StatusOK,
			"finalize installer",
		)
		if status := createPackage(t, target.ObjectID); status != http.StatusCreated {
			t.Fatalf("attach finalized installer status = %d, want %d", status, http.StatusCreated)
		}
		assertStatus(
			t,
			fixture.request(t, http.MethodDelete, installerPath(target.ObjectID)),
			http.StatusConflict,
			"delete attached installer",
		)
	})

	t.Run("delete rejects another object prefix", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiIconPath, "icon.png", pngSignature)
		path := installerPath(target.ObjectID)
		assertStatus(t, fixture.request(t, http.MethodDelete, path), http.StatusBadRequest, "delete icon as installer")
		if _, err := fixture.objects.GetByID(t.Context(), target.ObjectID); err != nil {
			t.Fatalf("get cross-prefix object: %v", err)
		}
	})

	t.Run("multipart is rejected by file storage", func(t *testing.T) {
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "multipart.pkg", []byte("single part"))
		path := fmt.Sprintf("%s/multipart/parts/1", installerPath(target.ObjectID))
		rec := fixture.requestJSON(t, http.MethodPost, path, MunkiMultipartPartRequest{CRC64NVME: "0123456789abcdef"})
		assertStatus(t, rec, http.StatusBadRequest, "sign multipart part")
	})
}

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

func TestMunkiIconUploadLifecycle(t *testing.T) {
	fixture := newMunkiFixture(t)
	attachPath := fmt.Sprintf("/api/munki/software/%d/icon", fixture.softwareID)
	icon := pngSignature
	target := fixture.stage(t, munkiIconPath, "icon.png", icon)

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
		target := fixture.beginUpload(t, munkiPackageInstallerPath, "wrong-prefix.pkg", []byte("installer"))
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
		target := fixture.stage(t, munkiIconPath, "not-an-icon.txt", []byte("not an image"))
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
	banner := fixture.beginUpload(t, clientResourcesBannerUploadPath, "banner.png", pngSignature)
	archive := fixture.beginUpload(t, clientResourcesArchiveUploadPath, "resources.zip", []byte("archive"))

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
