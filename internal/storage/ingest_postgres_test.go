//go:build postgres

package storage

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestFinalizeMakesCanonicalObjectImmutableFromUploadTarget(t *testing.T) {
	db, ctx := testdb.Open(t)
	backend := newTestBackend(t)
	objects := NewObjectStore(db, backend, testLogger())
	uploads := NewIngestor(objects, backend)
	object, target, err := uploads.BeginDirect(ctx, "munki/packages", "Installer.pkg")
	if err != nil {
		t.Fatalf("begin upload: %v", err)
	}
	initial := []byte("initial installer bytes")
	if err := backend.Put(
		ctx,
		stagingKey(object.ID),
		bytes.NewReader(initial),
		PutOptions{},
	); err != nil {
		t.Fatalf("put upload bytes: %v", err)
	}
	if _, err := uploads.Finalize(ctx, object.ID, object.Prefix); err != nil {
		t.Fatalf("finalize upload: %v", err)
	}

	if err := backend.Put(
		ctx,
		stagingKey(object.ID),
		bytes.NewReader([]byte("late replacement")),
		PutOptions{},
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
	db, ctx := testdb.Open(t)
	backend := &overwritingMoveBackend{
		Backend:     newTestBackend(t),
		replacement: []byte("replacement bytes"),
	}
	objects := NewObjectStore(db, backend, testLogger())
	uploads := NewIngestor(objects, backend)
	object, err := objects.CreatePending(ctx, "munki/icons", "icon.png")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}
	if err := backend.Put(
		ctx,
		stagingKey(object.ID),
		bytes.NewReader([]byte("initial bytes")),
		PutOptions{},
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
	Backend

	replacement []byte
}

func (b *overwritingMoveBackend) Move(
	ctx context.Context,
	sourceKey string,
	destinationKey string,
	options PutOptions,
) error {
	if err := b.Put(ctx, sourceKey, bytes.NewReader(b.replacement), PutOptions{}); err != nil {
		return err
	}
	return b.Backend.Move(ctx, sourceKey, destinationKey, options)
}

func newTestBackend(t *testing.T) Backend {
	t.Helper()
	backend, err := New(t.Context(), Config{
		Kind:        KindFile,
		TransferTTL: time.Minute,
		File: FileConfig{
			Root:             t.TempDir(),
			BaseURL:          "https://woodstar.example",
			CapabilityKeyHex: hex.EncodeToString(bytes.Repeat([]byte{0x42}, 32)),
		},
	})
	if err != nil {
		t.Fatalf("create file storage: %v", err)
	}
	return backend
}
