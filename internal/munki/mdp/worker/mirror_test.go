package worker

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/woodleighschool/woodstar/internal/munki/mdp/wire"
)

func desiredPackage(content string) wire.DesiredPackage {
	return wire.DesiredPackage{PackageID: 7, Filename: "Installer.pkg", SHA256: sha256Hex([]byte(content)), SizeBytes: int64(len(content))}
}

func writeCacheFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func requireAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s remains: %v", path, err)
	}
}

func TestReconcileOwnsDirectoryAcrossRestarts(t *testing.T) {
	dir := t.TempDir()
	pkg := desiredPackage("verified installer")
	cache, err := openMirror(dir)
	if err != nil {
		t.Fatal(err)
	}
	retained := cache.localPath(pkg.PackageID, pkg.SHA256)
	writeCacheFile(t, retained, "verified installer")
	obsolete := []string{"state.json", "state.json.tmp", "7-OldName.pkg", "8-forgotten.pkg", filepath.Base(retained) + ".download"}
	for _, name := range obsolete {
		writeCacheFile(t, filepath.Join(dir, name), "untracked or interrupted bytes")
	}
	outside := filepath.Join(t.TempDir(), "keep")
	writeCacheFile(t, outside, "outside the cache")
	link := filepath.Join(dir, "orphan-link")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := cache.reconcile(t.Context(), []wire.DesiredPackage{pkg}, nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range obsolete {
		requireAbsent(t, filepath.Join(dir, name))
	}
	requireAbsent(t, link)
	if _, err := os.Stat(outside); err != nil {
		t.Fatal(err)
	}
	if _, ok := cache.get(pkg.PackageID); !ok {
		t.Fatal("verified package was not available")
	}

	restarted, err := openMirror(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := restarted.get(pkg.PackageID); ok {
		t.Fatal("package advertised before verification")
	}
	pkg.Filename = "Renamed.pkg"
	if err := restarted.reconcile(t.Context(), []wire.DesiredPackage{pkg}, nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := restarted.get(pkg.PackageID); !ok {
		t.Fatal("retained bytes not recovered after restart and rename")
	}
	// Matching size is insufficient after restart.
	writeCacheFile(t, retained, strings.Repeat("x", int(pkg.SizeBytes)))
	restarted, err = openMirror(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.reconcile(t.Context(), []wire.DesiredPackage{pkg}, nil); err != nil {
		t.Fatal(err)
	}
	requireAbsent(t, retained)
	if _, ok := restarted.get(pkg.PackageID); ok {
		t.Fatal("corrupt bytes were advertised")
	}
}

func TestReplacementReclaimsOldBytesBeforeDownload(t *testing.T) {
	old := desiredPackage("previous installer")
	next := desiredPackage("replacement installer")
	sess := newTestSession(t, "http://woodstar.invalid")
	oldPath := sess.mirror.localPath(old.PackageID, old.SHA256)
	writeCacheFile(t, oldPath, "previous installer")
	if err := sess.applyDesiredSet(t.Context(), []wire.DesiredPackage{old}); err != nil {
		t.Fatal(err)
	}
	waitEvent(t, sess.events, wire.EventPackageCurrent)
	var requests atomic.Int32
	srv := fakeWoodstar(t, func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		if _, err := os.Stat(oldPath); !errors.Is(err, os.ErrNotExist) {
			t.Error("obsolete installer still occupies space during replacement")
		}
		_, _ = io.WriteString(w, "replacement installer")
	})
	sess.client.serverURL = srv.URL
	if err := sess.applyDesiredSet(t.Context(), []wire.DesiredPackage{next}); err != nil {
		t.Fatal(err)
	}
	waitEvent(t, sess.events, wire.EventPackageCurrent)
	if requests.Load() != 1 {
		t.Fatalf("downloads = %d, want 1", requests.Load())
	}
	requireAbsent(t, oldPath)
	body, err := os.ReadFile(sess.mirror.localPath(next.PackageID, next.SHA256))
	if err != nil || string(body) != "replacement installer" {
		t.Fatalf("replacement = %q, %v", body, err)
	}
}

func TestAllocationFailureRetriesWithoutNetwork(t *testing.T) {
	for _, failure := range []error{unix.ENOSPC, unix.EDQUOT} {
		t.Run(failure.Error(), func(t *testing.T) {
			pkg := desiredPackage("installer")
			var network atomic.Int32
			srv := fakeWoodstar(t, func(w http.ResponseWriter, _ *http.Request) { network.Add(1); _, _ = io.WriteString(w, "installer") })
			sess := newTestSession(t, srv.URL)
			transport := sess.client.apiHTTP.Transport
			sess.client.apiHTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) { network.Add(1); return transport.RoundTrip(r) })
			allocationFails := true
			sess.mirror.allocate = func(f *os.File, size int64) error {
				if allocationFails {
					return failure
				}
				return allocateFile(f, size)
			}
			for range 3 {
				if err := sess.fetchOnce(t.Context(), pkg); !errors.Is(err, failure) {
					t.Fatalf("allocation failure = %v", err)
				}
				requireAbsent(t, sess.mirror.localPath(pkg.PackageID, pkg.SHA256)+".download")
			}
			if network.Load() != 0 {
				t.Fatalf("network requests during full disk = %d", network.Load())
			}
			// A new connection shares the cache, but neither bypasses allocation nor
			// needs a free-space counter reset when storage becomes available.
			next := newSession(sess.mirror, sess.client, discardLogger(), 1, 1)
			if err := next.fetchOnce(t.Context(), pkg); !errors.Is(err, failure) {
				t.Fatalf("reconnected allocation = %v", err)
			}
			allocationFails = false
			if err := next.fetchOnce(t.Context(), pkg); err != nil {
				t.Fatal(err)
			}
			if network.Load() != 2 {
				t.Fatalf("recovered requests = %d, want URL and blob", network.Load())
			}
		})
	}
}

func TestDownloadRejectsWrongLengthWithoutPublishing(t *testing.T) {
	for _, body := range []string{"short", "expected-extra"} {
		t.Run(body, func(t *testing.T) {
			pkg := desiredPackage("expected")
			srv := fakeWoodstar(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, body) })
			sess := newTestSession(t, srv.URL)
			if err := sess.fetchOnce(t.Context(), pkg); err == nil {
				t.Fatal("accepted wrong length")
			}
			requireAbsent(t, sess.mirror.localPath(pkg.PackageID, pkg.SHA256))
			requireAbsent(t, sess.mirror.localPath(pkg.PackageID, pkg.SHA256)+".download")
			if _, ok := sess.mirror.get(pkg.PackageID); ok {
				t.Fatal("unverified download advertised")
			}
		})
	}
}

func TestReconcileRetriesFailedCleanupWithoutDesiredUpdate(t *testing.T) {
	pkg := desiredPackage("installer")
	sess := newTestSession(t, "http://woodstar.invalid")
	writeCacheFile(t, sess.mirror.localPath(pkg.PackageID, pkg.SHA256), "installer")
	obstacle := filepath.Join(sess.mirror.dataDir, "obsolete-directory")
	if err := os.Mkdir(obstacle, 0o700); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(obstacle, "file")
	writeCacheFile(t, child, "prevents directory removal")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); sess.reconcileLoop(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	missing := desiredPackage("another installer")
	missing.PackageID = 8
	sess.submitDesired([]wire.DesiredPackage{pkg, missing})
	if event := waitEvent(t, sess.events, wire.EventPackageCurrent); event.PackageID != pkg.PackageID {
		t.Fatalf("healthy retained installer not advertised: %+v", event)
	}
	event := waitEvent(t, sess.events, wire.EventPackageError)
	if event.PackageID != missing.PackageID || !strings.Contains(event.Error, "remove cache file") {
		t.Fatalf("cleanup error not reported: %+v", event)
	}
	if err := os.Remove(child); err != nil {
		t.Fatal(err)
	}
	if event := waitEvent(t, sess.events, wire.EventPackageSyncing); event.PackageID != missing.PackageID {
		t.Fatalf("missing download not resumed after cleanup: %+v", event)
	}
	requireAbsent(t, obstacle)
}

func TestDesiredChangesJoinObsoleteDownloads(t *testing.T) {
	a := desiredPackage("installer A")
	b := desiredPackage("installer B")
	started := make(chan struct{}, 2)
	var requests atomic.Int32
	srv := fakeWoodstar(t, func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) <= 2 {
			_, _ = io.WriteString(w, "install")
			if err := http.NewResponseController(w).Flush(); err != nil {
				t.Error(err)
				return
			}
			started <- struct{}{}
			<-r.Context().Done()
			return
		}
		_, _ = io.WriteString(w, "installer A")
	})
	sess := newTestSession(t, srv.URL)
	if err := sess.applyDesiredSet(t.Context(), []wire.DesiredPackage{a}); err != nil {
		t.Fatal(err)
	}
	<-started
	partial := sess.mirror.localPath(a.PackageID, a.SHA256) + ".download"
	for range 3 {
		if err := sess.applyDesiredSet(t.Context(), []wire.DesiredPackage{a}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(partial); err != nil {
			t.Fatalf("active download removed: %v", err)
		}
	}
	if err := sess.applyDesiredSet(t.Context(), []wire.DesiredPackage{b}); err != nil {
		t.Fatal(err)
	}
	<-started
	requireAbsent(t, partial)
	if err := sess.applyDesiredSet(t.Context(), []wire.DesiredPackage{a}); err != nil {
		t.Fatal(err)
	}
	event := waitEvent(t, sess.events, wire.EventPackageCurrent)
	if event.SHA256 != a.SHA256 {
		t.Fatalf("current event = %+v", event)
	}
	sess.wait()
	requireAbsent(t, sess.mirror.localPath(b.PackageID, b.SHA256))
	requireAbsent(t, sess.mirror.localPath(b.PackageID, b.SHA256)+".download")
	entries, err := os.ReadDir(sess.mirror.dataDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("cache contains %d files, %v", len(entries), err)
	}
	if requests.Load() != 3 {
		t.Fatalf("requests = %d, want 3", requests.Load())
	}
}
