package events

import (
	"context"
	"log/slog"
	"time"

	"github.com/woodleighschool/woodstar/internal/job"
)

const (
	defaultRetentionDays = 90
	defaultSweepInterval = time.Hour
)

type CleanupOptions struct {
	RetentionDays int
	SweepInterval time.Duration
}

type CleanupStore interface {
	SweepEventsBefore(context.Context, time.Time) (int, error)
}

func StartCleanup(
	ctx context.Context,
	store CleanupStore,
	options CleanupOptions,
	logger *slog.Logger,
) *job.Handle {
	if store == nil {
		return nil
	}
	if options.RetentionDays <= 0 {
		options.RetentionDays = defaultRetentionDays
	}
	if options.SweepInterval <= 0 {
		options.SweepInterval = defaultSweepInterval
	}
	return job.Start(ctx, func(ctx context.Context) {
		cleanupLoop(ctx, store, options, logger)
	})
}

func cleanupLoop(ctx context.Context, store CleanupStore, options CleanupOptions, logger *slog.Logger) {
	sweep(ctx, store, options, logger)

	ticker := time.NewTicker(options.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep(ctx, store, options, logger)
		}
	}
}

func sweep(ctx context.Context, store CleanupStore, options CleanupOptions, logger *slog.Logger) {
	cutoff := time.Now().AddDate(0, 0, -options.RetentionDays)
	if _, err := store.SweepEventsBefore(ctx, cutoff); err != nil && logger != nil {
		logger.WarnContext(ctx, "Santa event cleanup failed", "err", err)
	}
}
