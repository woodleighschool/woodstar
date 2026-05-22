package reports

import (
	"context"
	"log/slog"
	"time"
)

const defaultMaxReportRows = 1000

// CleanupOptions controls periodic report-result maintenance.
type CleanupOptions struct {
	MaxReportRows int
}

// StartCleanup starts the periodic report-row trimmer. The goroutine exits
// when ctx is cancelled.
func StartCleanup(
	ctx context.Context,
	store *Store,
	options CleanupOptions,
	logger *slog.Logger,
) {
	if store == nil {
		return
	}
	if options.MaxReportRows <= 0 {
		options.MaxReportRows = defaultMaxReportRows
	}
	go reportTrimLoop(ctx, store, options, logger)
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
