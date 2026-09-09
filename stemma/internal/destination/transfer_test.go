package destination

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/woodleighschool/stemma/plugin"
)

func TestMultipartUploadRetriesWithoutChangingContent(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "Example.pkg"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	const partSize int64 = 64 << 20
	const size = partSize + 1
	if err := file.Truncate(size); err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		t.Fatal(err)
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	var partDigests [2]string
	for i := range partDigests {
		hash.Reset()
		if _, err := io.Copy(hash, io.NewSectionReader(file, int64(i)*partSize, min(partSize, size-int64(i)*partSize))); err != nil {
			t.Fatal(err)
		}
		partDigests[i] = hex.EncodeToString(hash.Sum(nil))
	}
	var partAttempts [2]atomic.Int32
	var completions, finalizations, cleanups atomic.Int32
	transfer := multipartPartHandler(t, partSize, partDigests, &partAttempts)
	complete := multipartCompletionHandler(t, &completions)
	var origin string
	remote := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		write := func(value any) {
			if err := json.NewEncoder(w).Encode(value); err != nil {
				t.Error(err)
			}
		}
		if strings.HasPrefix(r.URL.Path, "/transfer/") {
			transfer(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer synthetic-key" {
			t.Error("API request has no bearer token")
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/munki/package-installers":
			write(map[string]any{"object_id": 42, "upload": map[string]any{"strategy": "multipart"}})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/munki/package-installers/42/multipart/parts/"):
			part := filepath.Base(r.URL.Path)
			write(uploadTarget{URL: origin + "/transfer/" + part, Method: http.MethodPut, Headers: map[string]string{"X-Upload-Token": "signed-part"}})
		case r.Method == http.MethodPut && r.URL.Path == "/api/munki/package-installers/42/multipart":
			complete(w, r)
		case r.Method == http.MethodPut && r.URL.Path == "/api/munki/package-installers/42":
			finalizations.Add(1)
			write(map[string]any{"id": 42, "size_bytes": size, "sha256": digest})
		case r.Method == http.MethodDelete:
			cleanups.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	origin = remote.config.URL
	id, err := remote.upload(t.Context(), plugin.Artifact{Path: file.Name(), Filename: "Example.pkg", Size: size, SHA256: digest})
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 || partAttempts[0].Load() != 2 || partAttempts[1].Load() != 1 || completions.Load() != 2 || finalizations.Load() != 1 || cleanups.Load() != 0 {
		t.Fatalf("id=%d attempts=%d/%d completions=%d finalizations=%d cleanups=%d", id, partAttempts[0].Load(), partAttempts[1].Load(), completions.Load(), finalizations.Load(), cleanups.Load())
	}
}

func multipartPartHandler(t *testing.T, partSize int64, digests [2]string, attempts *[2]atomic.Int32) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("X-Upload-Token") != "signed-part" || r.Method != http.MethodPut {
			t.Error("signed transfer received incorrect method or credentials")
		}
		part, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/transfer/"))
		if err != nil || part < 1 || part > 2 {
			t.Error("unexpected multipart number")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		hash := sha256.New()
		count, err := io.Copy(hash, r.Body)
		want := partSize
		if part == 2 {
			want = 1
		}
		if err != nil || count != want || r.ContentLength != want {
			t.Errorf("part %d content length=%d read=%d want=%d: %v", part, r.ContentLength, count, want, err)
		}
		if hex.EncodeToString(hash.Sum(nil)) != digests[part-1] {
			t.Errorf("part %d content changed", part)
		}
		if attempts[part-1].Add(1) == 1 && part == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("ETag", "etag-"+strconv.Itoa(part))
	}
}

func multipartCompletionHandler(t *testing.T, completions *atomic.Int32) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Parts []struct {
				PartNumber int    `json:"part_number"`
				ETag       string `json:"etag"`
			} `json:"parts"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Parts) != 2 {
			t.Errorf("invalid multipart completion: %+v %v", body, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		for i, part := range body.Parts {
			if part.PartNumber != i+1 || part.ETag != "etag-"+strconv.Itoa(i+1) {
				t.Errorf("invalid completed part: %+v", part)
			}
		}
		if completions.Add(1) == 1 {
			connection, _, err := http.NewResponseController(w).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			_ = connection.Close()
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func TestAPICreateDoesNotRetry(t *testing.T) {
	var attempts atomic.Int32
	remote := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	if err := remote.request(t.Context(), http.MethodPost, "/api/munki/software", map[string]any{"name": "Example"}, nil); err == nil || attempts.Load() != 1 {
		t.Fatalf("non-idempotent create attempts=%d err=%v", attempts.Load(), err)
	}
}

func TestMultipartUploadWithoutETagCleansUp(t *testing.T) {
	file := filepath.Join(t.TempDir(), "Example.pkg")
	if err := os.WriteFile(file, []byte("body"), 0o600); err != nil {
		t.Fatal(err)
	}
	var cleaned atomic.Bool
	var origin string
	remote := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/munki/package-installers":
			_, _ = io.WriteString(w, `{"object_id":42,"upload":{"strategy":"multipart"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/munki/package-installers/42/multipart/parts/1":
			_ = json.NewEncoder(w).Encode(uploadTarget{URL: origin + "/transfer", Method: http.MethodPut})
		case r.Method == http.MethodPut && r.URL.Path == "/transfer":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/munki/package-installers/42":
			cleaned.Store(true)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request after missing ETag: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	origin = remote.config.URL
	digest := sha256.Sum256([]byte("body"))
	_, err := remote.upload(t.Context(), plugin.Artifact{Path: file, Filename: "Example.pkg", Size: 4, SHA256: hex.EncodeToString(digest[:])})
	if err == nil || !strings.Contains(err.Error(), "ETag") || !cleaned.Load() {
		t.Fatalf("missing ETag: cleaned=%t err=%v", cleaned.Load(), err)
	}
}

func TestAPIResponsePreservesLargeIDs(t *testing.T) {
	remote := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":9007199254740993}`)
	}))
	var result map[string]any
	if err := remote.request(t.Context(), http.MethodGet, "/api/munki/software/9007199254740993", nil, &result); err != nil {
		t.Fatal(err)
	}
	if number(result["id"]) != 9007199254740993 {
		t.Fatalf("ID lost precision: %v", result)
	}
}

func TestTransferRejectsRedirectsAndRedactsTargets(t *testing.T) {
	var attempts atomic.Int32
	remote := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		http.Redirect(w, r, "/redirected", http.StatusTemporaryRedirect)
	}))
	_, err := remote.transferBytes(t.Context(), uploadTarget{URL: remote.config.URL + "/signed?token=private", Method: http.MethodPut}, strings.NewReader("body"), 4)
	if err == nil || strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), remote.config.URL) || attempts.Load() != 1 {
		t.Fatalf("redirect attempts=%d err=%v", attempts.Load(), err)
	}
}

func TestTransferHonorsRetryAfterAndCancellation(t *testing.T) {
	for _, retryAfter := range []string{"3600", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)} {
		t.Run(retryAfter, func(t *testing.T) {
			var attempts atomic.Int32
			remote := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempts.Add(1)
				w.Header().Set("Retry-After", retryAfter)
				w.WriteHeader(http.StatusTooManyRequests)
			}))
			ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
			defer cancel()
			_, err := remote.transferBytes(ctx, uploadTarget{URL: remote.config.URL + "/signed?token=private", Method: http.MethodPut}, strings.NewReader("body"), 4)
			if !errors.Is(err, context.DeadlineExceeded) || attempts.Load() != 1 {
				t.Fatalf("rate-limited transfer attempts=%d err=%v", attempts.Load(), err)
			}
		})
	}
}

func testClient(t *testing.T, handler http.Handler) *client {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	remote, err := newClient(config{URL: server.URL, APIKey: "synthetic-key", CAFile: caPath})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = remote.api.Close()
		_ = remote.transfer.Close()
	})
	return remote
}
