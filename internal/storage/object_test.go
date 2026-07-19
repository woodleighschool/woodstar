package storage

import (
	"context"
	"errors"
	"log/slog"
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

func TestDeleteRetainsRegistryRowWhenBackendDeletionFails(t *testing.T) {
	db, ctx := dbtest.Open(t)
	backend := &deletionBackend{err: errors.New("backend unavailable")}
	store := NewObjectStore(db, backend)
	object, err := store.CreatePending(ctx, "munki/icons", "icon.png")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}

	if err := store.Delete(ctx, object.ID); err == nil {
		t.Fatal("delete error = nil, want backend failure")
	}
	retained, err := store.GetByID(ctx, object.ID)
	if err != nil {
		t.Fatalf("get object after failed deletion: %v", err)
	}
	if retained.DeletionRequestedAt == nil {
		t.Fatal("failed deletion was not retained for retry")
	}

	backend.err = nil
	if err := store.Delete(ctx, object.ID); err != nil {
		t.Fatalf("retry deletion: %v", err)
	}
	if _, err := store.GetByID(ctx, object.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get deleted object error = %v, want ErrNotFound", err)
	}
}

func TestDeletionSweepRetriesOutsideCanceledRequestContext(t *testing.T) {
	db, ctx := dbtest.Open(t)
	backend := &deletionBackend{err: errors.New("backend unavailable")}
	store := NewObjectStore(db, backend)
	object, err := store.CreatePending(ctx, "munki/icons", "icon.png")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}
	requestCtx, cancelRequest := context.WithCancel(ctx)
	if err := store.RequestDeletion(requestCtx, db.Pool(), object.ID); err != nil {
		t.Fatalf("request deletion: %v", err)
	}
	cancelRequest()

	logger := slog.New(slog.DiscardHandler)
	sweepDeletions(t.Context(), store, logger)
	queued, err := store.GetByID(ctx, object.ID)
	if err != nil {
		t.Fatalf("get object after failed sweep: %v", err)
	}
	if queued.DeletionRequestedAt == nil {
		t.Fatal("failed sweep cleared deletion request")
	}

	backend.err = nil
	sweepDeletions(t.Context(), store, logger)
	if _, err := store.GetByID(ctx, object.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get object after retry error = %v, want ErrNotFound", err)
	}
	if backend.sawCanceledContext {
		t.Fatal("deletion used the canceled request context")
	}
}

func TestDeletionSweepDoesNotDeleteReferencedObjectBytes(t *testing.T) {
	db, ctx := dbtest.Open(t)
	backend := &deletionBackend{}
	store := NewObjectStore(db, backend)
	object, err := store.CreatePending(ctx, "munki/icons", "icon.png")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}
	object, err = store.MarkAvailable(ctx, object.ID, 1, "image/png", strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("mark object available: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
INSERT INTO munki_software (name, display_name, icon_object_id)
VALUES ('Referenced', 'Referenced', $1)`, object.ID); err != nil {
		t.Fatalf("reference object: %v", err)
	}
	if err := store.RequestDeletion(ctx, db.Pool(), object.ID); err != nil {
		t.Fatalf("request deletion: %v", err)
	}

	sweepDeletions(ctx, store, slog.New(slog.DiscardHandler))
	retained, err := store.GetByID(ctx, object.ID)
	if err != nil {
		t.Fatalf("get referenced object: %v", err)
	}
	if retained.DeletionRequestedAt == nil {
		t.Fatal("referenced object lost its deletion request")
	}
	if backend.calls != 0 {
		t.Fatalf("backend delete calls = %d, want 0", backend.calls)
	}
	if _, err := db.Pool().Exec(ctx, `DELETE FROM munki_software WHERE icon_object_id = $1`, object.ID); err != nil {
		t.Fatalf("remove object reference: %v", err)
	}
	sweepDeletions(ctx, store, slog.New(slog.DiscardHandler))
	if _, err := store.GetByID(ctx, object.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get object after final reference removal error = %v, want ErrNotFound", err)
	}
	if backend.calls != 1 {
		t.Fatalf("backend delete calls after final unlink = %d, want 1", backend.calls)
	}
}

type deletionBackend struct {
	err                error
	sawCanceledContext bool
	calls              int
}

func (b *deletionBackend) Delete(ctx context.Context, _ string) error {
	b.calls++
	b.sawCanceledContext = b.sawCanceledContext || ctx.Err() != nil
	return b.err
}

func objectIDs(objects []Object) []int64 {
	ids := make([]int64, len(objects))
	for i, object := range objects {
		ids[i] = object.ID
	}
	return ids
}
