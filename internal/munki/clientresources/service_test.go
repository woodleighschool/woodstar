package clientresources

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/objectupload"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestServiceSaveReplaceAndDelete(t *testing.T) {
	db, ctx := dbtest.Open(t)
	backend, err := storage.New(ctx, storage.Config{
		Kind:          storage.KindFile,
		FileRoot:      t.TempDir(),
		BaseURL:       "https://woodstar.example",
		CapabilityKey: []byte("client resources test capability key"),
	})
	if err != nil {
		t.Fatalf("create file storage: %v", err)
	}
	objects := storage.NewObjectStore(db, backend)
	uploads := objectupload.NewService(objects, backend)
	service := NewService(NewStore(db, objects), objects, uploads, backend)
	banner := createPendingBanner(t, ctx, uploads, backend)

	first, err := service.Save(ctx, Mutation{
		BannerObjectID:  banner.ID,
		BannerAlignment: BannerAlignmentCenter,
		Links: []Link{{
			Label:         "Support",
			Target:        "https://example.com/support",
			OpenInBrowser: true,
		}},
		FooterText:  "Managed by Example IT",
		FooterLinks: []Link{},
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	firstBanner, err := objects.GetByID(ctx, first.BannerObjectID)
	if err != nil {
		t.Fatalf("get saved banner: %v", err)
	}
	firstArchive, err := objects.GetByID(ctx, first.ArchiveObjectID)
	if err != nil {
		t.Fatalf("get saved archive: %v", err)
	}
	if !firstBanner.Available() || !firstArchive.Available() {
		t.Fatal("Save did not publish the banner and archive")
	}
	if firstBanner.ContentType != "image/png" || firstArchive.ContentType != "application/zip" {
		t.Fatalf("stored content types = %q and %q", firstBanner.ContentType, firstArchive.ContentType)
	}
	files := openArchive(t, ctx, backend, *firstArchive)
	if got := files.body["templates/showcase_template.html"]; !strings.Contains(
		got,
		"left: 50%; transform: translateX(-50%);",
	) {
		t.Fatalf("showcase template = %q", got)
	}
	if _, ok := files.body["templates/sidebar_template.html"]; !ok {
		t.Fatal("links did not produce sidebar_template.html")
	}

	second, err := service.Save(ctx, Mutation{
		BannerObjectID:  first.BannerObjectID,
		BannerAlignment: BannerAlignmentLeft,
		Links:           []Link{},
		FooterLinks:     []Link{},
	})
	if err != nil {
		t.Fatalf("replace Save: %v", err)
	}
	if second.ArchiveObjectID == first.ArchiveObjectID {
		t.Fatal("replace Save reused the prior archive")
	}
	if _, err := objects.GetByID(ctx, first.ArchiveObjectID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("old archive GetByID error = %v, want ErrNotFound", err)
	}

	if err := service.Delete(ctx); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := service.Get(ctx); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("Get after Delete error = %v, want ErrNotFound", err)
	}
	for _, objectID := range []int64{second.BannerObjectID, second.ArchiveObjectID} {
		if _, err := objects.GetByID(ctx, objectID); !errors.Is(err, dbutil.ErrNotFound) {
			t.Fatalf("object %d GetByID error = %v, want ErrNotFound", objectID, err)
		}
	}
}

func createPendingBanner(
	t *testing.T,
	ctx context.Context,
	uploads *objectupload.Service,
	backend storage.Store,
) *storage.Object {
	t.Helper()
	banner, target, err := uploads.Begin(ctx, BannerObjectPrefix, "banner.png")
	if err != nil {
		t.Fatalf("begin banner upload: %v", err)
	}

	var body bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	img.Set(0, 0, color.RGBA{R: 0xff, A: 0xff})
	if err := png.Encode(&body, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	targetURL, err := url.Parse(target.URL)
	if err != nil {
		t.Fatalf("parse banner upload URL: %v", err)
	}
	uploadKey := strings.TrimPrefix(targetURL.Path, "/storage/")
	if err := backend.Put(ctx, uploadKey, bytes.NewReader(body.Bytes()), storage.PutOptions{}); err != nil {
		t.Fatalf("put banner: %v", err)
	}
	return banner
}

func openArchive(
	t *testing.T,
	ctx context.Context,
	backend storage.Store,
	archive storage.Object,
) archiveContents {
	t.Helper()
	reader, _, err := backend.Open(ctx, archive.Key())
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	body, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	return readArchive(t, body)
}
