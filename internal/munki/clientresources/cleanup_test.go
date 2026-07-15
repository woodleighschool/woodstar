package clientresources

import (
	"context"
	"testing"
)

func TestCleanupObjectsDetachesCanceledRequest(t *testing.T) {
	t.Parallel()

	requestCtx, cancel := context.WithCancel(context.Background())
	cancel()
	cleaner := &recordingObjectCleaner{}

	if err := cleanupObjects(requestCtx, cleaner, 42); err != nil {
		t.Fatalf("cleanupObjects() error = %v", err)
	}
	if cleaner.contextErr != nil {
		t.Fatalf("cleanup context error = %v, want nil", cleaner.contextErr)
	}
	if !cleaner.hasDeadline {
		t.Fatal("cleanup context has no deadline")
	}
	if len(cleaner.ids) != 1 || cleaner.ids[0] != 42 {
		t.Fatalf("cleaned IDs = %v, want [42]", cleaner.ids)
	}
}

type recordingObjectCleaner struct {
	contextErr  error
	hasDeadline bool
	ids         []int64
}

func (c *recordingObjectCleaner) DeleteUnreferenced(ctx context.Context, ids ...int64) error {
	c.contextErr = ctx.Err()
	_, c.hasDeadline = ctx.Deadline()
	c.ids = append(c.ids, ids...)
	return ctx.Err()
}
