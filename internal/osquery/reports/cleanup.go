package reports

import (
	"context"
	"log/slog"
	"time"

	"github.com/woodleighschool/woodstar/internal/job"
)

const defaultMaxReportRows = 1000

// CleanupOptions controls periodic report-result maintenance.
type CleanupOptions struct {
	MaxReportRows int
}

// StartCleanup starts the periodic report-row trimmer. The returned handle
// stops the goroutine and waits for it to finish.
func StartCleanup(
	ctx context.Context,
	store *Store,
	options CleanupOptions,
	logger *slog.Logger,
) *job.Handle {
	if store == nil {
		return nil
	}
	if options.MaxReportRows <= 0 {
		options.MaxReportRows = defaultMaxReportRows
	}
	return job.Start(ctx, func(ctx context.Context) {
		reportTrimLoop(ctx, store, options, logger)
	})
}

func reportTrimLoop(
	ctx context.Context,
	store *Store,
	options CleanupOptions,
	logger *slog.Logger,
) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := store.TrimResults(ctx, options.MaxReportRows); err != nil && logger != nil {
				logger.WarnContext(ctx, "report row cleanup failed", "err", err)
			}
		}
	}
}
