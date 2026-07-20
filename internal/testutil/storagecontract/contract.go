// Package storagecontract provides shared backend conformance assertions.
package storagecontract

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/woodleighschool/woodstar/internal/storage"
)

// Run verifies the backend operations shared by every storage provider.
func Run(t *testing.T, backend storage.Backend) {
	t.Helper()

	ctx := t.Context()
	key := "contract/source object with spaces.bin"
	first := []byte("woodstar-storage-contract-first")
	if err := backend.Put(ctx, key, bytes.NewReader(first), storage.PutOptions{
		ContentType: "application/octet-stream",
	}); err != nil {
		t.Fatalf("put object: %v", err)
	}
	assertObject(t, backend, key, first)
	assertRange(t, backend, key, first, 5, 12)

	overwritten := []byte("replacement bytes")
	if err := backend.Put(ctx, key, bytes.NewReader(overwritten), storage.PutOptions{
		ContentType: "application/x-woodstar-replacement",
	}); err != nil {
		t.Fatalf("overwrite object: %v", err)
	}
	assertObject(t, backend, key, overwritten)

	destinationKey := "contract/moved object with spaces.bin"
	if err := backend.Move(ctx, key, destinationKey, storage.PutOptions{
		ContentType: "application/x-woodstar-moved",
	}); err != nil {
		t.Fatalf("move object: %v", err)
	}
	assertObject(t, backend, destinationKey, overwritten)
	assertMissing(t, backend, key)

	if err := backend.Delete(ctx, destinationKey); err != nil {
		t.Fatalf("delete object: %v", err)
	}
	assertMissing(t, backend, destinationKey)
	if err := backend.Delete(ctx, destinationKey); err != nil {
		t.Fatalf("delete absent object: %v", err)
	}

	assertMissing(t, backend, "contract/object that never existed.bin")
}

func assertRange(
	t *testing.T,
	backend storage.Backend,
	key string,
	want []byte,
	start int,
	end int,
) {
	t.Helper()

	reader, _, err := backend.Open(t.Context(), key)
	if err != nil {
		t.Fatalf("open %q for range: %v", key, err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			t.Errorf("close %q range: %v", key, err)
		}
	}()
	if _, err := reader.Seek(int64(start), io.SeekStart); err != nil {
		t.Fatalf("seek %q range: %v", key, err)
	}
	got := make([]byte, end-start+1)
	if _, err := io.ReadFull(reader, got); err != nil {
		t.Fatalf("read %q range: %v", key, err)
	}
	if !bytes.Equal(got, want[start:end+1]) {
		t.Fatalf("read %q range bytes = %q, want %q", key, got, want[start:end+1])
	}
}

func assertObject(t *testing.T, backend storage.Backend, key string, want []byte) {
	t.Helper()

	reader, info, err := backend.Open(t.Context(), key)
	if err != nil {
		t.Fatalf("open %q: %v", key, err)
	}
	got, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil {
		t.Fatalf("read %q: %v", key, readErr)
	}
	if closeErr != nil {
		t.Fatalf("close %q: %v", key, closeErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("object %q bytes = %q, want %q", key, got, want)
	}
	if info.Size != int64(len(want)) {
		t.Fatalf("object %q size = %d, want %d", key, info.Size, len(want))
	}
}

func assertMissing(t *testing.T, backend storage.Backend, key string) {
	t.Helper()

	reader, _, err := backend.Open(t.Context(), key)
	if reader != nil {
		if closeErr := reader.Close(); closeErr != nil {
			t.Errorf("close reader returned for missing object %q: %v", key, closeErr)
		}
	}
	if !errors.Is(err, storage.ErrObjectNotFound) {
		t.Fatalf("open missing object %q error = %v, want storage.ErrObjectNotFound", key, err)
	}
}
