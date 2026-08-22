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

// SnapshotApplier applies fetched Entra snapshots to local directory state.
type SnapshotApplier interface {
	ApplyProviderSnapshot(ctx context.Context, source directory.Source, snapshot directory.ProviderSnapshot) error
}

// Service fetches and applies full Entra snapshots.
type Service struct {
	applier SnapshotApplier
	fetcher Fetcher
	logger  *slog.Logger
}

// SyncResult summarizes one applied provider snapshot.
type SyncResult struct {
	Users      int `json:"users"`
	Groups     int `json:"groups"`
	DurationMS int `json:"duration_ms"`
}

// NewService composes an Entra fetcher with directory reconciliation.
func NewService(
	applier SnapshotApplier,
	fetcher Fetcher,
	logger *slog.Logger,
) *Service {
	return &Service{applier: applier, fetcher: fetcher, logger: logger}
}

// Sync performs a single full reconciliation. Errors from either the fetch or
// database reconciliation phase abort the pass and are returned for logging.
func (s *Service) Sync(ctx context.Context) (SyncResult, error) {
	if s.fetcher == nil {
		return SyncResult{}, errors.New("entra: no fetcher configured")
	}
	if s.applier == nil {
		return SyncResult{}, errors.New("entra: no snapshot applier configured")
	}
	started := time.Now()
	snapshot, err := s.fetcher.Fetch(ctx)
	if err != nil {
		return SyncResult{}, err
	}
	if err := s.applier.ApplyProviderSnapshot(ctx, directory.SourceEntra, snapshot); err != nil {
		return SyncResult{}, err
	}
	result := SyncResult{
		Users:      len(snapshot.Users),
		Groups:     len(snapshot.Groups),
		DurationMS: int(time.Since(started).Milliseconds()),
	}
	s.logger.InfoContext(ctx, "entra sync complete",
		"operation", "sync",
		"users", result.Users,
		"groups", result.Groups,
		"duration_ms", result.DurationMS,
	)
	return result, nil
}
