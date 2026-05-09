package queries

import (
	"context"
	"log/slog"
	"time"

	"github.com/woodleighschool/woodstar/internal/models"
)

// CleanupOptions controls periodic query-execution maintenance.
type CleanupOptions struct {
	MaxReportRows int
}

// DefaultCleanupOptions returns practical single-binary defaults.
func DefaultCleanupOptions() CleanupOptions {
	return CleanupOptions{MaxReportRows: 1000}
}

// StartCleanup starts the periodic report-row trimmer. The goroutine exits
// when ctx is cancelled. Live queries are not persisted, so no campaign
// timeout/TTL ticker is needed here — the LiveQueryManager handles its own
// per-query timeouts in-process.
func StartCleanup(
	ctx context.Context,
	queries *models.QueryStore,
	options CleanupOptions,
	logger *slog.Logger,
) {
	if queries == nil {
		return
	}
	if options.MaxReportRows <= 0 {
		options.MaxReportRows = DefaultCleanupOptions().MaxReportRows
	}
	go reportTrimLoop(ctx, queries, options, logger)
}

func reportTrimLoop(
	ctx context.Context,
	queries *models.QueryStore,
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
