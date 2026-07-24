package worker

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func newTestWorker(t *testing.T, serverURL string) *Worker {
	t.Helper()
	w, err := New(Config{
		ServerURL:           serverURL,
		Key:                 "dp-key",
		DataDir:             t.TempDir(),
		DownloadConcurrency: 2,
	}, discardLogger())
	if err != nil {
		t.Fatalf("New worker: %v", err)
	}
	return w
}

func newTestSession(t *testing.T, serverURL string) *session {
	t.Helper()
	mirror, err := loadMirror(t.TempDir())
	if err != nil {
		t.Fatalf("loadMirror: %v", err)
	}
	client, err := newWoodstarClient(serverURL, "dp-key", "")
	if err != nil {
		t.Fatalf("new Woodstar client: %v", err)
	}
	return newSession(mirror, client, discardLogger(), 2, time.Millisecond)
}

// fakeWoodstar serves the worker's HTTP side: a per-job download URL that points
// back at its own storage route, which streams content with the given handler.
func fakeWoodstar(t *testing.T, storage http.HandlerFunc) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/munki/distribution/packages/", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"download_url": srv.URL + "/storage/blob"})
	})
	mux.HandleFunc("/storage/blob", storage)
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// waitEvent reads the session's event stream until the wanted type appears.
func waitEvent(t *testing.T, events <-chan packageEvent, want string) packageEvent {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case event := <-events:
			if event.Type == want {
				return event
			}
		case <-deadline:
			t.Fatalf("event %q not seen", want)
		}
	}
}

func TestSessionPrunesUndesiredMirror(t *testing.T) {
	content := []byte("chrome-installer-payload")
	sha := sha256Hex(content)
	size := int64(len(content))
	sess := newTestSession(t, "http://woodstar.invalid")
	path := sess.mirror.localPath(7, "Chrome.pkg")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("seed mirror file: %v", err)
	}
	sess.mirror.put(7, packageState{Filename: "Chrome.pkg", SHA256: sha, SizeBytes: size})

	sess.applyDesiredSet(t.Context(), nil)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("stale file still present: %v", err)
	}
	if _, ok := sess.mirror.get(7); ok {
		t.Fatal("pruned package still in mirror")
	}
}

func TestSessionRejectsCorruptDownload(t *testing.T) {
	srv := fakeWoodstar(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("corrupt"))
	})
	sess := newTestSession(t, srv.URL)

	sess.applyDesiredSet(t.Context(), []desiredPackage{
		{PackageID: 7, Filename: "Chrome.pkg", SHA256: sha256Hex([]byte("expected")), SizeBytes: 8},
	})

	if event := waitEvent(t, sess.events, eventPackageError); event.PackageID != 7 || event.Error == "" {
		t.Fatalf("error event = %+v, want package 7 with message", event)
	}

	if _, ok := sess.mirror.get(7); ok {
		t.Fatal("corrupt download was accepted into the mirror")
	}
	if _, err := os.Stat(sess.mirror.localPath(7, "Chrome.pkg")); !os.IsNotExist(err) {
		t.Fatalf("corrupt file left on disk: %v", err)
	}
}

func TestSessionRetriesUntilDownloadSucceeds(t *testing.T) {
	var attempts atomic.Int32
	srv := fakeWoodstar(t, func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) < 3 {
			_, _ = w.Write([]byte("corrupt"))
			return
		}
		_, _ = w.Write([]byte("expected"))
	})
	sess := newTestSession(t, srv.URL)

	sess.applyDesiredSet(t.Context(), []desiredPackage{
		{
			PackageID: 7,
			Filename:  "Chrome.pkg",
			SHA256:    sha256Hex([]byte("expected")),
			SizeBytes: int64(len("expected")),
		},
	})

	if event := waitEvent(t, sess.events, eventPackageCurrent); event.PackageID != 7 {
		t.Fatalf("current event = %+v, want package 7 after retries", event)
	}
	if _, ok := sess.mirror.get(7); !ok {
		t.Fatal("package not mirrored after a successful retry")
	}
}

func TestDownloadErrorRedactsPresignedCredentials(t *testing.T) {
	client := &woodstarClient{
		downloadHTTP: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		})},
	}
	const rawURL = "https://access:secret@storage.example/package.pkg?signature=private" //nolint:gosec // Credential-redaction fixture.
	err := client.download(t.Context(), rawURL, t.TempDir()+"/package.pkg")
	if err == nil {
		t.Fatal("download returned nil error")
	}
	message := err.Error()
	if strings.Contains(message, "access") || strings.Contains(message, "secret") ||
		strings.Contains(message, "signature") || strings.Contains(message, "private") {
		t.Fatalf("download error contains presigned credentials: %q", message)
	}
	if !strings.Contains(message, "https://storage.example/package.pkg") {
		t.Fatalf("download error = %q, want redacted target", message)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestConnectOnceReportsCurrentForMirroredPackage(t *testing.T) {
	content := []byte("installer-bytes-0123456789")
	sha := sha256Hex(content)
	size := int64(len(content))

	gotCurrent := make(chan packageEvent, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = ws.Close(websocket.StatusNormalClosure, "") }()
		send(t, r.Context(), ws, serverMessage{
			Type:              messageHello,
			DistributionPoint: pointIdentity{ID: 1, Name: "test"},
		})
		send(t, r.Context(), ws, serverMessage{
			Type: messageDesiredSet,
			Packages: []desiredPackage{
				{PackageID: 7, Filename: "Chrome.pkg", SHA256: sha, SizeBytes: size},
			},
		})
		for {
			_, raw, err := ws.Read(r.Context())
			if err != nil {
				return
			}
			var event packageEvent
			if err := json.Unmarshal(raw, &event); err == nil &&
				event.Type == eventPackageCurrent && event.PackageID == 7 {
				gotCurrent <- event
				return
			}
		}
	}))
	defer srv.Close()

	worker := newTestWorker(t, srv.URL)
	// Pre-seed the mirror so the desired package is already satisfied and the
	// worker reports current without needing a download.
	if err := os.WriteFile(worker.mirror.localPath(7, "Chrome.pkg"), content, 0o600); err != nil {
		t.Fatalf("seed mirror file: %v", err)
	}
	worker.mirror.put(7, packageState{Filename: "Chrome.pkg", SHA256: sha, SizeBytes: size})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = worker.connectOnce(ctx)
	}()

	select {
	case event := <-gotCurrent:
		if event.SHA256 != sha {
			t.Fatalf("current event = %+v, want hash %s", event, sha)
		}
	case <-ctx.Done():
		t.Fatal("worker did not report current before timeout")
	}
	cancel()
	<-done
}

func TestConnectOnceReadinessFollowsControlSession(t *testing.T) {
	accepted := make(chan struct{})
	sendHello := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = ws.Close(websocket.StatusNormalClosure, "") }()
		close(accepted)
		<-sendHello
		if err := writeJSON(r.Context(), ws, serverMessage{
			Type:              messageHello,
			DistributionPoint: pointIdentity{ID: 1, Name: "test"},
		}); err != nil {
			return
		}
		_, _, _ = ws.Read(r.Context())
	}))
	defer srv.Close()

	worker := newTestWorker(t, srv.URL)
	handler := worker.handler()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = worker.connectOnce(ctx)
	}()

	select {
	case <-accepted:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not open control WebSocket")
	}
	assertProbeResponse(t, handler, "/readyz", http.StatusServiceUnavailable, "not ready\n")

	close(sendHello)
	waitForProbeStatus(t, handler, "/readyz", http.StatusOK)
	assertProbeResponse(t, handler, "/readyz", http.StatusOK, "ready\n")

	cancel()
	<-done
	assertProbeResponse(t, handler, "/readyz", http.StatusServiceUnavailable, "not ready\n")
}

func TestConnectOnceRejectsUnexpectedMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = ws.Close(websocket.StatusNormalClosure, "") }()
		if err := ws.Write(r.Context(), websocket.MessageText, []byte(`{"type":"unknown"}`)); err != nil {
			return
		}
	}))
	defer srv.Close()

	worker := newTestWorker(t, srv.URL)
	err := worker.connectOnce(context.Background())
	if err == nil || !strings.Contains(err.Error(), `unexpected message type "unknown"`) {
		t.Fatalf("connectOnce error = %v, want unexpected message type", err)
	}
}

func assertProbeResponse(t *testing.T, handler http.Handler, path string, wantCode int, wantBody string) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != wantCode || rec.Body.String() != wantBody {
		t.Fatalf(
			"%s response = %d %q, want %d %q",
			path,
			rec.Code,
			rec.Body.String(),
			wantCode,
			wantBody,
		)
	}
}

func waitForProbeStatus(t *testing.T, handler http.Handler, path string, want int) {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		handler.ServeHTTP(rec, req)
		if rec.Code == want {
			return
		}
		select {
		case <-timer.C:
			t.Fatalf("%s status remained %d, want %d", path, rec.Code, want)
		case <-ticker.C:
		}
	}
}

func TestMirrorSnapshotRoundTrips(t *testing.T) {
	dir := t.TempDir()
	mirror, err := loadMirror(dir)
	if err != nil {
		t.Fatalf("loadMirror: %v", err)
	}
	mirror.setIdentity(pointIdentity{ID: 3, Name: "Melbourne"})
	mirror.put(7, packageState{Filename: "Chrome.pkg", SHA256: sha256Hex([]byte("x")), SizeBytes: 1})
	if err := mirror.save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	reloaded, err := loadMirror(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.identity.ID != 3 {
		t.Fatalf("identity = %+v, want id 3", reloaded.identity)
	}
	if _, ok := reloaded.get(7); !ok {
		t.Fatal("package state not restored from snapshot")
	}
}

func send(t *testing.T, ctx context.Context, ws *websocket.Conn, msg serverMessage) {
	t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal server message: %v", err)
	}
	if err := ws.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("write server message: %v", err)
	}
}
