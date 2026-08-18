package activity

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/woodleighschool/woodstar/internal/backgroundjobs"
)

const (
	CleanupJobKind     = "activity_cleanup"
	CleanupJobInterval = 24 * time.Hour
)

type cleanupStore interface {
	SweepBefore(context.Context, time.Time) (int, error)
}

// CleanupJobArgs identifies a scheduled activity-retention sweep.
type CleanupJobArgs struct {
	Trigger       backgroundjobs.Trigger `json:"trigger"`
	RetentionDays int                    `json:"retention_days"`
}

// Kind returns the River job kind.
func (CleanupJobArgs) Kind() string { return CleanupJobKind }

// InsertOpts prevents overlapping activity cleanup jobs.
func (CleanupJobArgs) InsertOpts() river.InsertOpts { return backgroundjobs.SingletonInsertOpts() }

// CleanupWorker removes activity outside the configured retention window.
type CleanupWorker struct {
	river.WorkerDefaults[CleanupJobArgs]

	store  cleanupStore
	logger *slog.Logger
}

// NewCleanupWorker returns an activity cleanup worker.
func NewCleanupWorker(store cleanupStore, logger *slog.Logger) *CleanupWorker {
	return &CleanupWorker{store: store, logger: logger}
}

// Work runs one retention sweep.
func (w *CleanupWorker) Work(ctx context.Context, job *river.Job[CleanupJobArgs]) error {
	started := time.Now()
	deleted, err := w.store.SweepBefore(ctx, started.AddDate(0, 0, -job.Args.RetentionDays))
	output := cleanupOutput{Deleted: deleted, DurationMS: int(time.Since(started).Milliseconds())}
	if err != nil {
		output.Error = err.Error()
		return errors.Join(err, river.RecordOutput(ctx, output))
	}
	if deleted > 0 {
		w.logger.InfoContext(ctx, "activity cleanup complete",
			"operation", "sweep",
			"activities", deleted,
			"duration_ms", output.DurationMS,
		)
	}
	return river.RecordOutput(ctx, output)
}

type cleanupOutput struct {
	Deleted    int    `json:"deleted"`
	DurationMS int    `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}
