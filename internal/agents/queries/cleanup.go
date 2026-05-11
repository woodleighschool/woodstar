package queries

import (
	"context"
	"log/slog"
	"time"
)

const defaultMaxReportRows = 1000

// CleanupOptions controls periodic query-execution maintenance.
type CleanupOptions struct {
	MaxReportRows int
}

// StartCleanup starts the periodic report-row trimmer. The goroutine exits
// when ctx is cancelled. Live queries are not persisted, so no campaign
// timeout/TTL ticker is needed here — the LiveQueryManager handles its own
// per-query timeouts in-process.
func StartCleanup(
	ctx context.Context,
	queries *Store,
	options CleanupOptions,
	logger *slog.Logger,
) {
	if queries == nil {
		return
	}
	if options.MaxReportRows <= 0 {
		options.MaxReportRows = defaultMaxReportRows
	}
	go reportTrimLoop(ctx, queries, options, logger)
}

func reportTrimLoop(
	ctx context.Context,
	queries *Store,
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
			if err := queries.TrimResults(ctx, options.MaxReportRows); err != nil && logger != nil {
				logger.WarnContext(ctx, "report row cleanup failed", "err", err)
			}
		}
	}
}
