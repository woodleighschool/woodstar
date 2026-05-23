package directory

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// Fetcher returns a directory snapshot. Implemented by EntraClient in
// production; tests pass an in-memory fake.
type Fetcher interface {
	Fetch(ctx context.Context) (Snapshot, error)
}

// Service runs sync passes on demand and on a fixed interval.
type Service struct {
	store   *Store
	fetcher Fetcher
	logger  *slog.Logger
}

// NewService composes a Store with a Fetcher and logger.
func NewService(store *Store, fetcher Fetcher, logger *slog.Logger) *Service {
	return &Service{store: store, fetcher: fetcher, logger: logger}
}

// Sync performs a single full reconciliation. Errors from either the fetch or
// database reconciliation phase abort the pass and are returned for logging.
func (s *Service) Sync(ctx context.Context) error {
	if s.fetcher == nil {
		return errors.New("directory: no fetcher configured")
	}
	started := time.Now()
	snapshot, err := s.fetcher.Fetch(ctx)
	if err != nil {
		return err
	}
	if err := s.store.Apply(ctx, snapshot); err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "directory sync complete",
		"component", "directory",
		"operation", "sync",
		"users", len(snapshot.Users),
		"groups", len(snapshot.Groups),
		"duration_ms", time.Since(started).Milliseconds(),
	)
	return nil
}

// StartScheduler runs Sync once immediately, then again every interval. The
// returned function stops the scheduler before the parent context is cancelled.
func (s *Service) StartScheduler(ctx context.Context, interval time.Duration) func() {
	if interval <= 0 {
		interval = time.Hour
	}
	ctx, stop := context.WithCancel(ctx)
	go func() {
		s.runOnce(ctx)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
	return stop
}

func (s *Service) runOnce(ctx context.Context) {
	if err := s.Sync(ctx); err != nil {
		s.logger.ErrorContext(ctx, "directory sync failed",
			"component", "directory",
			"operation", "sync",
			"err", err,
		)
	}
}
