package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestFinalizeMakesCanonicalObjectImmutableFromUploadTarget(t *testing.T) {
	db, ctx := dbtest.Open(t)
	backend := newTestBackend(t)
	objects := NewObjectStore(db, backend)
	uploads := NewIngestor(objects, backend)
	object, target, err := uploads.Begin(ctx, "munki/packages", "Installer.pkg")
	if err != nil {
		t.Fatalf("begin upload: %v", err)
	}
	initial := []byte("initial installer bytes")
	if err := backend.Put(
		ctx,
		uploadKey(object.ID),
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
		uploadKey(object.ID),
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
	db, ctx := dbtest.Open(t)
	backend := &overwritingMoveBackend{
		Backend:     newTestBackend(t),
		replacement: []byte("replacement bytes"),
	}
	objects := NewObjectStore(db, backend)
	uploads := NewIngestor(objects, backend)
	object, err := objects.CreatePending(ctx, "munki/icons", "icon.png")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}
	if err := backend.Put(
		ctx,
		uploadKey(object.ID),
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

func TestValidateCompletedPartsRequiresStrictAscendingNonemptyParts(t *testing.T) {
	t.Parallel()
	valid := []CompletedPart{
		{PartNumber: 1, ETag: `"first"`},
		{PartNumber: 10_000, ETag: `"last"`},
	}
	if err := validateCompletedParts(valid); err != nil {
		t.Fatalf("validate ascending parts: %v", err)
	}
	for name, parts := range map[string][]CompletedPart{
		"empty":      nil,
		"zero":       {{PartNumber: 0, ETag: `"etag"`}},
		"too large":  {{PartNumber: 10_001, ETag: `"etag"`}},
		"blank etag": {{PartNumber: 1, ETag: "  "}},
		"duplicate":  {{PartNumber: 1, ETag: `"one"`}, {PartNumber: 1, ETag: `"again"`}},
		"descending": {{PartNumber: 2, ETag: `"two"`}, {PartNumber: 1, ETag: `"one"`}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateCompletedParts(parts); !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("validate parts error = %v, want ErrInvalidInput", err)
			}
		})
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
		Kind:          KindFile,
		FileRoot:      t.TempDir(),
		BaseURL:       "https://woodstar.example",
		CapabilityKey: []byte("object upload test capability key"),
	})
	if err != nil {
		t.Fatalf("create file storage: %v", err)
	}
	return backend
}
