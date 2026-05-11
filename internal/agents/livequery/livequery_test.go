package livequery

import (
	"sync"
	"testing"
	"time"
)

// fanSubscribe and fanPublish are tested here as package-internal methods via
// the white-box test. The Hub type no longer exists; this package owns all
// fan-out state inside LiveQueryManager.

func newTestManager() *LiveQueryManager {
	return NewLiveQueryManager(time.Minute)
}

func TestFanPublishesToSubscribers(t *testing.T) {
	m := newTestManager()
	events, release := m.fanSubscribe(12)
	defer release()

	m.fanPublish(12, LiveQueryEvent{HostID: 4, Status: "success"})

	got := <-events
	if got.HostID != 4 || got.Status != "success" {
		t.Fatalf("event = %#v, want host 4 success", got)
	}
}

func TestFanReleaseStopsDelivery(t *testing.T) {
	m := newTestManager()
	events, release := m.fanSubscribe(12)
	release()

	m.fanPublish(12, LiveQueryEvent{HostID: 4, Status: "success"})

	select {
	case got, ok := <-events:
		if ok {
			t.Fatalf("received event after release: %#v", got)
		}
	default:
		t.Fatalf("released subscription channel was not closed")
	}
}

func TestFanFansOutToAllSubscribers(t *testing.T) {
	m := newTestManager()
	a, releaseA := m.fanSubscribe(7)
	defer releaseA()
	b, releaseB := m.fanSubscribe(7)
	defer releaseB()

	m.fanPublish(7, LiveQueryEvent{HostID: 1, Status: "success"})

	if got := <-a; got.HostID != 1 || got.Status != "success" {
		t.Fatalf("subscriber a got %#v", got)
	}
	if got := <-b; got.HostID != 1 || got.Status != "success" {
		t.Fatalf("subscriber b got %#v", got)
	}
}

func TestFanDoesNotBlockSlowSubscribers(t *testing.T) {
	m := newTestManager()
	events, release := m.fanSubscribe(9)
	defer release()

	// Fill the cap-32 buffer plus extras; fanPublish must never block on a slow
	// consumer. The extras land on the floor — we verify only that fanPublish
	// returned and the first 32 are queued.
	for range 64 {
		m.fanPublish(9, LiveQueryEvent{HostID: 1, Status: "success"})
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

func TestFanPublishIsSafeUnderConcurrency(t *testing.T) {
	m := newTestManager()
	events, release := m.fanSubscribe(11)
	defer release()

	const publishers = 4
	const perPublisher = 8

	var wg sync.WaitGroup
	wg.Add(publishers)
	for range publishers {
		go func() {
			defer wg.Done()
			for range perPublisher {
				m.fanPublish(11, LiveQueryEvent{HostID: 1, Status: "success"})
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

func TestFanReleaseUnknownSubscriberIsNoOp(_ *testing.T) {
	m := newTestManager()
	_, release := m.fanSubscribe(3)
	release()
	// Releasing a second time on a now-empty campaign is a safe no-op.
	release()
}
