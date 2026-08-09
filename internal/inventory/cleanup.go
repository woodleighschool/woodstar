package inventory

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

const cleanupInterval = time.Hour

// CleanupStore removes software inventory that is no longer observed on a host.
type CleanupStore interface {
	PruneUnreferencedSoftware(ctx context.Context) (CleanupResult, error)
}

// Cleanup owns the software inventory cleanup loop.
type Cleanup struct {
	stop context.CancelFunc
	done <-chan struct{}
}

// Stop cancels the cleanup loop and waits for it to exit.
func (c *Cleanup) Stop() {
	c.stop()
	<-c.done
}

// StartCleanup starts an immediate cleanup pass followed by hourly passes.
func StartCleanup(ctx context.Context, store CleanupStore, logger *slog.Logger) *Cleanup {
	ctx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		cleanupLoop(ctx, store, logger)
	}()
	return &Cleanup{stop: stop, done: done}
}

func cleanupLoop(ctx context.Context, store CleanupStore, logger *slog.Logger) {
	sweep(ctx, store, logger)
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep(ctx, store, logger)
		}
	}
}

func sweep(ctx context.Context, store CleanupStore, logger *slog.Logger) {
	started := time.Now()
	result, err := store.PruneUnreferencedSoftware(ctx)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.WarnContext(ctx, "inventory software cleanup failed", "operation", "sweep", "err", err)
		}
		return
	}
	if result.SoftwareVersions == 0 && result.SoftwareTitles == 0 {
		return
	}
	logger.InfoContext(ctx, "inventory software cleanup complete",
		"operation", "sweep",
		"software_versions", result.SoftwareVersions,
		"software_titles", result.SoftwareTitles,
		"duration_ms", time.Since(started).Milliseconds(),
	)
}
