//go:build postgres

package storage

import (
	"bytes"
	"encoding/hex"
	"net/url"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/storage/capability"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestDirectUploadTargetsPendingObjectKey(t *testing.T) {
	db, ctx := testdb.Open(t)
	backend := newTestBackend(t)
	objects := NewObjectStore(db, backend, testLogger())
	uploads := NewIngestor(objects, backend)
	object, target, err := uploads.BeginDirect(ctx, "munki/packages", "Installer.pkg")
	if err != nil {
		t.Fatalf("begin upload: %v", err)
	}
	parsed, err := url.Parse(target.URL)
	if err != nil {
		t.Fatalf("parse upload URL: %v", err)
	}
	claims, err := capability.Verify[BlobCapabilityClaims](
		testCapabilityKey,
		parsed.Query().Get("cap"),
		capability.OpPut,
		time.Now(),
	)
	if err != nil {
		t.Fatalf("verify upload capability: %v", err)
	}
	if claims.Key != object.Key() {
		t.Fatalf("upload key = %q, want object key %q", claims.Key, object.Key())
	}

	body := []byte("installer bytes")
	if err := backend.Put(ctx, object.Key(), bytes.NewReader(body), PutOptions{}); err != nil {
		t.Fatalf("put upload bytes: %v", err)
	}
	finalized, err := uploads.Finalize(ctx, object.ID, object.Prefix)
	if err != nil {
		t.Fatalf("finalize upload: %v", err)
	}
	if !finalized.Available() {
		t.Fatal("finalized object is still pending")
	}
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
