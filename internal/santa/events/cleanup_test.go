package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/santa/events"
)

func TestCleanupStopDoesNotWaitForStalledSweep(t *testing.T) {
	store := &blockingStore{started: make(chan struct{})}
	cleanup := events.StartCleanup(t.Context(), store, events.CleanupOptions{
		RetentionDays: 1,
		SweepInterval: time.Nanosecond,
	}, nil)

	select {
	case <-store.started:
	case <-time.After(time.Second):
		t.Fatal("cleanup sweep did not start")
	}

	done := make(chan struct{})
	go func() {
		cleanup.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cleanup stop waited for stalled sweep")
	}
}

type blockingStore struct {
	started chan struct{}
}

func (s *blockingStore) SweepEventsBefore(context.Context, time.Time) (int, error) {
	select {
	case <-s.started:
	default:
		close(s.started)
	}
	select {}
}
