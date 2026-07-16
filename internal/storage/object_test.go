package storage

import (
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestNormalizeUploadFilename(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Firefox-120.0.dmg":   "Firefox-120.0.dmg",
		"/tmp/Firefox.dmg":    "Firefox.dmg",
		`C:\Users\me\App.pkg`: "App.pkg",
		"  Spaced.icns  ":     "Spaced.icns",
		"Acmé Café.png":       "Acmé Café.png",
		"sub/dir/file.pkg":    "file.pkg",
		"../../etc/passwd":    "passwd",
	}
	for in, want := range cases {
		got := normalizeUploadFilename(in)
		if got != want {
			t.Errorf("normalizeUploadFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateUploadFilenameRejects(t *testing.T) {
	t.Parallel()
	for _, in := range []string{
		"",
		"   ",
		".",
		"..",
		"/",
		"a/b/..",
		"with\x00null.pkg",
	} {
		name := normalizeUploadFilename(in)
		if err := validateUploadFilename(name); !errors.Is(err, dbutil.ErrInvalidInput) {
			t.Errorf("validateUploadFilename(%q) error = %v, want ErrInvalidInput", name, err)
		}
	}
}

func TestListByPrefixReturnsAvailableObjectsNewestFirst(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewObjectStore(db, nil)

	first, err := store.CreatePending(ctx, "munki/icons", "first.png")
	if err != nil {
		t.Fatalf("create first object: %v", err)
	}
	if _, err := store.MarkAvailable(
		ctx,
		first.ID,
		1,
		"image/png",
		strings.Repeat("a", 64),
	); err != nil {
		t.Fatalf("finalize first object: %v", err)
	}
	second, err := store.CreatePending(ctx, "munki/icons", "second.png")
	if err != nil {
		t.Fatalf("create second object: %v", err)
	}
	if _, err := store.MarkAvailable(
		ctx,
		second.ID,
		1,
		"image/png",
		strings.Repeat("b", 64),
	); err != nil {
		t.Fatalf("finalize second object: %v", err)
	}
	if _, err := store.CreatePending(ctx, "munki/icons", "pending.png"); err != nil {
		t.Fatalf("create pending object: %v", err)
	}
	other, err := store.CreatePending(ctx, "munki/packages", "other.pkg")
	if err != nil {
		t.Fatalf("create other-prefix object: %v", err)
	}
	if _, err := store.MarkAvailable(
		ctx,
		other.ID,
		1,
		"application/octet-stream",
		strings.Repeat("c", 64),
	); err != nil {
		t.Fatalf("finalize other-prefix object: %v", err)
	}

	objects, count, err := store.ListByPrefix(ctx, "munki/icons", dbutil.ListParams{})
	if err != nil {
		t.Fatalf("list objects: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if len(objects) != 2 || objects[0].ID != second.ID || objects[1].ID != first.ID {
		t.Fatalf("object IDs = %v, want [%d %d]", objectIDs(objects), second.ID, first.ID)
	}
}

func TestMarkAvailableNormalizesContentType(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewObjectStore(db, nil)
	object, err := store.CreatePending(ctx, "munki/icons", "icon.png")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}

	available, err := store.MarkAvailable(
		ctx,
		object.ID,
		1,
		"IMAGE/PNG; profile=\"screen\"",
		strings.Repeat("a", 64),
	)
	if err != nil {
		t.Fatalf("mark available: %v", err)
	}
	if available.ContentType != "image/png; profile=screen" {
		t.Fatalf("content type = %q, want normalized media type", available.ContentType)
	}
}

func TestMarkAvailableRejectsInvalidContentType(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewObjectStore(db, nil)
	object, err := store.CreatePending(ctx, "munki/icons", "icon.png")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}

	_, err = store.MarkAvailable(ctx, object.ID, 1, "not a content type", strings.Repeat("a", 64))
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("mark available error = %v, want ErrInvalidInput", err)
	}
}

func TestMultipartUploadIDMustBeNonblankAndClosedBeforeAvailability(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := NewObjectStore(db, nil)
	object, err := store.CreatePending(ctx, "munki/packages", "installer.pkg")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}
	if _, _, err := store.RecordMultipartUploadID(ctx, object.ID, "  "); !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("record blank multipart ID error = %v, want ErrInvalidInput", err)
	}
	uploadID, created, err := store.RecordMultipartUploadID(ctx, object.ID, "upload-1")
	if err != nil {
		t.Fatalf("record multipart ID: %v", err)
	}
	if !created || uploadID != "upload-1" {
		t.Fatalf("recorded multipart = %q/%t, want upload-1/true", uploadID, created)
	}
	_, err = store.MarkAvailable(
		ctx,
		object.ID,
		1,
		"application/octet-stream",
		strings.Repeat("a", 64),
	)
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("finalize open multipart error = %v, want ErrInvalidInput", err)
	}
	if err := store.ClearMultipartUploadID(ctx, object.ID, uploadID); err != nil {
		t.Fatalf("clear multipart ID: %v", err)
	}
	available, err := store.MarkAvailable(
		ctx,
		object.ID,
		1,
		"application/octet-stream",
		strings.Repeat("a", 64),
	)
	if err != nil {
		t.Fatalf("finalize closed multipart: %v", err)
	}
	if available.MultipartUploadID != nil {
		t.Fatalf("available multipart ID = %v, want nil", available.MultipartUploadID)
	}
}

func objectIDs(objects []Object) []int64 {
	ids := make([]int64, len(objects))
	for i, object := range objects {
		ids[i] = object.ID
	}
	return ids
}
