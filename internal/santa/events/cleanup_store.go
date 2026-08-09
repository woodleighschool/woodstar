package events

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/woodleighschool/woodstar/internal/backgroundjobs"
)

const CleanupJobKind = "santa_event_cleanup"

type CleanupStore interface {
	SweepEventsBefore(ctx context.Context, cutoff time.Time) (int, error)
}

// CleanupJobArgs identifies a scheduled Santa event-retention sweep.
type CleanupJobArgs struct {
	Trigger       backgroundjobs.Trigger `json:"trigger"`
	RetentionDays int                    `json:"retention_days"`
}

func (CleanupJobArgs) Kind() string { return CleanupJobKind }

func (CleanupJobArgs) InsertOpts() river.InsertOpts { return backgroundjobs.SingletonInsertOpts() }

// CleanupWorker removes Santa events outside the retention window.
type CleanupWorker struct {
	river.WorkerDefaults[CleanupJobArgs]

	store  CleanupStore
	logger *slog.Logger
}

func NewCleanupWorker(store CleanupStore, logger *slog.Logger) *CleanupWorker {
	return &CleanupWorker{store: store, logger: logger}
}

func (w *CleanupWorker) Work(ctx context.Context, job *river.Job[CleanupJobArgs]) error {
	started := time.Now()
	cutoff := started.AddDate(0, 0, -job.Args.RetentionDays)
	deleted, err := w.store.SweepEventsBefore(ctx, cutoff)
	output := cleanupJobOutput{
		Deleted:    deleted,
		DurationMS: int(time.Since(started).Milliseconds()),
	}
	if err != nil {
		output.Error = err.Error()
		return errors.Join(err, river.RecordOutput(ctx, output))
	}
	if deleted > 0 {
		w.logger.InfoContext(ctx, "santa event cleanup complete",
			"operation", "sweep",
			"events", deleted,
			"duration_ms", output.DurationMS,
		)
	}
	return river.RecordOutput(ctx, output)
}

type cleanupJobOutput struct {
	Deleted    int    `json:"deleted"`
	DurationMS int    `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// SweepEventsBefore deletes Santa events that occurred before cutoff.
func (s *Store) SweepEventsBefore(ctx context.Context, cutoff time.Time) (int, error) {
	var deleted int
	err := s.pool.QueryRow(ctx, `
WITH deleted_execution AS (
	DELETE FROM santa_execution_events
	WHERE santa_execution_events.occurred_at < $1
	RETURNING 1
),
deleted_file_access AS (
	DELETE FROM santa_file_access_events
	WHERE santa_file_access_events.occurred_at < $1
	RETURNING 1
),
deleted_standalone_rules AS (
	DELETE FROM santa_standalone_rule_creation_events
	WHERE santa_standalone_rule_creation_events.occurred_at < $1
	RETURNING 1
)
SELECT
	(SELECT count(*) FROM deleted_execution)::integer
	+ (SELECT count(*) FROM deleted_file_access)::integer
	+ (SELECT count(*) FROM deleted_standalone_rules)::integer AS deleted_count`,
		cutoff,
	).Scan(&deleted)
	if err != nil {
		return 0, err
	}
	return deleted, nil
}
