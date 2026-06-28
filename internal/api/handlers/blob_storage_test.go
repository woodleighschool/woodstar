package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/storage/capability"
)

func TestBlobGetServesBytesAndRanges(t *testing.T) {
	t.Parallel()
	store := newTestFileStore(t)
	const key = "munki/packages/1/Installer.pkg"
	if err := store.Put(t.Context(), key, strings.NewReader("0123456789"), storage.PutOptions{}); err != nil {
		t.Fatalf("Put: %v", err)
	}
	router := newBlobTestRouter(store)
	token := signBlobCapability(t, storage.BlobCapabilityClaims{
		Op:          capability.OpGet,
		Key:         key,
		Exp:         time.Now().Add(time.Minute).Unix(),
		ContentType: "application/octet-stream",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/storage/munki/packages/1/Installer.pkg?cap="+token, nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Body.String() != "0123456789" {
		t.Fatalf("body = %q, want full object", rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("Content-Type = %q, want application/octet-stream", got)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/storage/munki/packages/1/Installer.pkg?cap="+token, nil)
	req.Header.Set("Range", "bytes=2-5")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("range status = %d, want %d; body = %q", rec.Code, http.StatusPartialContent, rec.Body.String())
	}
	if rec.Body.String() != "2345" {
		t.Fatalf("range body = %q, want 2345", rec.Body.String())
	}
	if got := rec.Header().Get("Content-Range"); got != "bytes 2-5/10" {
		t.Fatalf("Content-Range = %q, want bytes 2-5/10", got)
	}
}

func TestBlobGetRejectsInvalidExpiredAndMissingObjects(t *testing.T) {
	t.Parallel()
	store := newTestFileStore(t)
	router := newBlobTestRouter(store)
	expired := signBlobCapability(t, storage.BlobCapabilityClaims{
		Op:  capability.OpGet,
		Key: "munki/icons/1/icon.png",
		Exp: time.Now().Add(-time.Minute).Unix(),
	})
	missing := signBlobCapability(t, storage.BlobCapabilityClaims{
		Op:  capability.OpGet,
		Key: "munki/icons/1/icon.png",
		Exp: time.Now().Add(time.Minute).Unix(),
	})

	cases := []struct {
		name string
		cap  string
		want int
	}{
		{name: "invalid", cap: "invalid", want: http.StatusUnauthorized},
		{name: "expired", cap: expired, want: http.StatusGone},
		{name: "missing", cap: missing, want: http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/storage/munki/icons/1/icon.png?cap="+tc.cap, nil)
			router.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d; body = %q", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestBlobPutWritesAndRejectsWrongOperation(t *testing.T) {
	t.Parallel()
	store := newTestFileStore(t)
	router := newBlobTestRouter(store)
	key := "munki/icons/7/icon.png"
	putToken := signBlobCapability(t, storage.BlobCapabilityClaims{
		Op:          capability.OpPut,
		Key:         key,
		Exp:         time.Now().Add(time.Minute).Unix(),
		ContentType: "image/png",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/storage/munki/icons/7/icon.png?cap="+putToken,
		bytes.NewReader([]byte("png bytes")),
	)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	reader, _, err := store.Open(t.Context(), key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != "png bytes" {
		t.Fatalf("stored bytes = %q, want png bytes", got)
	}

	getToken := signBlobCapability(t, storage.BlobCapabilityClaims{
		Op:  capability.OpGet,
		Key: key,
		Exp: time.Now().Add(time.Minute).Unix(),
	})
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(
		http.MethodPut,
		"/storage/munki/icons/7/icon.png?cap="+getToken,
		strings.NewReader("wrong"),
	)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong op status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestBlobRejectsMismatchedPathAndSignedKey(t *testing.T) {
	t.Parallel()
	store := newTestFileStore(t)
	router := newBlobTestRouter(store)
	token := signBlobCapability(t, storage.BlobCapabilityClaims{
		Op:  capability.OpGet,
		Key: "munki/icons/7/icon.png",
		Exp: time.Now().Add(time.Minute).Unix(),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/storage/munki/icons/8/icon.png?cap="+token, nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestBlobGetFailsWhenStoreCannotSeek(t *testing.T) {
	t.Parallel()
	router := newBlobTestRouter(nonSeekStore{})
	token := signBlobCapability(t, storage.BlobCapabilityClaims{
		Op:  capability.OpGet,
		Key: "munki/packages/1/Installer.pkg",
		Exp: time.Now().Add(time.Minute).Unix(),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/storage/munki/packages/1/Installer.pkg?cap="+token, nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestBlobGetLogsInternalServeFailures(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	router := chi.NewRouter()
	RegisterBlobStorage(router, nonSeekStore{}, testCapabilityKey, logger)
	token := signBlobCapability(t, storage.BlobCapabilityClaims{
		Op:  capability.OpGet,
		Key: "munki/packages/1/Installer.pkg",
		Exp: time.Now().Add(time.Minute).Unix(),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/storage/munki/packages/1/Installer.pkg?cap="+token, nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	line := logs.String()
	for _, want := range []string{
		`"msg":"storage blob handler failed"`,
		`"operation":"get-storage-object"`,
		`"status":500`,
		`"key":"munki/packages/1/Installer.pkg"`,
		`"err":"storage object reader is not seekable"`,
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("log line %q does not contain %s", line, want)
		}
	}
}

func newBlobTestRouter(store storage.Store) chi.Router {
	r := chi.NewRouter()
	RegisterBlobStorage(r, store, testCapabilityKey, discardLogger())
	return r
}

func signBlobCapability(t *testing.T, claims storage.BlobCapabilityClaims) string {
	t.Helper()
	token, err := capability.Sign(testCapabilityKey, claims)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	return token
}

func newTestFileStore(t *testing.T) storage.Backend {
	t.Helper()
	store, err := storage.New(t.Context(), storage.Config{
		Kind:          storage.KindFile,
		FileRoot:      t.TempDir(),
		PublicURL:     "https://woodstar.example",
		CapabilityKey: testCapabilityKey,
		PresignTTL:    time.Minute,
	})
	if err != nil {
		t.Fatalf("new storage backend: %v", err)
	}
	return store
}

type nonSeekStore struct{}

func (nonSeekStore) Open(_ context.Context, _ string) (io.ReadCloser, storage.ObjectInfo, error) {
	return io.NopCloser(strings.NewReader("bytes")), storage.ObjectInfo{}, nil
}

func (nonSeekStore) Put(context.Context, string, io.Reader, storage.PutOptions) error {
	return errors.New("unexpected put")
}

func (nonSeekStore) Delete(context.Context, string) error {
	return nil
}

func (nonSeekStore) Stat(context.Context, string) (storage.ObjectInfo, error) {
	return storage.ObjectInfo{}, nil
}

var testCapabilityKey = []byte("storage capability test key")
