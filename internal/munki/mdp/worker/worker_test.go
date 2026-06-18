package worker

import (
	"context"
	"encoding/json"
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
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	client := newWoodstarClient(serverURL, "dp-key")
	return newSession(ctx, mirror, client, discardLogger(), 2, time.Millisecond)
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

func TestSessionMirrorsVerifiesAndPrunes(t *testing.T) {
	content := []byte("chrome-installer-payload")
	sha := sha256Hex(content)
	size := int64(len(content))
	srv := fakeWoodstar(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(content)
	})
	sess := newTestSession(t, srv.URL)

	sess.applyDesiredSet([]desiredPackage{
		{PackageID: 7, Filename: "Chrome.pkg", SHA256: sha, SizeBytes: size},
	})

	if event := waitEvent(t, sess.events, eventPackageCurrent); event.PackageID != 7 || event.SHA256 != sha {
		t.Fatalf("current event = %+v, want package 7 with hash", event)
	}

	path := sess.mirror.localPath(7, "Chrome.pkg")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mirrored file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("mirrored bytes = %q, want installer", got)
	}
	state, ok := sess.mirror.get(7)
	if !ok || state.SHA256 != sha || state.SizeBytes != size {
		t.Fatalf("mirror state = %+v (present %v), want verified", state, ok)
	}

	// The package vanishes from the desired set and must be pruned.
	sess.applyDesiredSet(nil)
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

	sess.applyDesiredSet([]desiredPackage{
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

	sess.applyDesiredSet([]desiredPackage{
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
		defer ws.Close(websocket.StatusNormalClosure, "")
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

func TestConnectOnceRejectsUnexpectedMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close(websocket.StatusNormalClosure, "")
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
