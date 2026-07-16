package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// eventBuffer holds package events between the jobs that emit them and the
// writer that flushes them to Woodstar, so a job never blocks on the socket.
const eventBuffer = 64

// maxJobRetry caps the backoff between failed mirror attempts so a persistently
// failing package keeps retrying without flooding logs or the server.
const maxJobRetry = 5 * time.Minute

// session is one connection's reconciler. It owns the in-flight download jobs
// and the event stream for a single WebSocket; the long-lived mirror it writes
// into outlives it. Reconciliation runs off the read loop so a slow download
// never stalls liveness or control messages.
type session struct {
	mirror     *mirror
	client     *woodstarClient
	logger     *slog.Logger
	sem        chan struct{}
	events     chan packageEvent
	desiredCh  chan []desiredPackage
	retryDelay time.Duration

	mu      sync.Mutex
	desired map[int64]desiredPackage
	jobs    map[int64]*jobHandle
	wg      sync.WaitGroup
}

// jobHandle tracks one running mirror job and the exact bytes it targets, so a
// desired-set change can tell a still-correct job from one to cancel and restart.
type jobHandle struct {
	cancel context.CancelFunc
	sha    string
	size   int64
}

func newSession(
	m *mirror,
	client *woodstarClient,
	logger *slog.Logger,
	concurrency int,
	retryDelay time.Duration,
) *session {
	if concurrency < 1 {
		concurrency = 1
	}
	return &session{
		mirror:     m,
		client:     client,
		logger:     logger,
		sem:        make(chan struct{}, concurrency),
		events:     make(chan packageEvent, eventBuffer),
		desiredCh:  make(chan []desiredPackage, 1),
		retryDelay: retryDelay,
		desired:    map[int64]desiredPackage{},
		jobs:       map[int64]*jobHandle{},
	}
}

// submitDesired hands the latest desired set to the reconcile loop without
// blocking the read loop. A pending set is replaced: only the newest matters.
func (s *session) submitDesired(pkgs []desiredPackage) {
	select {
	case s.desiredCh <- pkgs:
	default:
		select {
		case <-s.desiredCh:
		default:
		}
		select {
		case s.desiredCh <- pkgs:
		default:
		}
	}
}

// reconcileLoop applies desired sets one at a time until the connection ends.
func (s *session) reconcileLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case pkgs := <-s.desiredCh:
			s.applyDesiredSet(ctx, pkgs)
		}
	}
}

// writeEvents flushes package events to the WebSocket. It is the connection's
// sole writer; the read loop only reads.
func (s *session) writeEvents(ctx context.Context, ws *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-s.events:
			if err := writeJSON(ctx, ws, event); err != nil {
				return
			}
		}
	}
}

// wait blocks until every running job has stopped. The caller cancels the
// session context first, so jobs abort their downloads and clean up temp files
// before a reconnect reuses their paths.
func (s *session) wait() {
	s.wg.Wait()
}

// applyDesiredSet reconciles the mirror against the full desired list: it starts
// or restarts jobs for missing or changed packages, re-advertises packages it
// already holds, and prunes everything no longer wanted.
func (s *session) applyDesiredSet(ctx context.Context, pkgs []desiredPackage) {
	wanted := make(map[int64]bool, len(pkgs))
	var current, start []desiredPackage

	s.mu.Lock()
	for _, pkg := range pkgs {
		wanted[pkg.PackageID] = true
		s.desired[pkg.PackageID] = pkg
		if s.mirror.satisfies(pkg.PackageID, pkg.SHA256, pkg.SizeBytes) {
			s.cancelJobLocked(pkg.PackageID)
			current = append(current, pkg)
		} else {
			start = append(start, pkg)
		}
	}
	prune := s.prunableLocked(wanted)
	for _, id := range prune {
		s.cancelJobLocked(id)
		delete(s.desired, id)
	}
	for _, pkg := range start {
		s.ensureJobLocked(ctx, pkg)
	}
	s.mu.Unlock()

	for _, pkg := range current {
		s.logger.DebugContext(ctx, "package already current",
			"package_id", pkg.PackageID, "filename", pkg.Filename)
		s.emit(ctx, packageEvent{Type: eventPackageCurrent, PackageID: pkg.PackageID, SHA256: pkg.SHA256})
	}
	for _, id := range prune {
		s.logger.DebugContext(ctx, "pruning package", "package_id", id)
		s.pruneBytes(id)
	}
	if len(prune) > 0 {
		s.save(ctx)
	}
	s.logger.DebugContext(ctx, "reconciled desired set",
		"desired", len(pkgs),
		"already_current", len(current),
		"downloading", len(start),
		"pruned", len(prune),
	)
}

// prunableLocked returns mirrored or in-flight package ids no longer wanted.
func (s *session) prunableLocked(wanted map[int64]bool) []int64 {
	seen := map[int64]bool{}
	var prune []int64
	add := func(id int64) {
		if !wanted[id] && !seen[id] {
			seen[id] = true
			prune = append(prune, id)
		}
	}
	for id := range s.jobs {
		add(id)
	}
	for _, id := range s.mirror.packageIDs() {
		add(id)
	}
	return prune
}

func (s *session) cancelJobLocked(packageID int64) {
	if handle, ok := s.jobs[packageID]; ok {
		handle.cancel()
		delete(s.jobs, packageID)
	}
}

// ensureJobLocked starts a mirror job for a package unless one targeting the
// same bytes is already running; a job for different bytes is cancelled first.
func (s *session) ensureJobLocked(ctx context.Context, pkg desiredPackage) {
	if handle, ok := s.jobs[pkg.PackageID]; ok {
		if handle.sha == pkg.SHA256 && handle.size == pkg.SizeBytes {
			return
		}
		handle.cancel()
		delete(s.jobs, pkg.PackageID)
	}
	jobCtx, cancel := context.WithCancel(ctx)
	s.jobs[pkg.PackageID] = &jobHandle{cancel: cancel, sha: pkg.SHA256, size: pkg.SizeBytes}
	s.wg.Add(1)
	go s.runJob(jobCtx, pkg)
}

// runJob mirrors one package, retrying on failure until it succeeds, the package
// changes, or the connection ends. It emits syncing once, an error per failed
// attempt, and current when the verified bytes land.
func (s *session) runJob(ctx context.Context, pkg desiredPackage) {
	defer s.wg.Done()
	defer s.finishJob(pkg)

	s.logger.DebugContext(ctx, "mirroring package",
		"package_id", pkg.PackageID, "filename", pkg.Filename, "size_bytes", pkg.SizeBytes)
	s.emit(ctx, packageEvent{Type: eventPackageSyncing, PackageID: pkg.PackageID})

	delay := s.retryDelay
	for attempt := 1; ; attempt++ {
		if ctx.Err() != nil {
			return
		}
		if err := s.fetchOnce(ctx, pkg); err == nil {
			s.logger.InfoContext(ctx, "package mirrored",
				"package_id", pkg.PackageID, "filename", pkg.Filename, "size_bytes", pkg.SizeBytes)
			s.emit(ctx, packageEvent{
				Type: eventPackageCurrent, PackageID: pkg.PackageID, SHA256: pkg.SHA256,
			})
			return
		} else if ctx.Err() == nil {
			s.logFailure(ctx, pkg.PackageID, attempt, err)
			s.emit(ctx, packageEvent{
				Type: eventPackageError, PackageID: pkg.PackageID, Error: err.Error(),
			})
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		delay = min(delay*2, maxJobRetry)
	}
}

// logFailure warns on a package's first failed attempt and drops to debug for
// the retries, so a persistently failing package does not flood the warn log.
func (s *session) logFailure(ctx context.Context, packageID int64, attempt int, err error) {
	if attempt == 1 {
		s.logger.WarnContext(ctx, "package mirror failed, retrying",
			"package_id", packageID, "err", err)
		return
	}
	s.logger.DebugContext(ctx, "package mirror retry failed",
		"package_id", packageID, "attempt", attempt, "err", err)
}

func (s *session) finishJob(pkg desiredPackage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if handle, ok := s.jobs[pkg.PackageID]; ok &&
		handle.sha == pkg.SHA256 && handle.size == pkg.SizeBytes {
		delete(s.jobs, pkg.PackageID)
	}
}

// fetchOnce downloads and verifies one package. It holds a transfer slot for the
// whole attempt so the URL it fetches is used immediately rather than queued
// behind other downloads until it expires.
func (s *session) fetchOnce(ctx context.Context, pkg desiredPackage) error {
	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-s.sem }()

	url, err := s.client.downloadURL(ctx, pkg.PackageID)
	if err != nil {
		return fetchError("url", pkg.PackageID, err)
	}

	path := s.mirror.localPath(pkg.PackageID, pkg.Filename)
	tmp := path + ".download"
	if err := s.client.download(ctx, url, tmp); err != nil {
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
	s.mirror.put(pkg.PackageID, packageState{
		Filename:   filepath.Base(pkg.Filename),
		SHA256:     pkg.SHA256,
		SizeBytes:  pkg.SizeBytes,
		VerifiedAt: time.Now(),
	})
	s.save(ctx)
	return nil
}

func (s *session) pruneBytes(packageID int64) {
	if state, ok := s.mirror.get(packageID); ok {
		_ = os.Remove(s.mirror.localPath(packageID, state.Filename))
	}
	s.mirror.remove(packageID)
}

func (s *session) save(ctx context.Context) {
	if err := s.mirror.save(); err != nil {
		s.logger.WarnContext(ctx, "snapshot failed", "err", err)
	}
}

// emit hands a package event to the writer, dropping it if the connection has
// already ended (its replacement reconciles from a fresh desired set).
func (s *session) emit(ctx context.Context, event packageEvent) {
	select {
	case s.events <- event:
	case <-ctx.Done():
	}
}

func fetchError(operation string, packageID int64, err error) error {
	return fmt.Errorf("package %d %s: %w", packageID, operation, err)
}
