package worker

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/coder/websocket"

	"github.com/woodleighschool/woodstar/internal/munki/mdp/wire"
)

// eventBuffer holds package events between the jobs that emit them and the
// writer that flushes them to the server, so a job never blocks on the socket.
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
	events     chan wire.PackageEvent
	desiredCh  chan []wire.DesiredPackage
	retryDelay time.Duration

	jobs map[int64]*jobHandle
}

// jobHandle tracks one running mirror job and the exact bytes it targets, so a
// desired-set change can tell a still-correct job from one to cancel and restart.
type jobHandle struct {
	cancel context.CancelFunc
	sha    string
	size   int64
	done   chan struct{}
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
		events:     make(chan wire.PackageEvent, eventBuffer),
		desiredCh:  make(chan []wire.DesiredPackage, 1),
		retryDelay: retryDelay,
		jobs:       map[int64]*jobHandle{},
	}
}

// submitDesired hands the latest desired set to the reconcile loop without
// blocking the read loop. A pending set is replaced: only the newest matters.
func (s *session) submitDesired(pkgs []wire.DesiredPackage) {
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
	defer s.wait()
	var desired []wire.DesiredPackage
	var retry <-chan time.Time
	delay := s.retryDelay
	for {
		select {
		case <-ctx.Done():
			return
		case desired = <-s.desiredCh:
			delay = s.retryDelay
		case <-retry:
		}
		retry = nil
		if err := s.applyDesiredSet(ctx, desired); err != nil && ctx.Err() == nil {
			s.logger.WarnContext(ctx, "cache reconciliation failed", "err", err)
			for _, pkg := range desired {
				if _, active := s.jobs[pkg.PackageID]; active {
					continue
				}
				if state, ok := s.mirror.get(pkg.PackageID); ok && state.SHA256 == pkg.SHA256 && state.SizeBytes == pkg.SizeBytes {
					continue
				}
				s.emit(ctx, wire.PackageEvent{Type: wire.EventPackageError, PackageID: pkg.PackageID, Error: err.Error()})
			}
			retry = time.After(delay)
			delay = min(delay*2, maxJobRetry)
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

// The coordinator owns jobs and joins them before the next session can start.
func (s *session) wait() {
	for _, handle := range s.jobs {
		handle.cancel()
	}
	for _, handle := range s.jobs {
		<-handle.done
	}
}

func (s *session) applyDesiredSet(ctx context.Context, pkgs []wire.DesiredPackage) error {
	wanted := make(map[int64]wire.DesiredPackage, len(pkgs))
	for _, pkg := range pkgs {
		digest, err := hex.DecodeString(pkg.SHA256)
		if err != nil || len(digest) != 32 || pkg.SHA256 != strings.ToLower(pkg.SHA256) ||
			pkg.PackageID <= 0 || pkg.SizeBytes < 0 {
			return fmt.Errorf("invalid desired package %d", pkg.PackageID)
		}
		if _, exists := wanted[pkg.PackageID]; exists {
			return fmt.Errorf("duplicate desired package %d", pkg.PackageID)
		}
		wanted[pkg.PackageID] = pkg
	}
	for id, handle := range s.jobs {
		pkg, keep := wanted[id]
		if !keep || pkg.SHA256 != handle.sha || pkg.SizeBytes != handle.size {
			handle.cancel()
			<-handle.done
			delete(s.jobs, id)
		}
	}
	active := make(map[int64]bool, len(s.jobs))
	for id, handle := range s.jobs {
		select {
		case <-handle.done:
			delete(s.jobs, id)
		default:
			active[id] = true
		}
	}
	reconcileErr := s.mirror.reconcile(ctx, pkgs, active)
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, pkg := range pkgs {
		if active[pkg.PackageID] {
			continue
		}
		if state, ok := s.mirror.get(pkg.PackageID); ok && state.SHA256 == pkg.SHA256 && state.SizeBytes == pkg.SizeBytes {
			s.emit(ctx, wire.PackageEvent{Type: wire.EventPackageCurrent, PackageID: pkg.PackageID, SHA256: pkg.SHA256})
			continue
		}
		if reconcileErr != nil {
			continue
		}
		jobCtx, cancel := context.WithCancel(ctx)
		handle := &jobHandle{cancel: cancel, sha: pkg.SHA256, size: pkg.SizeBytes, done: make(chan struct{})}
		s.jobs[pkg.PackageID] = handle
		go func() {
			defer close(handle.done)
			s.runJob(jobCtx, pkg)
		}()
	}
	return reconcileErr
}

// runJob mirrors one package, retrying on failure until it succeeds, the package
// changes, or the connection ends. It emits syncing once, an error per failed
// attempt, and current when the verified bytes land.
func (s *session) runJob(ctx context.Context, pkg wire.DesiredPackage) {
	s.logger.DebugContext(ctx, "mirroring package",
		"package_id", pkg.PackageID, "filename", pkg.Filename, "size_bytes", pkg.SizeBytes)
	s.emit(ctx, wire.PackageEvent{Type: wire.EventPackageSyncing, PackageID: pkg.PackageID})

	delay := s.retryDelay
	for attempt := 1; ; attempt++ {
		if ctx.Err() != nil {
			return
		}
		if err := s.fetchOnce(ctx, pkg); err == nil {
			s.logger.InfoContext(ctx, "package mirrored",
				"package_id", pkg.PackageID, "filename", pkg.Filename, "size_bytes", pkg.SizeBytes)
			s.emit(ctx, wire.PackageEvent{
				Type: wire.EventPackageCurrent, PackageID: pkg.PackageID, SHA256: pkg.SHA256,
			})
			return
		} else if ctx.Err() == nil {
			s.logFailure(ctx, pkg.PackageID, attempt, err)
			s.emit(ctx, wire.PackageEvent{
				Type: wire.EventPackageError, PackageID: pkg.PackageID, Error: err.Error(),
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

func (s *session) fetchOnce(ctx context.Context, pkg wire.DesiredPackage) error {
	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-s.sem }()
	if err := ctx.Err(); err != nil {
		return err
	}
	err := s.mirror.fetch(ctx, pkg, func(dest io.Writer) (int64, error) {
		url, err := s.client.downloadURL(ctx, pkg.PackageID)
		if err != nil {
			return 0, err
		}
		return s.client.download(ctx, url, dest, pkg.SizeBytes)
	})
	if err != nil {
		return fmt.Errorf("cache package %d: %w", pkg.PackageID, err)
	}
	return nil
}

// emit hands a package event to the writer, dropping it if the connection has
// already ended (its replacement reconciles from a fresh desired set).
func (s *session) emit(ctx context.Context, event wire.PackageEvent) {
	select {
	case s.events <- event:
	case <-ctx.Done():
	}
}
