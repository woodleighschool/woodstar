package storage

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const (
	deletionSweepInterval = time.Minute
	deletionTimeout       = 15 * time.Second
	deletionBatchSize     = 100
)

// DeletionWorker retries requested object deletions independently of requests.
type DeletionWorker struct {
	stop context.CancelFunc
	done <-chan struct{}
}

// Stop cancels the deletion loop and waits for it to exit.
func (w *DeletionWorker) Stop() {
	w.stop()
	<-w.done
}

// StartDeletionWorker starts the storage-owned object deletion loop.
func StartDeletionWorker(ctx context.Context, objects *ObjectStore, logger *slog.Logger) *DeletionWorker {
	ctx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		deletionLoop(ctx, objects, logger)
	}()
	return &DeletionWorker{stop: stop, done: done}
}

func deletionLoop(ctx context.Context, objects *ObjectStore, logger *slog.Logger) {
	sweepDeletions(ctx, objects, logger)

	ticker := time.NewTicker(deletionSweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweepDeletions(ctx, objects, logger)
		}
	}
}

func sweepDeletions(ctx context.Context, objects *ObjectStore, logger *slog.Logger) {
	ids, err := objects.listDeletionRequests(ctx, deletionBatchSize)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.WarnContext(ctx, "storage object deletion sweep failed", "operation", "list", "err", err)
		}
		return
	}

	for _, id := range ids {
		deletionCtx, cancel := context.WithTimeout(ctx, deletionTimeout)
		err := objects.Delete(deletionCtx, id)
		cancel()
		switch {
		case err == nil, errors.Is(err, dbutil.ErrNotFound):
		case errors.Is(err, dbutil.ErrConflict):
			if retryErr := objects.retryDeletionLater(ctx, id); retryErr != nil {
				logger.WarnContext(
					ctx,
					"storage object deletion sweep failed",
					"operation",
					"retry",
					"object_id",
					id,
					"err",
					retryErr,
				)
			}
		default:
			if !errors.Is(err, context.Canceled) {
				logger.WarnContext(ctx, "storage object deletion failed", "object_id", id, "err", err)
				if retryErr := objects.retryDeletionLater(ctx, id); retryErr != nil {
					logger.WarnContext(
						ctx,
						"storage object deletion sweep failed",
						"operation",
						"retry",
						"object_id",
						id,
						"err",
						retryErr,
					)
				}
			}
		}
	}
}

func (s *ObjectStore) listDeletionRequests(ctx context.Context, limit int) ([]int64, error) {
	rows, err := s.db.Pool().Query(ctx, `
SELECT id
FROM storage_objects
WHERE deletion_requested_at IS NOT NULL
ORDER BY deletion_requested_at, id
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

func (s *ObjectStore) retryDeletionLater(ctx context.Context, id int64) error {
	_, err := s.db.Pool().Exec(ctx, `
UPDATE storage_objects
SET deletion_requested_at = now(),
    updated_at = now()
WHERE id = $1`, id)
	return err
}
