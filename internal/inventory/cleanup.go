package inventory

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/woodleighschool/woodstar/internal/backgroundjobs"
)

const (
	CleanupJobKind     = "inventory_software_cleanup"
	CleanupJobInterval = time.Hour
)

// CleanupStore removes software inventory that is no longer observed on a host.
type CleanupStore interface {
	PruneUnreferencedSoftware(ctx context.Context) (CleanupResult, error)
}

// CleanupJobArgs identifies a scheduled inventory cleanup.
type CleanupJobArgs struct {
	Trigger backgroundjobs.Trigger `json:"trigger"`
}

func (CleanupJobArgs) Kind() string { return CleanupJobKind }

func (CleanupJobArgs) InsertOpts() river.InsertOpts { return backgroundjobs.SingletonInsertOpts() }

// CleanupWorker removes unreferenced software versions and titles.
type CleanupWorker struct {
	river.WorkerDefaults[CleanupJobArgs]

	store  CleanupStore
	logger *slog.Logger
}

func NewCleanupWorker(store CleanupStore, logger *slog.Logger) *CleanupWorker {
	return &CleanupWorker{store: store, logger: logger}
}

func (w *CleanupWorker) Work(ctx context.Context, _ *river.Job[CleanupJobArgs]) error {
	started := time.Now()
	result, err := w.store.PruneUnreferencedSoftware(ctx)
	output := cleanupJobOutput{
		CleanupResult: result,
		DurationMS:    int(time.Since(started).Milliseconds()),
	}
	if err != nil {
		output.Error = err.Error()
		return errors.Join(err, river.RecordOutput(ctx, output))
	}
	if result.SoftwareVersions > 0 || result.SoftwareTitles > 0 {
		w.logger.InfoContext(ctx, "inventory software cleanup complete",
			"operation", "sweep",
			"software_versions", result.SoftwareVersions,
			"software_titles", result.SoftwareTitles,
			"duration_ms", output.DurationMS,
		)
	}
	return river.RecordOutput(ctx, output)
}

type cleanupJobOutput struct {
	CleanupResult

	DurationMS int    `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}
