package santa

import (
	"context"
	"log/slog"
	"time"
)

const (
	defaultEventRetentionDays = 90
	defaultEventSweepInterval = time.Hour
)

type EventCleanupOptions struct {
	RetentionDays int
	SweepInterval time.Duration
}

type EventCleanup struct {
	stop context.CancelFunc
	done <-chan struct{}
}

func (c *EventCleanup) Stop() {
	if c == nil {
		return
	}
	c.stop()
	<-c.done
}

func StartEventCleanup(
	ctx context.Context,
	store *Store,
	options EventCleanupOptions,
	logger *slog.Logger,
) *EventCleanup {
	if store == nil {
		return nil
	}
	if options.RetentionDays <= 0 {
		options.RetentionDays = defaultEventRetentionDays
	}
	if options.SweepInterval <= 0 {
		options.SweepInterval = defaultEventSweepInterval
	}
	ctx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		eventCleanupLoop(ctx, store, options, logger)
	}()
	return &EventCleanup{stop: stop, done: done}
}

func eventCleanupLoop(ctx context.Context, store *Store, options EventCleanupOptions, logger *slog.Logger) {
	ticker := time.NewTicker(options.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().AddDate(0, 0, -options.RetentionDays)
			if _, err := store.SweepEventsBefore(ctx, cutoff); err != nil && logger != nil {
				logger.WarnContext(ctx, "Santa event cleanup failed", "err", err)
			}
		}
	}
}
