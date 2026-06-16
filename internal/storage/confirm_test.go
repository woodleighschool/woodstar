package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
)

// TestConfirmUploadedDerivesSizeAndSHA256FromBackend proves the server hashes
// and sizes the landed bytes itself rather than trusting client-supplied
// metadata, which is the integrity guarantee installer uploads rely on.
func TestConfirmUploadedDerivesSizeAndSHA256FromBackend(t *testing.T) {
	db, ctx := dbtest.Open(t)
	backend, err := New(ctx, Config{
		Kind:          KindFile,
		FileRoot:      t.TempDir(),
		PublicURL:     "https://woodstar.example",
		CapabilityKey: testCapabilityKey,
	})
	if err != nil {
		t.Fatalf("create file storage: %v", err)
	}
	store := NewObjectStore(db, backend)

	obj, err := store.CreatePending(ctx, "munki/packages", "Installer.pkg", "application/octet-stream")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}

	const body = "installer payload bytes"
	if err := backend.Put(ctx, obj.Key(), bytes.NewReader([]byte(body)), PutOptions{}); err != nil {
		t.Fatalf("put object bytes: %v", err)
	}

	confirmed, err := store.ConfirmUploaded(ctx, obj.ID)
	if err != nil {
		t.Fatalf("confirm uploaded: %v", err)
	}

	wantHash := sha256.Sum256([]byte(body))
	if confirmed.SizeBytes == nil || *confirmed.SizeBytes != int64(len(body)) {
		t.Fatalf("size = %v, want %d", confirmed.SizeBytes, len(body))
	}
	if confirmed.SHA256 == nil || *confirmed.SHA256 != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("sha256 = %v, want server-derived hash", confirmed.SHA256)
	}
	if confirmed.AvailableAt == nil {
		t.Fatal("confirmed object not marked available")
	}
}
