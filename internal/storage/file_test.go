package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/storage/capability"
)

func TestFileStoreRoundTrip(t *testing.T) {
	t.Parallel()
	store := newTestFileStore(t)
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
	store := newTestFileStoreAt(t, root)
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
	store := newTestFileStore(t)
	if err := store.Put(context.Background(), "../escape", bytes.NewReader([]byte("x")), PutOptions{}); err == nil {
		t.Fatal("Put with traversal key returned nil error, want rejection")
	}
}

func TestFileStorePresignGetProducesBlobCapability(t *testing.T) {
	t.Parallel()
	store := newTestFileStore(t)
	now := time.Now()

	rawURL, err := store.PresignGet(
		context.Background(),
		"munki/icons/7/icon.png",
		time.Minute,
		GetOptions{ContentType: "image/png"},
	)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if got := parsed.Scheme + "://" + parsed.Host + parsed.Path; got != "https://woodstar.example/storage/blob" {
		t.Fatalf("blob URL = %q, want https://woodstar.example/storage/blob", got)
	}
	claims, err := capability.Verify(
		testCapabilityKey,
		parsed.Query().Get("cap"),
		capability.OpGet,
		now,
	)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Key != "munki/icons/7/icon.png" {
		t.Fatalf("key = %q, want object key", claims.Key)
	}
	if claims.ContentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", claims.ContentType)
	}
}

func TestFileStorePresignPutProducesWoodstarUploadTarget(t *testing.T) {
	t.Parallel()
	store := newTestFileStore(t)
	now := time.Now()

	target, err := store.PresignPut(
		context.Background(),
		"munki/packages/42/Installer.pkg",
		time.Minute,
		PutOptions{ContentType: "application/octet-stream"},
	)
	if err != nil {
		t.Fatalf("PresignPut: %v", err)
	}
	if target.Method != "PUT" {
		t.Fatalf("method = %q, want PUT", target.Method)
	}
	if target.Transport != UploadTransportWoodstar {
		t.Fatalf("transport = %q, want woodstar", target.Transport)
	}
	if !strings.HasPrefix(target.URL, "https://woodstar.example/storage/blob?cap=") {
		t.Fatalf("url = %q, want blob capability URL", target.URL)
	}
	parsed, err := url.Parse(target.URL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	claims, err := capability.Verify(
		testCapabilityKey,
		parsed.Query().Get("cap"),
		capability.OpPut,
		now,
	)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Key != "munki/packages/42/Installer.pkg" {
		t.Fatalf("key = %q, want object key", claims.Key)
	}
}

func TestNewRejectsUnknownKind(t *testing.T) {
	t.Parallel()
	if _, err := New(context.Background(), Config{Kind: "bogus"}); err == nil {
		t.Fatal("New with unknown kind returned nil error")
	}
}

func newTestFileStore(t *testing.T) *fileStore {
	t.Helper()
	return newTestFileStoreAt(t, t.TempDir())
}

func newTestFileStoreAt(t *testing.T, root string) *fileStore {
	t.Helper()
	store, err := newFileStore(root, "https://woodstar.example", testCapabilityKey, time.Minute)
	if err != nil {
		t.Fatalf("newFileStore: %v", err)
	}
	return store
}

var testCapabilityKey = []byte("storage capability test key")
