package clientresources

import (
	"context"
	"time"
)

const objectCleanupTimeout = 15 * time.Second

type objectCleaner interface {
	DeleteUnreferenced(context.Context, ...int64) error
}

func cleanupObjects(ctx context.Context, objects objectCleaner, ids ...int64) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), objectCleanupTimeout)
	defer cancel()
	return objects.DeleteUnreferenced(cleanupCtx, ids...)
}
