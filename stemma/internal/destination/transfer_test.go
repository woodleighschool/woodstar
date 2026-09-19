package destination

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/woodleighschool/stemma/plugin"
)

func TestFinalizationPreservesCallerDeadlineAndCancellation(t *testing.T) {
	remote, err := New(Config{URL: "https://woodstar.test", APIKey: "synthetic-key"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = remote.Close() }()
	deadline := time.Now().Add(30 * time.Minute)
	ctx, cancel := context.WithDeadline(t.Context(), deadline)
	defer cancel()
	var attempts int
	remote.api.SetTransport(roundTripFunc(func(request *http.Request) (*http.Response, error) {
		attempts++
		if got, ok := request.Context().Deadline(); !ok || !got.Equal(deadline) {
			t.Errorf("finalization deadline=%v, want %v", got, deadline)
		}
		cancel()
		return nil, request.Context().Err()
	}))
	if err := remote.finalizeUpload(ctx, plugin.Artifact{}, 42); !errors.Is(err, context.Canceled) || attempts != 1 {
		t.Fatalf("finalization attempts=%d: %v", attempts, err)
	}
}

func TestUploadCancellationStopsHashing(t *testing.T) {
	file := filepath.Join(t.TempDir(), "Example.pkg")
	if err := os.WriteFile(file, []byte("body"), 0o600); err != nil {
		t.Fatal(err)
	}
	remote, err := New(Config{URL: "https://woodstar.test", APIKey: "synthetic-key"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = remote.Close() }()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	artifact := plugin.Artifact{Path: file, Filename: "Example.pkg", Size: 4, SHA256: strings.Repeat("0", 64)}
	if _, err := remote.Upload(ctx, artifact); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled hashing reached digest verification: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

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
	var remote *Client
	var origin string
	remote, origin = testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	var logs bytes.Buffer
	ctx := plugin.WithLogger(t.Context(), slog.New(slog.NewJSONHandler(&logs, nil)))
	id, err := remote.Upload(ctx, plugin.Artifact{Path: file.Name(), Filename: "Example.pkg", Size: size, SHA256: digest})
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 || partAttempts[0].Load() != 2 || partAttempts[1].Load() != 1 || completions.Load() != 2 || finalizations.Load() != 1 || cleanups.Load() != 0 {
		t.Fatalf("id=%d attempts=%d/%d completions=%d finalizations=%d cleanups=%d", id, partAttempts[0].Load(), partAttempts[1].Load(), completions.Load(), finalizations.Load(), cleanups.Load())
	}
	// Progress spans every part, and a retried part must not count twice.
	if last := lastProgress(t, &logs, size); !last.Final || last.Current != size {
		t.Fatalf("installer progress ended at %+v", last)
	}
}

type progressRecord struct {
	Progress bool  `json:"progress"`
	Current  int64 `json:"current"`
	Total    int64 `json:"total"`
	Final    bool  `json:"progress_final"`
}

// lastProgress returns the final progress record after checking that none
// counts beyond the installer.
func lastProgress(t *testing.T, logs io.Reader, size int64) progressRecord {
	t.Helper()
	var last progressRecord
	for decoder := json.NewDecoder(logs); decoder.More(); {
		var record progressRecord
		if err := decoder.Decode(&record); err != nil {
			t.Fatal(err)
		}
		if !record.Progress {
			continue
		}
		if record.Current > size || record.Total != size {
			t.Fatalf("installer progress: %+v", record)
		}
		last = record
	}
	return last
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

func TestFailedUploadReleasesItsReservation(t *testing.T) {
	for _, test := range []struct{ name, strategy, want string }{
		{"multipart part without ETag", "multipart", "ETag"},
		// Finalization belongs to the upload: no later run resumes the object.
		{"finalized content differs", "direct-put", "does not match"},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "Example.pkg")
			if err := os.WriteFile(file, []byte("body"), 0o600); err != nil {
				t.Fatal(err)
			}
			var cleaned atomic.Bool
			var remote *Client
			var origin string
			remote, origin = testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				target := uploadTarget{URL: origin + "/transfer", Method: http.MethodPut}
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/api/munki/package-installers":
					_ = json.NewEncoder(w).Encode(map[string]any{"object_id": 42, "upload": map[string]any{"strategy": test.strategy, "target": target}})
				case r.Method == http.MethodPost && r.URL.Path == "/api/munki/package-installers/42/multipart/parts/1":
					_ = json.NewEncoder(w).Encode(target)
				case r.Method == http.MethodPut && r.URL.Path == "/transfer":
					w.WriteHeader(http.StatusOK)
				case r.Method == http.MethodPut && r.URL.Path == "/api/munki/package-installers/42":
					_, _ = io.WriteString(w, `{"id":42,"size_bytes":4,"sha256":"`+strings.Repeat("0", 64)+`"}`)
				case r.Method == http.MethodDelete && r.URL.Path == "/api/munki/package-installers/42":
					cleaned.Store(true)
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			digest := sha256.Sum256([]byte("body"))
			_, err := remote.Upload(t.Context(), plugin.Artifact{Path: file, Filename: "Example.pkg", Size: 4, SHA256: hex.EncodeToString(digest[:])})
			if err == nil || !strings.Contains(err.Error(), test.want) || !cleaned.Load() {
				t.Fatalf("cleaned=%t err=%v", cleaned.Load(), err)
			}
		})
	}
}

func TestTransferRejectsRedirectsAndRedactsTargets(t *testing.T) {
	var attempts atomic.Int32
	remote, origin := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		http.Redirect(w, r, "/redirected", http.StatusTemporaryRedirect)
	}))
	_, err := remote.transferBytes(t.Context(), uploadTarget{URL: origin + "/signed?token=private", Method: http.MethodPut}, strings.NewReader("body"), 4)
	if err == nil || strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), origin) || attempts.Load() != 1 {
		t.Fatalf("redirect attempts=%d err=%v", attempts.Load(), err)
	}
}

func TestTransferHonorsRetryAfterAndCancellation(t *testing.T) {
	for _, retryAfter := range []string{"3600", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)} {
		t.Run(retryAfter, func(t *testing.T) {
			var attempts atomic.Int32
			remote, origin := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempts.Add(1)
				w.Header().Set("Retry-After", retryAfter)
				w.WriteHeader(http.StatusTooManyRequests)
			}))
			ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
			defer cancel()
			_, err := remote.transferBytes(ctx, uploadTarget{URL: origin + "/signed?token=private", Method: http.MethodPut}, strings.NewReader("body"), 4)
			if !errors.Is(err, context.DeadlineExceeded) || attempts.Load() != 1 {
				t.Fatalf("rate-limited transfer attempts=%d err=%v", attempts.Load(), err)
			}
		})
	}
}
