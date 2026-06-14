package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreRoundTrip(t *testing.T) {
	t.Parallel()
	store, err := newFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("newFileStore: %v", err)
	}
	ctx := context.Background()
	key := "munki/packages/42/installer.pkg"
	want := []byte("installer bytes")

	if err := store.Put(ctx, key, bytes.NewReader(want), PutOptions{}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	info, err := store.Stat(ctx, key)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size != int64(len(want)) {
		t.Fatalf("Stat size = %d, want %d", info.Size, len(want))
	}

	rc, _, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Open returned %q, want %q", got, want)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Stat(ctx, key); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("Stat after delete = %v, want ErrObjectNotFound", err)
	}
}

func TestFileStoreDeletePrunesEmptyDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := newFileStore(root)
	if err != nil {
		t.Fatalf("newFileStore: %v", err)
	}
	ctx := context.Background()
	if err := store.Put(ctx, "munki/icons/7/icon.png", bytes.NewReader([]byte("x")), PutOptions{}); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := store.Delete(ctx, "munki/icons/7/icon.png"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "munki", "icons", "7")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("object dir still present: %v", err)
	}
}

func TestFileStoreRejectsTraversal(t *testing.T) {
	t.Parallel()
	store, err := newFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("newFileStore: %v", err)
	}
	if err := store.Put(context.Background(), "../escape", bytes.NewReader([]byte("x")), PutOptions{}); err == nil {
		t.Fatal("Put with traversal key returned nil error, want rejection")
	}
}

func TestNewRejectsUnknownKind(t *testing.T) {
	t.Parallel()
	if _, err := New(context.Background(), Config{Kind: "bogus"}); err == nil {
		t.Fatal("New with unknown kind returned nil error")
	}
}
