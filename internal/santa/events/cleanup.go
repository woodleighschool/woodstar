package events

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type CleanupStore interface {
	SweepEventsBefore(ctx context.Context, cutoff time.Time) (int, error)
}

// Cleanup owns the Santa event-retention background loop.
type Cleanup struct {
	stop context.CancelFunc
	done <-chan struct{}
}

// Stop cancels the cleanup loop and waits for it to exit.
func (c *Cleanup) Stop() {
	c.stop()
	<-c.done
}

func StartCleanup(
	ctx context.Context,
	store CleanupStore,
	retentionDays int,
	sweepInterval time.Duration,
	logger *slog.Logger,
) *Cleanup {
	ctx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		cleanupLoop(ctx, store, retentionDays, sweepInterval, logger)
	}()
	return &Cleanup{stop: stop, done: done}
}

func cleanupLoop(
	ctx context.Context,
	store CleanupStore,
	retentionDays int,
	sweepInterval time.Duration,
	logger *slog.Logger,
) {
	sweep(ctx, store, retentionDays, logger)

	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep(ctx, store, retentionDays, logger)
		}
	}
}

func sweep(ctx context.Context, store CleanupStore, retentionDays int, logger *slog.Logger) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	if _, err := store.SweepEventsBefore(ctx, cutoff); err != nil && !errors.Is(err, context.Canceled) {
		logger.WarnContext(ctx, "santa event cleanup failed", "operation", "sweep", "err", err)
	}
}

// SweepEventsBefore deletes Santa events that occurred before cutoff.
func (s *Store) SweepEventsBefore(ctx context.Context, cutoff time.Time) (int, error) {
	var deleted int
	err := s.db.Pool().QueryRow(ctx, `
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
