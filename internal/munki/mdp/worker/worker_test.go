package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func TestReconcileDownloadsVerifiesAndPrunes(t *testing.T) {
	content := []byte("chrome-installer-payload")
	sha := sha256Hex(content)
	size := int64(len(content))

	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/munki/distribution/packages/7/content" {
			_, _ = w.Write(content)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer fake.Close()

	worker := newTestWorker(t, fake.URL)
	ctx := context.Background()

	worker.reconcile(ctx, []desiredPackage{
		{PackageID: 7, Filename: "Chrome.pkg", SHA256: sha, SizeBytes: size},
	})

	path := worker.mirror.localPath(7, "Chrome.pkg")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mirrored file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("mirrored bytes = %q, want installer", got)
	}
	state, ok := worker.mirror.get(7)
	if !ok || state.SHA256 != sha || state.SizeBytes != size {
		t.Fatalf("mirror state = %+v (present %v), want verified", state, ok)
	}

	// A second package vanishes from the desired set and must be pruned.
	worker.reconcile(ctx, nil)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("stale file still present: %v", err)
	}
	if _, ok := worker.mirror.get(7); ok {
		t.Fatal("pruned package still in mirror")
	}
}

func TestReconcileRejectsCorruptDownload(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("corrupt"))
	}))
	defer fake.Close()

	worker := newTestWorker(t, fake.URL)
	failures := worker.reconcile(context.Background(), []desiredPackage{
		{PackageID: 7, Filename: "Chrome.pkg", SHA256: sha256Hex([]byte("expected")), SizeBytes: 8},
	})
	if failures[7] == "" {
		t.Fatalf("failures = %+v, want package error", failures)
	}

	if _, ok := worker.mirror.get(7); ok {
		t.Fatal("corrupt download was accepted into the mirror")
	}
	if _, err := os.Stat(worker.mirror.localPath(7, "Chrome.pkg")); !os.IsNotExist(err) {
		t.Fatalf("corrupt file left on disk: %v", err)
	}
}

func TestReconcileRetriesBeforeReportingPackageError(t *testing.T) {
	attempts := 0
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < packageMirrorAttempts {
			_, _ = w.Write([]byte("corrupt"))
			return
		}
		_, _ = w.Write([]byte("expected"))
	}))
	defer fake.Close()

	worker := newTestWorker(t, fake.URL)
	failures := worker.reconcile(context.Background(), []desiredPackage{
		{
			PackageID: 7,
			Filename:  "Chrome.pkg",
			SHA256:    sha256Hex([]byte("expected")),
			SizeBytes: int64(len("expected")),
		},
	})
	if len(failures) != 0 {
		t.Fatalf("failures = %+v, want success after retry", failures)
	}
	if attempts != packageMirrorAttempts {
		t.Fatalf("attempts = %d, want %d", attempts, packageMirrorAttempts)
	}
}

func TestConnectOnceProcessesHelloAndReportsState(t *testing.T) {
	gotState := make(chan stateMessage, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close(websocket.StatusNormalClosure, "")
		hello := serverMessage{Type: messageHello, DistributionPoint: pointIdentity{ID: 1, Name: "test"}}
		data, err := json.Marshal(hello)
		if err != nil {
			return
		}
		if err := ws.Write(r.Context(), websocket.MessageText, data); err != nil {
			return
		}
		_, raw, err := ws.Read(r.Context())
		if err != nil {
			return
		}
		var msg stateMessage
		if err := json.Unmarshal(raw, &msg); err == nil {
			gotState <- msg
		}
	}))
	defer srv.Close()

	worker := newTestWorker(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = worker.connectOnce(ctx)
	}()

	select {
	case msg := <-gotState:
		if msg.Type != messageState {
			t.Fatalf("state report = %+v, want clean state", msg)
		}
	case <-ctx.Done():
		t.Fatal("worker did not report state before timeout")
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
