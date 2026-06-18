package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	shutdownTimeout = 15 * time.Second

	initialReconnectDelay = 1 * time.Second
	maxReconnectDelay     = 30 * time.Second
	// connectionStableAfter is how long a connection must last before its
	// successor starts again from the initial reconnect delay.
	connectionStableAfter = time.Minute

	// maxMessageBytes bounds an inbound hello/desired_changed message so a large
	// fleet's package list is not truncated by the default read limit.
	maxMessageBytes = 8 << 20

	packageMirrorAttempts = 3
)

// Worker is the woodstar mdp face: a WebSocket control channel that mirrors
// installers on demand and a serve node that hands them to redirected Munki
// clients.
type Worker struct {
	cfg    Config
	logger *slog.Logger
	mirror *mirror
	client *woodstarClient
	server *server
}

// New restores the worker's mirror and wires its collaborators.
func New(cfg Config, logger *slog.Logger) (*Worker, error) {
	m, err := loadMirror(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	return &Worker{
		cfg:    cfg,
		logger: logger,
		mirror: m,
		client: newWoodstarClient(cfg.ServerURL, cfg.Key),
		server: &server{mirror: m, key: []byte(cfg.Key), logger: logger},
	}, nil
}

// Run starts the serve node and the Woodstar connection loop. It returns when
// ctx is cancelled or the serve node exits on its own; a serve-node failure
// stops the connection loop so the worker never lingers half-up.
func (w *Worker) Run(ctx context.Context) error {
	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", w.cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", w.cfg.ListenAddr, err)
	}

	httpServer := &http.Server{Handler: w.server.handler(), ReadHeaderTimeout: 10 * time.Second}
	serveErr := make(chan error, 1)
	go func() { serveErr <- httpServer.Serve(listener) }()

	w.logger.InfoContext(ctx, "started",
		"listen_addr", w.cfg.ListenAddr,
		"server_url", w.cfg.ServerURL,
		"data_dir", w.cfg.DataDir,
	)

	connCtx, cancelConn := context.WithCancel(ctx)
	defer cancelConn()
	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		w.connectLoop(connCtx)
	}()

	var runErr error
	select {
	case <-ctx.Done():
	case err := <-serveErr:
		runErr = fmt.Errorf("serve node: %w", err)
	}

	w.logger.InfoContext(ctx, "shutting down")
	cancelConn()
	<-loopDone

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil && runErr == nil {
		runErr = fmt.Errorf("shutdown serve node: %w", err)
	}
	if runErr == nil {
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			runErr = fmt.Errorf("serve node: %w", err)
		}
	}
	return runErr
}

// connectLoop keeps a Woodstar connection up, reconnecting with capped backoff.
// Each reconnect receives the whole desired set in the next hello.
func (w *Worker) connectLoop(ctx context.Context) {
	backoff := initialReconnectDelay
	for {
		if ctx.Err() != nil {
			return
		}
		start := time.Now()
		err := w.connectOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		if time.Since(start) >= connectionStableAfter {
			backoff = initialReconnectDelay
		}
		w.logger.WarnContext(ctx, "connection lost", "retry_in", backoff.String(), "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, maxReconnectDelay)
	}
}

// connectOnce dials Woodstar, then processes hello and desired_changed messages
// (reconciling and reporting state after each) until the connection ends.
func (w *Worker) connectOnce(ctx context.Context) error {
	w.logger.DebugContext(ctx, "connecting", "url", w.connectURL())
	// coder/websocket owns resp.Body (nil on success, a NopCloser on error), so we
	// neither read nor close it; doing so dereferences nil on the success path.
	//nolint:bodyclose // resp.Body lifecycle is managed by coder/websocket's Dial
	ws, _, err := websocket.Dial(ctx, w.connectURL(), &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + w.cfg.Key}},
	})
	if err != nil {
		return err
	}
	defer ws.Close(websocket.StatusNormalClosure, "")
	ws.SetReadLimit(maxMessageBytes)
	w.logger.InfoContext(ctx, "connected", "server_url", w.cfg.ServerURL)

	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			return err
		}
		var msg serverMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return fmt.Errorf("decode server message: %w", err)
		}
		switch msg.Type {
		case messageHello, messageDesiredChanged:
			failures := w.applyDesired(ctx, msg)
			if err := w.sendState(ctx, ws, failures); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected message type %q", msg.Type)
		}
	}
}

func (w *Worker) connectURL() string {
	base := w.cfg.ServerURL
	switch {
	case strings.HasPrefix(base, "https://"):
		base = "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		base = "ws://" + strings.TrimPrefix(base, "http://")
	}
	return base + "/api/munki/distribution/connect"
}

// applyDesired reconciles the mirror against a hello or desired_changed message.
func (w *Worker) applyDesired(ctx context.Context, msg serverMessage) map[int64]string {
	w.logger.DebugContext(ctx, "received message",
		"type", msg.Type,
		"packages", len(msg.Packages),
	)
	if msg.Type == messageHello {
		w.mirror.setIdentity(msg.DistributionPoint)
	}
	failures := w.reconcile(ctx, msg.Packages)
	if err := w.mirror.save(); err != nil {
		w.logger.WarnContext(ctx, "snapshot failed", "err", err)
	}
	return failures
}

func (w *Worker) sendState(ctx context.Context, ws *websocket.Conn, failures map[int64]string) error {
	packages := w.packageReport(failures)
	if err := writeJSON(ctx, ws, stateMessage{
		Type:     messageState,
		Packages: packages,
	}); err != nil {
		return err
	}
	w.logger.DebugContext(ctx, "reported state",
		"packages", len(packages),
	)
	return nil
}

// reconcile downloads missing or changed packages with bounded concurrency, then
// deletes any mirrored package no longer in the desired set.
func (w *Worker) reconcile(ctx context.Context, desired []desiredPackage) map[int64]string {
	w.logger.DebugContext(ctx, "reconciling", "desired", len(desired))
	wanted := make(map[int64]bool, len(desired))
	tokens := make(chan struct{}, w.cfg.DownloadConcurrency)
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		mirrored int
		upToDate int
		failed   int
		failures = map[int64]string{}
	)
	for _, pkg := range desired {
		wanted[pkg.PackageID] = true
		if w.upToDate(pkg) {
			upToDate++
			w.logger.DebugContext(ctx, "package up to date", "package_id", pkg.PackageID)
			continue
		}
		wg.Add(1)
		tokens <- struct{}{}
		go func(pkg desiredPackage) {
			defer wg.Done()
			defer func() { <-tokens }()
			if err := w.fetchWithRetry(ctx, pkg); err != nil {
				mu.Lock()
				failed++
				failures[pkg.PackageID] = err.Error()
				mu.Unlock()
				return
			}
			mu.Lock()
			mirrored++
			mu.Unlock()
		}(pkg)
	}
	wg.Wait()
	removed := w.pruneStale(ctx, wanted)
	w.logger.DebugContext(ctx, "reconcile complete",
		"mirrored", mirrored,
		"up_to_date", upToDate,
		"removed", removed,
		"failed", failed,
	)
	return failures
}

func (w *Worker) packageReport(failures map[int64]string) []reportedPackage {
	packages := w.mirror.report()
	if len(failures) == 0 {
		return packages
	}
	index := make(map[int64]int, len(packages))
	for i, pkg := range packages {
		index[pkg.PackageID] = i
	}
	for packageID, failure := range failures {
		if i, ok := index[packageID]; ok {
			packages[i].Status = packageStatusError
			packages[i].Error = failure
			continue
		}
		packages = append(packages, reportedPackage{
			PackageID: packageID,
			Status:    packageStatusError,
			Error:     failure,
		})
	}
	return packages
}

func (w *Worker) upToDate(pkg desiredPackage) bool {
	state, ok := w.mirror.get(pkg.PackageID)
	if !ok || state.SHA256 != pkg.SHA256 || state.SizeBytes != pkg.SizeBytes {
		return false
	}
	info, err := os.Stat(w.mirror.localPath(pkg.PackageID, state.Filename))
	return err == nil && info.Size() == pkg.SizeBytes
}

func (w *Worker) fetchWithRetry(ctx context.Context, pkg desiredPackage) error {
	var lastErr error
	for attempt := 1; attempt <= packageMirrorAttempts; attempt++ {
		if err := w.fetch(ctx, pkg); err != nil {
			lastErr = err
			if ctx.Err() != nil {
				return err
			}
			w.logger.WarnContext(ctx, "package mirror attempt failed",
				"package_id", pkg.PackageID,
				"attempt", attempt,
				"attempts", packageMirrorAttempts,
				"err", err,
			)
			continue
		}
		return nil
	}
	return lastErr
}

func (w *Worker) fetch(ctx context.Context, pkg desiredPackage) error {
	w.logger.DebugContext(ctx, "downloading package",
		"package_id", pkg.PackageID,
		"filename", filepath.Base(pkg.Filename),
		"size_bytes", pkg.SizeBytes,
	)
	path := w.mirror.localPath(pkg.PackageID, pkg.Filename)
	tmp := path + ".download"
	if err := w.client.download(ctx, pkg.PackageID, tmp); err != nil {
		_ = os.Remove(tmp)
		return fetchError("download", pkg.PackageID, err)
	}
	if err := verifyFile(tmp, pkg.SizeBytes, pkg.SHA256); err != nil {
		_ = os.Remove(tmp)
		return fetchError("verify", pkg.PackageID, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fetchError("commit", pkg.PackageID, err)
	}
	w.mirror.put(pkg.PackageID, packageState{
		Filename:   filepath.Base(pkg.Filename),
		SHA256:     pkg.SHA256,
		SizeBytes:  pkg.SizeBytes,
		VerifiedAt: time.Now(),
	})
	w.logger.DebugContext(ctx, "package mirrored",
		"package_id", pkg.PackageID,
		"size_bytes", pkg.SizeBytes,
	)
	return nil
}

// pruneStale removes any mirrored package no longer wanted and returns the count.
func (w *Worker) pruneStale(ctx context.Context, wanted map[int64]bool) int {
	removed := 0
	for _, id := range w.mirror.packageIDs() {
		if wanted[id] {
			continue
		}
		if state, ok := w.mirror.get(id); ok {
			_ = os.Remove(w.mirror.localPath(id, state.Filename))
		}
		w.mirror.remove(id)
		removed++
		w.logger.DebugContext(ctx, "removed package", "package_id", id)
	}
	return removed
}

func fetchError(operation string, packageID int64, err error) error {
	return fmt.Errorf("package %d %s: %w", packageID, operation, err)
}
