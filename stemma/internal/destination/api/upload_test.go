package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/woodleighschool/goodies/bloby"
	"github.com/woodleighschool/stemma/plugin"
)

// Multipart parts are 64 MiB until an installer needs more than 10,000 of them.
const partSize int64 = 64 << 20

const (
	installersPath = "/api/munki/package-installers"
	installerPath  = installersPath + "/42"
	partsPath      = installerPath + "/multipart/parts/"
)

func TestUploadDeclaresTheArtifactAndSendsTheSignedHeaders(t *testing.T) {
	artifact := leasedInstaller(t, []byte("synthetic installer bytes"))
	want, err := bloby.Digest(strings.NewReader("synthetic installer bytes"))
	if err != nil {
		t.Fatal(err)
	}
	repo, remote := serveRepository(t, "direct-put")
	var logs bytes.Buffer
	ctx := plugin.WithLogger(t.Context(), slog.New(slog.NewJSONHandler(&logs, nil)))
	id, err := remote.Upload(ctx, artifact)
	if err != nil {
		t.Fatal(err)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	// Storage keeps only a body sent with the signed header that matches the
	// declaration, so a stored object is the artifact.
	if repo.declared.Content != want || repo.declared.Filename != "Example.pkg" || repo.stored["object"] != artifact.Size {
		t.Fatalf("declared %+v stored %v, want %+v", repo.declared, repo.stored, want)
	}
	if id != 42 || repo.requests[http.MethodPut+" "+installerPath] != 1 || repo.requests[http.MethodDelete+" "+installerPath] != 0 {
		t.Fatalf("id=%d requests=%v", id, repo.requests)
	}
	if last := lastProgress(t, &logs, artifact.Size); !last.Final || last.Current != artifact.Size {
		t.Fatalf("installer progress ended at %+v", last)
	}
}

func TestMultipartUploadSignsEveryPartAndSendsFourAtATime(t *testing.T) {
	const parts = 6
	artifact := leasedParts(t, parts)
	synctest.Test(t, func(t *testing.T) {
		repo := newRepository(t, "multipart", "https://woodstar.test")
		remote := New(Config{URL: repo.origin, APIKey: "synthetic-key"})
		defer func() { _ = remote.Close() }()
		remote.api.SetTransport(handlerTransport{repo})
		remote.transfer.SetTransport(handlerTransport{repo})
		// Storage holds each part it is sent until the test lets them go.
		release := make(chan struct{})
		repo.answer = func(_ http.ResponseWriter, r *http.Request) bool {
			if strings.HasPrefix(r.URL.Path, "/storage/") {
				<-release
			}
			return false
		}
		done := make(chan error, 1)
		go func() {
			_, err := remote.Upload(t.Context(), artifact)
			done <- err
		}()
		// Once the upload can do nothing more, the parts storage holds are all
		// it will send at once.
		synctest.Wait()
		if held := repo.transfers(); held != 4 {
			t.Errorf("%d parts were in flight at once, want 4", held)
		}
		close(release)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		file, err := os.Open(artifact.Path)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = file.Close() }()
		repo.mu.Lock()
		defer repo.mu.Unlock()
		for part := range parts {
			name := strconv.Itoa(part + 1)
			offset := int64(part) * partSize
			length := min(partSize, artifact.Size-offset)
			want, err := bloby.CRC64NVME(io.NewSectionReader(file, offset, length))
			if err != nil {
				t.Fatal(err)
			}
			// Storage accepted the part under the checksum it was signed with.
			if repo.signed[name] != want || repo.stored[name] != length {
				t.Errorf("part %s signed %q and stored %d bytes, want %q and %d", name, repo.signed[name], repo.stored[name], want, length)
			}
		}
		if len(repo.stored) != parts || repo.peak != 4 || repo.requests[http.MethodPut+" "+installerPath] != 1 {
			t.Fatalf("parts=%d peak=%d requests=%v", len(repo.stored), repo.peak, repo.requests)
		}
	})
}

func TestMultipartUploadRepeatsFailedSigningAndTransfer(t *testing.T) {
	artifact := leasedParts(t, 3)
	repo, remote := serveRepository(t, "multipart")
	failed := []string{
		http.MethodPost + " " + partsPath + "1",
		http.MethodPost + " " + partsPath + "2",
		http.MethodPut + " /storage/1",
		http.MethodPut + " /storage/2",
	}
	repo.answer = func(w http.ResponseWriter, r *http.Request) bool {
		if repo.count(r) != 1 || !slices.Contains(failed, r.Method+" "+r.URL.Path) {
			return false
		}
		switch r.URL.Path {
		case partsPath + "1":
			dropConnection(t, w)
			return true
		case "/storage/1":
			// Storage fails after the whole part arrived. The other part fails
			// while it is still being sent.
			_, _ = io.Copy(io.Discard, r.Body)
		}
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusServiceUnavailable)
		return true
	}
	var logs bytes.Buffer
	ctx := plugin.WithLogger(t.Context(), slog.New(slog.NewJSONHandler(&logs, nil)))
	if _, err := remote.Upload(ctx, artifact); err != nil {
		t.Fatal(err)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, request := range failed {
		if repo.requests[request] != 2 {
			t.Errorf("%s was sent %d times, want 2", request, repo.requests[request])
		}
	}
	// Storage keeps a part only when its bytes match the checksum it was signed
	// with, so each part that was sent again arrived whole.
	if repo.stored["1"] != partSize || repo.stored["2"] != partSize || repo.stored["3"] != 1 || repo.requests[http.MethodDelete+" "+installerPath] != 0 {
		t.Fatalf("stored=%v requests=%v", repo.stored, repo.requests)
	}
	// A part that is sent again counts once.
	if last := lastProgress(t, &logs, artifact.Size); !last.Final || last.Current != artifact.Size {
		t.Fatalf("installer progress ended at %+v", last)
	}
}

func TestRefusedPartStopsTheUploadAndReleasesIt(t *testing.T) {
	artifact := leasedParts(t, 2)
	repo, remote := serveRepository(t, "multipart")
	arrived := make(chan struct{})
	sibling := make(chan int64, 1)
	repo.answer = func(w http.ResponseWriter, r *http.Request) bool {
		switch r.URL.Path {
		case "/storage/1":
			close(arrived)
			n, _ := io.Copy(io.Discard, r.Body)
			sibling <- n
		case "/storage/2":
			<-arrived
			w.WriteHeader(http.StatusBadRequest)
		default:
			return false
		}
		return true
	}
	_, err := remote.Upload(t.Context(), artifact)
	if err == nil || !strings.Contains(err.Error(), "HTTP 400") || strings.Contains(err.Error(), repo.origin) {
		t.Fatalf("refused part: %v", err)
	}
	if sent := <-sibling; sent == partSize {
		t.Error("the other part was sent in full after the upload failed")
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.requests[http.MethodPut+" "+installerPath] != 0 || repo.requests[http.MethodDelete+" "+installerPath] != 1 {
		t.Fatalf("requests=%v", repo.requests)
	}
}

func TestRefusedFinalizationFailsTheUpload(t *testing.T) {
	repo, remote := serveRepository(t, "direct-put")
	// The repository releases an upload it refuses to publish, so the release
	// that follows finds nothing.
	repo.answer = func(w http.ResponseWriter, r *http.Request) bool {
		switch r.Method + " " + r.URL.Path {
		case http.MethodPut + " " + installerPath:
			w.WriteHeader(http.StatusBadRequest)
		case http.MethodDelete + " " + installerPath:
			w.WriteHeader(http.StatusNotFound)
		default:
			return false
		}
		return true
	}
	_, err := remote.Upload(t.Context(), leasedInstaller(t, []byte("synthetic installer bytes")))
	if status, ok := errors.AsType[StatusError](err); !ok || status.Status != http.StatusBadRequest {
		t.Fatalf("refused finalization: %v", err)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.requests[http.MethodPut+" "+installerPath] != 1 || repo.requests[http.MethodDelete+" "+installerPath] != 1 {
		t.Fatalf("requests=%v", repo.requests)
	}
}

func TestCanceledUploadStopsAndReleasesItsReservation(t *testing.T) {
	for _, test := range []struct{ name, strategy, during string }{
		{"direct transfer", "direct-put", http.MethodPut + " /storage/object"},
		{"part transfer", "multipart", http.MethodPut + " /storage/2"},
		{"part signing", "multipart", http.MethodPost + " " + partsPath + "1"},
		{"finalization", "direct-put", http.MethodPut + " " + installerPath},
	} {
		t.Run(test.name, func(t *testing.T) {
			artifact := leasedInstaller(t, []byte("synthetic installer bytes"))
			if test.strategy == "multipart" {
				artifact = leasedParts(t, 2)
			}
			repo, remote := serveRepository(t, test.strategy)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			repo.answer = func(w http.ResponseWriter, r *http.Request) bool {
				if r.Method+" "+r.URL.Path != test.during {
					return false
				}
				cancel()
				_, _ = io.Copy(io.Discard, r.Body)
				// A request that could be repeated must not be once the run is canceled.
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusServiceUnavailable)
				return true
			}
			_, err := remote.Upload(ctx, artifact)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled upload: %v", err)
			}
			repo.mu.Lock()
			defer repo.mu.Unlock()
			// The release outlives the cancellation that caused it.
			if repo.requests[test.during] != 1 || repo.requests[http.MethodDelete+" "+installerPath] != 1 {
				t.Fatalf("requests=%v", repo.requests)
			}
			if test.during != http.MethodPut+" "+installerPath && repo.requests[http.MethodPut+" "+installerPath] != 0 {
				t.Fatal("finalized an upload that was canceled")
			}
		})
	}
}

func TestUploadCancellationStopsHashing(t *testing.T) {
	artifact := leasedInstaller(t, []byte("body"))
	artifact.SHA256 = strings.Repeat("0", 64)
	remote := New(Config{URL: "https://woodstar.test", APIKey: "synthetic-key"})
	defer func() { _ = remote.Close() }()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := remote.Upload(ctx, artifact); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled hashing reached digest verification: %v", err)
	}
}

func TestUploadRejectsAnArtifactThatDiffersFromItsDigest(t *testing.T) {
	artifact := leasedInstaller(t, []byte("synthetic installer bytes"))
	artifact.SHA256 = strings.Repeat("0", 64)
	repo, remote := serveRepository(t, "direct-put")
	_, err := remote.Upload(t.Context(), artifact)
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if err == nil || !strings.Contains(err.Error(), "digest changed") || len(repo.requests) != 0 {
		t.Fatalf("requests=%v err=%v", repo.requests, err)
	}
}

func TestTransferRejectsRedirectsAndRedactsTargets(t *testing.T) {
	var attempts atomic.Int32
	remote, origin := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		http.Redirect(w, r, "/redirected", http.StatusTemporaryRedirect)
	}))
	err := remote.transferBytes(t.Context(), uploadTarget{URL: origin + "/signed?token=private", Method: http.MethodPut}, func() io.ReadSeeker { return strings.NewReader("body") }, 4)
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
			err := remote.transferBytes(ctx, uploadTarget{URL: origin + "/signed?token=private", Method: http.MethodPut}, func() io.ReadSeeker { return strings.NewReader("body") }, 4)
			if !errors.Is(err, context.DeadlineExceeded) || attempts.Load() != 1 {
				t.Fatalf("rate-limited transfer attempts=%d err=%v", attempts.Load(), err)
			}
		})
	}
}

// repository is the upload contract as the server and its storage keep it. It
// reserves object 42 for declared content, signs each part for the checksum it
// is given, and stores only a body that matches what its target was signed for.
type repository struct {
	t        *testing.T
	strategy string
	origin   string
	// answer lets a test reply to a request in the repository's place. It
	// reports whether it did.
	answer func(http.ResponseWriter, *http.Request) bool

	mu           sync.Mutex
	declared     declaredUpload
	signed       map[string]string // The checksum each part number was signed with.
	stored       map[string]int64  // Bytes stored under each target.
	requests     map[string]int    // Times each method and path was requested.
	active, peak int               // Transfers storage is receiving, and the most at once.
}

func newRepository(t *testing.T, strategy, origin string) *repository {
	return &repository{t: t, strategy: strategy, origin: origin, signed: map[string]string{}, stored: map[string]int64{}, requests: map[string]int{}}
}

// serveRepository starts a repository that answers uploads with strategy, and
// a client connected to it.
func serveRepository(t *testing.T, strategy string) (*repository, *Client) {
	t.Helper()
	repo := newRepository(t, strategy, "")
	remote, origin := testClient(t, repo)
	repo.origin = origin
	return repo, remote
}

// handlerTransport answers requests from a handler in the caller's goroutine.
// Without a network, a test can wait for an upload to come to rest.
type handlerTransport struct{ handler http.Handler }

func (transport handlerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	transport.handler.ServeHTTP(recorder, r)
	response := recorder.Result()
	response.Request = r
	return response, nil
}

// count returns how many times r's method and path have been requested,
// r included.
func (repo *repository) count(r *http.Request) int {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	return repo.requests[r.Method+" "+r.URL.Path]
}

func (repo *repository) transfers() int {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	return repo.active
}

func (repo *repository) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	transfer := strings.HasPrefix(r.URL.Path, "/storage/")
	repo.mu.Lock()
	repo.requests[r.Method+" "+r.URL.Path]++
	if transfer {
		repo.active++
		repo.peak = max(repo.peak, repo.active)
	}
	repo.mu.Unlock()
	if transfer {
		defer func() {
			repo.mu.Lock()
			repo.active--
			repo.mu.Unlock()
		}()
	}
	// Signed targets authorize themselves and never see the API key.
	if authorization := r.Header.Get("Authorization"); transfer == (authorization != "") || !transfer && authorization != "Bearer synthetic-key" {
		repo.t.Errorf("%s %s carried Authorization %q", r.Method, r.URL.Path, authorization)
	}
	if repo.answer != nil && repo.answer(w, r) {
		return
	}
	switch {
	case transfer && r.Method == http.MethodPut:
		repo.store(w, r)
	case r.Method == http.MethodPost && r.URL.Path == installersPath:
		repo.reserve(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, partsPath):
		repo.sign(w, r)
	case r.Method == http.MethodPut && r.URL.Path == installerPath:
		repo.finalize(w)
	case r.Method == http.MethodDelete && r.URL.Path == installerPath:
		w.WriteHeader(http.StatusNoContent)
	default:
		repo.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func (repo *repository) reserve(w http.ResponseWriter, r *http.Request) {
	var declared declaredUpload
	if !decodeStrict(r.Body, &declared) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	repo.mu.Lock()
	repo.declared = declared
	repo.mu.Unlock()
	upload := map[string]any{"strategy": repo.strategy}
	if repo.strategy == "direct-put" {
		upload["target"] = uploadTarget{URL: repo.origin + "/storage/object", Method: http.MethodPut, Headers: map[string]string{"X-Checksum-Sha256": declared.SHA256}}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"object_id": 42, "upload": upload})
}

func (repo *repository) sign(w http.ResponseWriter, r *http.Request) {
	var part struct {
		CRC64NVME string `json:"crc64nvme"`
	}
	number := strings.TrimPrefix(r.URL.Path, partsPath)
	if !decodeStrict(r.Body, &part) || part.CRC64NVME == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	repo.mu.Lock()
	repo.signed[number] = part.CRC64NVME
	repo.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(uploadTarget{URL: repo.origin + "/storage/" + number, Method: http.MethodPut, Headers: map[string]string{"X-Checksum-Crc64nvme": part.CRC64NVME}})
}

// store keeps a body that has the length the request announced and the
// checksum its target was signed with, sent in the header the target named.
func (repo *repository) store(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/storage/")
	content, err := bloby.Digest(r.Body)
	repo.mu.Lock()
	defer repo.mu.Unlock()
	signed := r.Header.Get("X-Checksum-Crc64nvme") == repo.signed[name] && content.CRC64NVME == repo.signed[name]
	if name == "object" {
		signed = r.Header.Get("X-Checksum-Sha256") == repo.declared.SHA256 && content == repo.declared.Content
	}
	if err != nil || r.ContentLength != content.SizeBytes || !signed {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	repo.stored[name] = content.SizeBytes
}

// finalize publishes an upload whose stored bytes total the declared size.
func (repo *repository) finalize(w http.ResponseWriter) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	var total int64
	for _, size := range repo.stored {
		total += size
	}
	if len(repo.stored) == 0 || total != repo.declared.SizeBytes {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "filename": repo.declared.Filename, "size_bytes": repo.declared.SizeBytes, "sha256": repo.declared.SHA256})
}

func decodeStrict(body io.Reader, value any) bool {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(value) == nil
}

// dropConnection ends a request without a reply.
func dropConnection(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	connection, _, err := http.NewResponseController(w).Hijack()
	if err != nil {
		t.Error(err)
		return
	}
	_ = connection.Close()
}

// leasedInstaller writes content as a prepared installer.
func leasedInstaller(t *testing.T, content []byte) plugin.Artifact {
	t.Helper()
	path := filepath.Join(t.TempDir(), "Example.pkg")
	if err := os.WriteFile(path, content, 0o400); err != nil {
		t.Fatal(err)
	}
	return leased(t, path)
}

// leasedParts writes a sparse installer that uploads as count parts, the last
// of one byte. Each part begins with its own number, so no two share a
// checksum.
func leasedParts(t *testing.T, count int) plugin.Artifact {
	t.Helper()
	path := filepath.Join(t.TempDir(), "Example.pkg")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if err := file.Truncate(int64(count-1)*partSize + 1); err != nil {
		t.Fatal(err)
	}
	for part := range count - 1 {
		if _, err := file.WriteAt(binary.BigEndian.AppendUint64(nil, uint64(part+1)), int64(part)*partSize); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := file.WriteAt([]byte{byte(count)}, int64(count-1)*partSize); err != nil {
		t.Fatal(err)
	}
	return leased(t, path)
}

func leased(t *testing.T, path string) plugin.Artifact {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	content, err := bloby.Digest(file)
	if err != nil {
		t.Fatal(err)
	}
	return plugin.Artifact{Path: path, Filename: filepath.Base(path), Size: content.SizeBytes, SHA256: content.SHA256}
}

type progressRecord struct {
	Progress bool  `json:"progress"`
	Current  int64 `json:"current"`
	Total    int64 `json:"total"`
	Final    bool  `json:"progress_final"`
}

// lastProgress returns the final progress record after checking that none
// counts beyond the installer and that only the last one is final.
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
		if record.Current > size || record.Total != size || last.Final {
			t.Fatalf("installer progress: %+v after %+v", record, last)
		}
		last = record
	}
	return last
}

func TestUploadStagesMeasureCheckingTransferAndFinalizationSeparately(t *testing.T) {
	artifact := leasedInstaller(t, []byte("synthetic installer bytes"))
	for _, rejected := range []bool{false, true} {
		t.Run(strconv.FormatBool(rejected), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				repo := newRepository(t, "direct-put", "https://woodstar.test")
				remote := New(Config{URL: repo.origin, APIKey: "synthetic-key"})
				defer func() { _ = remote.Close() }()
				remote.api.SetTransport(handlerTransport{repo})
				remote.transfer.SetTransport(handlerTransport{repo})
				repo.answer = func(w http.ResponseWriter, r *http.Request) bool {
					switch {
					case r.Method == http.MethodPost && r.URL.Path == installersPath:
						time.Sleep(3 * time.Second)
					case strings.HasPrefix(r.URL.Path, "/storage/"):
						time.Sleep(5 * time.Second)
					case r.Method == http.MethodPut && r.URL.Path == installerPath:
						time.Sleep(7 * time.Second)
						if rejected {
							w.WriteHeader(http.StatusBadRequest)
							return true
						}
					}
					return false
				}
				var logs bytes.Buffer
				ctx := plugin.WithLogger(t.Context(), slog.New(slog.NewJSONHandler(&logs, nil)))
				_, err := remote.Upload(ctx, artifact)
				if (err != nil) != rejected {
					t.Fatalf("upload error: %v", err)
				}
				var starts, ends []string
				durations := map[string]time.Duration{"Checking installer": 3 * time.Second, "Uploading installer": 5 * time.Second, "Finalizing upload": 7 * time.Second}
				decoder := json.NewDecoder(&logs)
				for decoder.More() {
					var record struct {
						Message string        `json:"msg"`
						Start   bool          `json:"stage"`
						End     bool          `json:"stage_result"`
						Elapsed time.Duration `json:"elapsed"`
						Error   string        `json:"error"`
					}
					if err := decoder.Decode(&record); err != nil {
						t.Fatal(err)
					}
					if record.Start {
						if len(starts) != len(ends) {
							t.Fatalf("%s started before the previous operation finished", record.Message)
						}
						starts = append(starts, record.Message)
					}
					if record.End {
						ends = append(ends, record.Message)
						if record.Elapsed != durations[record.Message] || (record.Error != "") != (rejected && record.Message == "Finalizing upload") {
							t.Fatalf("operation timing or outcome: %+v", record)
						}
					}
				}
				want := []string{"Checking installer", "Uploading installer", "Finalizing upload"}
				if !slices.Equal(starts, want) || !slices.Equal(ends, want) {
					t.Fatalf("starts=%v ends=%v", starts, ends)
				}
			})
		})
	}
}
