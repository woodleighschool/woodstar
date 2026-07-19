package storage

import (
	"context"
	"log/slog"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
)

func TestDeletionWorkerStopCancelsInFlightDeletion(t *testing.T) {
	db, ctx := dbtest.Open(t)
	backend := &blockingDeletionBackend{
		started:  make(chan struct{}),
		canceled: make(chan struct{}),
	}
	objects := NewObjectStore(db, backend)
	object, err := objects.CreatePending(ctx, "munki/icons", "icon.png")
	if err != nil {
		t.Fatalf("create pending object: %v", err)
	}
	if err := objects.RequestDeletion(ctx, db.Pool(), object.ID); err != nil {
		t.Fatalf("request deletion: %v", err)
	}

	worker := StartDeletionWorker(t.Context(), objects, slog.New(slog.DiscardHandler))
	<-backend.started
	worker.Stop()
	<-backend.canceled
}

type blockingDeletionBackend struct {
	started  chan struct{}
	canceled chan struct{}
}

func (b *blockingDeletionBackend) Delete(ctx context.Context, _ string) error {
	close(b.started)
	<-ctx.Done()
	close(b.canceled)
	return ctx.Err()
}
