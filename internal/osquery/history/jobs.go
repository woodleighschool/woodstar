package history

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/woodleighschool/woodstar/internal/backgroundjobs"
)

const (
	SnapshotJobKind     = "osquery_history_snapshot"
	SnapshotJobInterval = BucketInterval
	CleanupJobKind      = "osquery_history_cleanup"
	CleanupJobInterval  = 24 * time.Hour
)

// SnapshotJobArgs identifies a scheduled osquery history snapshot.
type SnapshotJobArgs struct {
	Trigger backgroundjobs.Trigger `json:"trigger"`
}

// Kind returns the River job kind.
func (SnapshotJobArgs) Kind() string { return SnapshotJobKind }

// InsertOpts prevents overlapping history snapshots.
func (SnapshotJobArgs) InsertOpts() river.InsertOpts { return backgroundjobs.SingletonInsertOpts() }

// CleanupJobArgs identifies a scheduled osquery history-retention sweep.
type CleanupJobArgs struct {
	Trigger       backgroundjobs.Trigger `json:"trigger"`
	RetentionDays int                    `json:"retention_days"`
}

// Kind returns the River job kind.
func (CleanupJobArgs) Kind() string { return CleanupJobKind }

// InsertOpts prevents overlapping history cleanup jobs.
func (CleanupJobArgs) InsertOpts() river.InsertOpts { return backgroundjobs.SingletonInsertOpts() }

// SnapshotWorker records current osquery aggregates.
type SnapshotWorker struct {
	river.WorkerDefaults[SnapshotJobArgs]

	store *Store
}

// NewSnapshotWorker returns an osquery snapshot worker.
func NewSnapshotWorker(store *Store) *SnapshotWorker {
	return &SnapshotWorker{store: store}
}

// Work records one five-minute status point.
func (w *SnapshotWorker) Work(ctx context.Context, _ *river.Job[SnapshotJobArgs]) error {
	return w.store.Snapshot(ctx, time.Now())
}

// CleanupWorker removes osquery history outside the configured retention window.
type CleanupWorker struct {
	river.WorkerDefaults[CleanupJobArgs]

	store  *Store
	logger *slog.Logger
}

// NewCleanupWorker returns an osquery history cleanup worker.
func NewCleanupWorker(store *Store, logger *slog.Logger) *CleanupWorker {
	return &CleanupWorker{store: store, logger: logger}
}

// Work runs one history-retention sweep.
func (w *CleanupWorker) Work(ctx context.Context, job *river.Job[CleanupJobArgs]) error {
	started := time.Now()
	result, err := w.store.SweepBefore(ctx, started.AddDate(0, 0, -job.Args.RetentionDays))
	output := cleanupOutput{CleanupResult: result, DurationMS: int(time.Since(started).Milliseconds())}
	if err != nil {
		output.Error = err.Error()
		return errors.Join(err, river.RecordOutput(ctx, output))
	}
	if result.HostPoints > 0 || result.PolicyPoints > 0 {
		w.logger.InfoContext(ctx, "osquery history cleanup complete",
			"operation", "sweep",
			"host_points", result.HostPoints,
			"policy_points", result.PolicyPoints,
			"duration_ms", output.DurationMS,
		)
	}
	return river.RecordOutput(ctx, output)
}

type cleanupOutput struct {
	CleanupResult

	DurationMS int    `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}
