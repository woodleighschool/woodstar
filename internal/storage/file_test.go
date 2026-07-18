package storage

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/storage/capability"
)

func TestFileStoreRejectsTraversal(t *testing.T) {
	t.Parallel()
	store := newTestFileStore(t)
	if err := store.Put(context.Background(), "../escape", bytes.NewReader([]byte("x")), PutOptions{}); err == nil {
		t.Fatal("Put with traversal key returned nil error, want rejection")
	}
}

func TestFileStoreDeliversCanonicalObjectDirectly(t *testing.T) {
	t.Parallel()
	store := newTestFileStore(t)
	sha256sum := strings.Repeat("a", 64)
	object := Object{
		ID:          7,
		Prefix:      "munki/icons",
		Filename:    "App.png",
		ContentType: "image/png",
		SHA256:      &sha256sum,
	}
	if err := store.Put(t.Context(), object.Key(), strings.NewReader("icon bytes"), PutOptions{}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/munki/icons/7/content", nil)
	if err := NewDelivery(store).Deliver(rec, req, object, DeliveryOptions{
		CacheControl: "private, max-age=86400",
	}); err != nil {
		t.Fatalf("Deliver: %v", err)
	}

	if rec.Code != http.StatusOK || rec.Body.String() != "icon bytes" {
		t.Fatalf("delivery status/body = %d/%q, want 200/icon bytes", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != object.ContentType {
		t.Fatalf("Content-Type = %q, want %q", got, object.ContentType)
	}
	if got := rec.Header().Get("Cache-Control"); got != "private, max-age=86400" {
		t.Fatalf("Cache-Control = %q, want private max-age", got)
	}
	if got := rec.Header().Get("ETag"); got != object.ETag() {
		t.Fatalf("ETag = %q, want %q", got, object.ETag())
	}
}

func TestFileStorePresignGetProducesBlobCapability(t *testing.T) {
	t.Parallel()
	store := newTestFileStore(t)
	now := time.Now()

	rawURL, err := store.PresignGet(
		context.Background(),
		"munki/icons/7/App Icon.png",
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
	if got := parsed.Scheme + "://" + parsed.Host + parsed.EscapedPath(); got != "https://woodstar.example/storage/munki/icons/7/App%20Icon.png" {
		t.Fatalf("blob URL = %q, want path-bound storage URL", got)
	}
	claims, err := capability.Verify[BlobCapabilityClaims](
		testCapabilityKey,
		parsed.Query().Get("cap"),
		capability.OpGet,
		now,
	)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Key != "munki/icons/7/App Icon.png" {
		t.Fatalf("key = %q, want object key", claims.Key)
	}
	if claims.ContentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", claims.ContentType)
	}
}

func TestFileStorePresignPutProducesUploadTarget(t *testing.T) {
	t.Parallel()
	store := newTestFileStore(t)
	now := time.Now()

	target, err := store.PresignPut(
		context.Background(),
		"munki/packages/42/Installer.pkg",
		time.Minute,
	)
	if err != nil {
		t.Fatalf("PresignPut: %v", err)
	}
	if target.Method != http.MethodPut {
		t.Fatalf("method = %q, want PUT", target.Method)
	}
	parsed, err := url.Parse(target.URL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if got := parsed.Scheme + "://" + parsed.Host + parsed.Path; got != "https://woodstar.example/storage/munki/packages/42/Installer.pkg" {
		t.Fatalf("url path = %q, want path-bound storage URL", got)
	}
	if parsed.Query().Get("cap") == "" {
		t.Fatalf("url = %q, want capability token", target.URL)
	}
	claims, err := capability.Verify[BlobCapabilityClaims](
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
