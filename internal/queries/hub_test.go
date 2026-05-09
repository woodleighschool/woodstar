package queries

import (
	"sync"
	"testing"
	"time"
)

func TestHubPublishesToSubscribers(t *testing.T) {
	hub := NewHub()
	events, release := hub.Subscribe(12)
	defer release()

	hub.Publish(12, LiveQueryEvent{HostID: 4, Status: "success"})

	got := <-events
	if got.HostID != 4 || got.Status != "success" {
		t.Fatalf("event = %#v, want host 4 success", got)
	}
}

func TestHubReleaseStopsDelivery(t *testing.T) {
	hub := NewHub()
	events, release := hub.Subscribe(12)
	release()

	hub.Publish(12, LiveQueryEvent{HostID: 4, Status: "success"})

	select {
	case got, ok := <-events:
		if ok {
			t.Fatalf("received event after release: %#v", got)
		}
	default:
		t.Fatalf("released subscription channel was not closed")
	}
}

func TestHubFansOutToAllSubscribers(t *testing.T) {
	hub := NewHub()
	a, releaseA := hub.Subscribe(7)
	defer releaseA()
	b, releaseB := hub.Subscribe(7)
	defer releaseB()

	hub.Publish(7, LiveQueryEvent{HostID: 1, Status: "success"})

	if got := <-a; got.HostID != 1 || got.Status != "success" {
		t.Fatalf("subscriber a got %#v", got)
	}
	if got := <-b; got.HostID != 1 || got.Status != "success" {
		t.Fatalf("subscriber b got %#v", got)
	}
}

func TestHubDoesNotBlockSlowSubscribers(t *testing.T) {
	hub := NewHub()
	events, release := hub.Subscribe(9)
	defer release()

	// Fill the cap-32 buffer plus extras; Publish must never block on a slow
	// consumer. The extras land on the floor — we verify only that Publish
	// returned and the first 32 are queued.
	for range 64 {
		hub.Publish(9, LiveQueryEvent{HostID: 1, Status: "success"})
	}

	delivered := 0
	for {
		select {
		case <-events:
			delivered++
		default:
			if delivered != 32 {
				t.Fatalf("delivered = %d, want 32 (channel cap)", delivered)
			}
			return
		}
	}
}

func TestHubPublishIsSafeUnderConcurrency(t *testing.T) {
	hub := NewHub()
	events, release := hub.Subscribe(11)
	defer release()

	const publishers = 4
	const perPublisher = 8

	var wg sync.WaitGroup
	wg.Add(publishers)
	for range publishers {
		go func() {
			defer wg.Done()
			for range perPublisher {
				hub.Publish(11, LiveQueryEvent{HostID: 1, Status: "success"})
			}
		}()
	}
	wg.Wait()

	deadline := time.After(50 * time.Millisecond)
	delivered := 0
	for {
		select {
		case <-events:
			delivered++
			if delivered == publishers*perPublisher {
				return
			}
		case <-deadline:
			// Some events may have been dropped if subscriber lagged behind
			// publishers; we only assert no race and that *some* events arrived.
			if delivered == 0 {
				t.Fatal("no events delivered under concurrent publish")
			}
			return
		}
	}
}

func TestHubReleaseUnknownSubscriberIsNoOp(_ *testing.T) {
	hub := NewHub()
	_, release := hub.Subscribe(3)
	release()
	// Releasing a second time on a now-empty campaign is a safe no-op.
	release()
}
