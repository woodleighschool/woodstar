package storage_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func TestS3MultipartRetryUsesRecordedUploadAndCanonicalCompletion(t *testing.T) {
	db, ctx := dbtest.Open(t)
	const body = "whole installer bytes"
	var mu sync.Mutex
	createRequests := 0
	completeRequests := 0
	canonicalExists := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		switch {
		case r.Method == http.MethodPost && r.URL.Query().Has("uploads"):
			mu.Lock()
			createRequests++
			mu.Unlock()
			_, _ = io.WriteString(
				w,
				`<CreateMultipartUploadResult><Bucket>test</Bucket><Key>installer</Key><UploadId>upload-1</UploadId></CreateMultipartUploadResult>`,
			)
		case r.Method == http.MethodPost && r.URL.Query().Get("uploadId") == "upload-1":
			mu.Lock()
			completeRequests++
			canonicalExists = true
			mu.Unlock()
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(
				w,
				`<Error><Code>NoSuchUpload</Code><Message>completion response was lost</Message></Error>`,
			)
		case r.Method == http.MethodGet:
			mu.Lock()
			exists := canonicalExists
			mu.Unlock()
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `<Error><Code>NoSuchKey</Code></Error>`)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			_, _ = io.WriteString(w, body)
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	backend, err := storage.New(ctx, storage.Config{
		Kind:        storage.KindS3,
		TransferTTL: time.Minute,
		S3: storage.S3Config{
			Bucket:    "test",
			Region:    "us-east-1",
			Endpoint:  server.URL,
			AccessKey: "access",
			SecretKey: "secret",
			PathStyle: true,
		},
	})
	if err != nil {
		t.Fatalf("create S3 backend: %v", err)
	}
	objects := storage.NewObjectStore(db, backend, slog.New(slog.DiscardHandler))
	uploads := storage.NewIngestor(objects, backend)
	object, action, err := uploads.Begin(ctx, packages.ObjectPrefix, "Installer.pkg")
	if err != nil {
		t.Fatalf("begin S3 upload: %v", err)
	}
	if _, ok := action.(storage.MultipartUploadAction); !ok {
		t.Fatalf("S3 upload action = %T, want storage.MultipartUploadAction", action)
	}

	first, err := uploads.CreateMultipart(ctx, object.ID, packages.ObjectPrefix)
	if err != nil {
		t.Fatalf("create multipart: %v", err)
	}
	second, err := uploads.CreateMultipart(ctx, object.ID, packages.ObjectPrefix)
	if err != nil {
		t.Fatalf("retry create multipart: %v", err)
	}
	if first.UploadID != "upload-1" || second != first {
		t.Fatalf("multipart uploads = %+v and %+v, want recorded upload-1", first, second)
	}
	mu.Lock()
	gotCreateRequests := createRequests
	mu.Unlock()
	if gotCreateRequests != 1 {
		t.Fatalf("create requests = %d, want 1", gotCreateRequests)
	}

	if err := uploads.CompleteMultipart(ctx, object.ID, packages.ObjectPrefix, []storage.CompletedPart{
		{PartNumber: 1, ETag: `"etag-1"`},
	}); err != nil {
		t.Fatalf("complete missing upload with canonical object: %v", err)
	}
	mu.Lock()
	gotCompleteRequests := completeRequests
	mu.Unlock()
	if gotCompleteRequests != 1 {
		t.Fatalf("complete requests = %d, want 1", gotCompleteRequests)
	}
	if err := uploads.CompleteMultipart(ctx, object.ID, packages.ObjectPrefix, []storage.CompletedPart{
		{PartNumber: 1, ETag: `"etag-1"`},
	}); err != nil {
		t.Fatalf("retry completed multipart: %v", err)
	}

	finalized, err := uploads.Finalize(ctx, object.ID, packages.ObjectPrefix)
	if err != nil {
		t.Fatalf("finalize canonical multipart object: %v", err)
	}
	wantHash := sha256.Sum256([]byte(body))
	if finalized.SHA256 == nil || *finalized.SHA256 != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("SHA-256 = %v, want whole-file digest", finalized.SHA256)
	}
	if finalized.MultipartUploadID != nil {
		t.Fatalf("multipart upload ID = %v, want closed", finalized.MultipartUploadID)
	}
}
