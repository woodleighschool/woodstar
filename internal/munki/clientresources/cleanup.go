package clientresources

import (
	"context"
	"errors"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const objectCleanupTimeout = 15 * time.Second

type objectCleaner interface {
	Delete(context.Context, int64) error
}

func cleanupObjects(ctx context.Context, objects objectCleaner, ids ...int64) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), objectCleanupTimeout)
	defer cancel()
	for _, id := range ids {
		if err := objects.Delete(cleanupCtx, id); err != nil &&
			!errors.Is(err, dbutil.ErrConflict) &&
			!errors.Is(err, dbutil.ErrNotFound) {
			return err
		}
	}
	return nil
}
