package clientresources

import (
	"context"
	"errors"
	"time"

	"github.com/woodleighschool/goodies/bloby"
)

const uploadCleanupTimeout = 15 * time.Second

type uploadCleaner interface {
	Delete(ctx context.Context, uploadID int64, prefix string) error
}

func cleanupUploads(ctx context.Context, uploads uploadCleaner, prefix string, ids ...int64) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), uploadCleanupTimeout)
	defer cancel()
	for _, id := range ids {
		if err := uploads.Delete(cleanupCtx, id, prefix); err != nil &&
			!errors.Is(err, bloby.ErrConflict) &&
			!errors.Is(err, bloby.ErrNotFound) {
			return err
		}
	}
	return nil
}
