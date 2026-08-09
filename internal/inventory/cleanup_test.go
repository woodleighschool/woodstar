package inventory_test

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/inventory"
)

func TestCleanupStopWaitsForInFlightSweep(t *testing.T) {
	store := &blockingCleanupStore{started: make(chan struct{})}
	cleanup := inventory.StartCleanup(t.Context(), store, slog.New(slog.DiscardHandler))

	select {
	case <-store.started:
	case <-time.After(time.Second):
		t.Fatal("inventory cleanup sweep did not start")
	}

	cleanup.Stop()
	if !store.done.Load() {
		t.Fatal("inventory cleanup stop returned before sweep observed cancellation")
	}
}

type blockingCleanupStore struct {
	started chan struct{}
	done    atomic.Bool
}

func (s *blockingCleanupStore) PruneUnreferencedSoftware(ctx context.Context) (inventory.CleanupResult, error) {
	close(s.started)
	<-ctx.Done()
	s.done.Store(true)
	return inventory.CleanupResult{}, ctx.Err()
}
