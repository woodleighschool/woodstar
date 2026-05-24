package events_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/santa/events"
)

// Stop must wait for the cleanup goroutine to exit so a slow sweep cannot
// still be running while the rest of shutdown advances.
func TestCleanupStopWaitsForInFlightSweep(t *testing.T) {
	store := &observingStore{started: make(chan struct{})}
	cleanup := events.StartCleanup(t.Context(), store, events.CleanupOptions{
		RetentionDays: 1,
		SweepInterval: time.Hour,
	}, nil)

	select {
	case <-store.started:
	case <-time.After(time.Second):
		t.Fatal("cleanup sweep did not start")
	}

	cleanup.Stop()

	if !store.done.Load() {
		t.Fatal("cleanup stop returned before sweep observed cancellation")
	}
}

type observingStore struct {
	started chan struct{}
	done    atomic.Bool
}

func (s *observingStore) SweepEventsBefore(ctx context.Context, _ time.Time) (int, error) {
	select {
	case <-s.started:
	default:
		close(s.started)
	}
	<-ctx.Done()
	s.done.Store(true)
	return 0, ctx.Err()
}
