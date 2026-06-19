package livequery

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRecordResultPublishesResultAndCloses(t *testing.T) {
	m := NewManager()
	handle := m.Start("select 1", []int64{4})

	events, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	defer release()

	m.RecordResult(handle.ID, 4, "mac-4", StatusSuccess, json.RawMessage(`[{"answer":"1"}]`), "")

	result := receiveEvent(t, events)
	if result.HostID != 4 || result.HostName != "mac-4" || result.Status != StatusSuccess {
		t.Fatalf("result = %#v, want host 4 success", result)
	}
	if string(result.Data) != `[{"answer":"1"}]` {
		t.Fatalf("data = %s, want query rows", result.Data)
	}

	assertClosed(t, events)
}

func TestPendingForHostClearsAfterResult(t *testing.T) {
	m := NewManager()
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

func TestStartReportsUniqueResolvedHosts(t *testing.T) {
	m := NewManager()
	handle := m.Start("select 1", []int64{4, 4, 5})

	if handle.ResolvedHostCount != 2 {
		t.Fatalf("ResolvedHostCount = %d, want unique host count 2", handle.ResolvedHostCount)
	}
}

func TestSubscribeCompletedQueryReceivesClosedChannel(t *testing.T) {
	m := NewManager()
	handle := m.Start("select 1", nil)

	events, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	defer release()

	assertClosed(t, events)
}

func TestOrphanedRunStopsPendingHostsAfterStreamDisconnect(t *testing.T) {
	m := NewManager()
	handle := m.Start("select 1", []int64{4, 5})

	events, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	release()
	assertClosed(t, events)

	m.stopOrphan(handle.ID)
	if work := m.PendingForHost(4); len(work) != 0 {
		t.Fatalf("work for orphaned host = %#v, want none", work)
	}
	if work := m.PendingForHost(5); len(work) != 0 {
		t.Fatalf("work for orphaned host = %#v, want none", work)
	}
}

func TestStopClearsPendingHostsAndCloses(t *testing.T) {
	m := NewManager()
	handle := m.Start("select 1", []int64{4, 5})

	events, release, err := m.Subscribe(handle.ID)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	defer release()

	if err := m.Stop(handle.ID); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if work := m.PendingForHost(4); len(work) != 0 {
		t.Fatalf("work for stopped host = %#v, want none", work)
	}
	if work := m.PendingForHost(5); len(work) != 0 {
		t.Fatalf("work for stopped host = %#v, want none", work)
	}

	first := receiveEvent(t, events)
	second := receiveEvent(t, events)
	if first.Status != StatusStopped || second.Status != StatusStopped {
		t.Fatalf("stopped events = %#v %#v, want stopped", first, second)
	}
	seen := map[int64]bool{first.HostID: true, second.HostID: true}
	if !seen[4] || !seen[5] {
		t.Fatalf("stopped hosts = %#v, want hosts 4 and 5", seen)
	}

	assertClosed(t, events)
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

func assertClosed(t *testing.T, events <-chan Event) {
	t.Helper()
	select {
	case _, ok := <-events:
		if ok {
			t.Fatal("event channel remained open")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for event channel to close")
	}
}
