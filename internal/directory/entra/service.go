package entra

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/woodleighschool/woodstar/internal/directory"
)

// Fetcher returns an Entra snapshot.
type Fetcher interface {
	Fetch(ctx context.Context) (directory.ProviderSnapshot, error)
}

// SnapshotApplier applies fetched Entra snapshots to Woodstar directory state.
type SnapshotApplier interface {
	ApplyProviderSnapshot(ctx context.Context, source directory.Source, snapshot directory.ProviderSnapshot) error
}

type DerivedLabelRefresher interface {
	RefreshDerived(ctx context.Context) error
}

// Service runs sync passes on demand and on a fixed interval.
type Service struct {
	applier        SnapshotApplier
	fetcher        Fetcher
	logger         *slog.Logger
	labelRefresher DerivedLabelRefresher
}

// NewService composes an Entra fetcher with directory reconciliation.
func NewService(
	applier SnapshotApplier,
	fetcher Fetcher,
	logger *slog.Logger,
	labelRefresher DerivedLabelRefresher,
) *Service {
	return &Service{applier: applier, fetcher: fetcher, logger: logger, labelRefresher: labelRefresher}
}

// Sync performs a single full reconciliation. Errors from either the fetch or
// database reconciliation phase abort the pass and are returned for logging.
func (s *Service) Sync(ctx context.Context) error {
	if s.fetcher == nil {
		return errors.New("entra: no fetcher configured")
	}
	if s.applier == nil {
		return errors.New("entra: no snapshot applier configured")
	}
	started := time.Now()
	snapshot, err := s.fetcher.Fetch(ctx)
	if err != nil {
		return err
	}
	if err := s.applier.ApplyProviderSnapshot(ctx, directory.SourceEntra, snapshot); err != nil {
		return err
	}
	if s.labelRefresher != nil {
		if err := s.labelRefresher.RefreshDerived(ctx); err != nil {
			return err
		}
	}
	s.logger.InfoContext(ctx, "entra sync complete",
		"component", "entra",
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
		if errors.Is(err, context.Canceled) {
			return
		}
		s.logger.ErrorContext(ctx, "entra sync failed",
			"component", "entra",
			"operation", "sync",
			"err", err,
		)
	}
}
