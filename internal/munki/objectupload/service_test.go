package objectupload

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestFinalizeDerivesMetadataFromLandedBytes(t *testing.T) {
	db, ctx := dbtest.Open(t)
	backend := newTestBackend(t)
	objects := storage.NewObjectStore(db, backend)
	uploads := NewService(objects, backend)

	object, err := objects.CreatePending(ctx, "munki/packages", "Installer.pkg")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}
	const body = "installer payload bytes"
	if err := backend.Put(
		ctx,
		uploadKey(object.ID),
		bytes.NewReader([]byte(body)),
		storage.PutOptions{},
	); err != nil {
		t.Fatalf("put object bytes: %v", err)
	}

	finalized, err := uploads.Finalize(ctx, object.ID, object.Prefix)
	if err != nil {
		t.Fatalf("finalize upload: %v", err)
	}
	wantHash := sha256.Sum256([]byte(body))
	if finalized.SizeBytes == nil || *finalized.SizeBytes != int64(len(body)) {
		t.Fatalf("size = %v, want %d", finalized.SizeBytes, len(body))
	}
	if finalized.SHA256 == nil || *finalized.SHA256 != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("sha256 = %v, want server-derived hash", finalized.SHA256)
	}
	if finalized.AvailableAt == nil {
		t.Fatal("finalized object not marked available")
	}
	if finalized.ContentType != "text/plain; charset=utf-8" {
		t.Fatalf("content type = %q, want detected text/plain", finalized.ContentType)
	}
}

func TestFinalizeMakesCanonicalObjectImmutableFromUploadTarget(t *testing.T) {
	db, ctx := dbtest.Open(t)
	backend := newTestBackend(t)
	objects := storage.NewObjectStore(db, backend)
	uploads := NewService(objects, backend)
	object, target, err := uploads.Begin(ctx, "munki/packages", "Installer.pkg")
	if err != nil {
		t.Fatalf("begin upload: %v", err)
	}
	initial := []byte("initial installer bytes")
	if err := backend.Put(
		ctx,
		uploadKey(object.ID),
		bytes.NewReader(initial),
		storage.PutOptions{},
	); err != nil {
		t.Fatalf("put upload bytes: %v", err)
	}
	if _, err := uploads.Finalize(ctx, object.ID, object.Prefix); err != nil {
		t.Fatalf("finalize upload: %v", err)
	}

	if err := backend.Put(
		ctx,
		uploadKey(object.ID),
		bytes.NewReader([]byte("late replacement")),
		storage.PutOptions{},
	); err != nil {
		t.Fatalf("write through stale upload target %q: %v", target.URL, err)
	}
	reader, _, err := backend.Open(ctx, object.Key())
	if err != nil {
		t.Fatalf("open canonical object: %v", err)
	}
	got, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatalf("read canonical object: %v", err)
	}
	if !bytes.Equal(got, initial) {
		t.Fatalf("canonical bytes = %q, want %q", got, initial)
	}
}

func TestFinalizeRejectsBytesChangedDuringPromotion(t *testing.T) {
	db, ctx := dbtest.Open(t)
	backend := &overwritingMoveBackend{
		Backend:     newTestBackend(t),
		replacement: []byte("replacement bytes"),
	}
	objects := storage.NewObjectStore(db, backend)
	uploads := NewService(objects, backend)
	object, err := objects.CreatePending(ctx, "munki/icons", "icon.png")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}
	if err := backend.Put(
		ctx,
		uploadKey(object.ID),
		bytes.NewReader([]byte("initial bytes")),
		storage.PutOptions{},
	); err != nil {
		t.Fatalf("put upload bytes: %v", err)
	}

	_, err = uploads.Finalize(ctx, object.ID, object.Prefix)
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("finalize changed upload error = %v, want ErrInvalidInput", err)
	}
	if _, err := objects.GetByID(ctx, object.ID); !errors.Is(err, dbutil.ErrNotFound) {
		t.Fatalf("get rejected upload error = %v, want ErrNotFound", err)
	}
}

type overwritingMoveBackend struct {
	storage.Backend

	replacement []byte
}

func (b *overwritingMoveBackend) Move(
	ctx context.Context,
	sourceKey string,
	destinationKey string,
	options storage.PutOptions,
) error {
	if err := b.Put(ctx, sourceKey, bytes.NewReader(b.replacement), storage.PutOptions{}); err != nil {
		return err
	}
	return b.Backend.Move(ctx, sourceKey, destinationKey, options)
}

func newTestBackend(t *testing.T) storage.Backend {
	t.Helper()
	backend, err := storage.New(t.Context(), storage.Config{
		Kind:          storage.KindFile,
		FileRoot:      t.TempDir(),
		BaseURL:       "https://woodstar.example",
		CapabilityKey: []byte("object upload test capability key"),
	})
	if err != nil {
		t.Fatalf("create file storage: %v", err)
	}
	return backend
}
