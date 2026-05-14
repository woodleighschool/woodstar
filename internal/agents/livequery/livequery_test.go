package livequery

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRecordResultPublishesResultAndCompletion(t *testing.T) {
	m := NewManager(time.Minute)
	handle := m.Start("select 1", []int64{4})

	events, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	defer release()

	m.RecordResult(handle.ID, 4, "mac-4", StatusSuccess, json.RawMessage(`[{"answer":"1"}]`), "")

	result := receiveEvent(t, events)
	if result.HostID != 4 || result.HostName != "mac-4" || result.Status != "success" {
		t.Fatalf("result = %#v, want host 4 success", result)
	}
	if string(result.Data) != `[{"answer":"1"}]` {
		t.Fatalf("data = %s, want query rows", result.Data)
	}

	completed := receiveEvent(t, events)
	if completed.Status != "completed" {
		t.Fatalf("completed = %#v, want completed event", completed)
	}
}

func TestPendingForHostClearsAfterResult(t *testing.T) {
	m := NewManager(time.Minute)
	handle := m.Start("select 1", []int64{4, 5})

	if work := m.PendingForHost(4); len(work) != 1 || work[0].QueryID != handle.ID || work[0].SQL != "select 1" {
		t.Fatalf("work for host 4 = %#v, want live query work", work)
	}

	m.RecordResult(handle.ID, 4, "mac-4", StatusSuccess, nil, "")

	if work := m.PendingForHost(4); len(work) != 0 {
		t.Fatalf("work for completed host = %#v, want none", work)
	}
	if work := m.PendingForHost(5); len(work) != 1 || work[0].QueryID != handle.ID {
		t.Fatalf("work for pending host = %#v, want still pending", work)
	}
}

func TestSubscribeCompletedQueryReceivesCompletedEvent(t *testing.T) {
	m := NewManager(time.Minute)
	handle := m.Start("select 1", nil)

	events, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	defer release()

	got := receiveEvent(t, events)
	if got.Status != "completed" {
		t.Fatalf("event = %#v, want completed", got)
	}
}

func TestTimeoutPublishesPendingHostsAndCompletion(t *testing.T) {
	m := NewManager(10 * time.Millisecond)
	handle := m.Start("select 1", []int64{4, 5})

	events, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	defer release()

	first := receiveEvent(t, events)
	second := receiveEvent(t, events)
	if first.Status != "timeout" || second.Status != "timeout" {
		t.Fatalf("timeout events = %#v %#v, want timeouts", first, second)
	}
	seen := map[int64]bool{first.HostID: true, second.HostID: true}
	if !seen[4] || !seen[5] {
		t.Fatalf("timeout hosts = %#v, want hosts 4 and 5", seen)
	}

	completed := receiveEvent(t, events)
	if completed.Status != "completed" {
		t.Fatalf("completed = %#v, want completed event", completed)
	}
}

func receiveEvent(t *testing.T, events <-chan Event) Event {
	t.Helper()
	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("event channel closed")
		}
		return event
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}
