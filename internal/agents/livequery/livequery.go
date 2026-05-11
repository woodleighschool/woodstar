// Package livequery runs ephemeral live queries entirely in-process and fans
// result events out to SSE subscribers.
package livequery

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ErrLiveQueryNotFound is returned when the manager has no live query for an id.
var ErrLiveQueryNotFound = errors.New("live query not found")

// LiveQueryStatus is the per-host outcome reported back to the SSE stream.
type LiveQueryStatus string

const (
	LiveStatusSuccess LiveQueryStatus = "success"
	LiveStatusError   LiveQueryStatus = "error"
	LiveStatusTimeout LiveQueryStatus = "timeout"
)

// LiveQueryWork is one queued live query for a host (read by /distributed/read).
type LiveQueryWork struct {
	QueryID int64
	SQL     string
}

// LiveQueryHandle is the public summary of a started live query.
type LiveQueryHandle struct {
	ID                int64
	SQL               string
	StartedAt         time.Time
	ResolvedHostCount int
}

// LiveQueryEvent is published to subscribers for SSE delivery.
type LiveQueryEvent struct {
	HostID   int64           `json:"host_id,omitempty"`
	HostName string          `json:"host_name,omitempty"`
	Status   string          `json:"status"`
	Data     json.RawMessage `json:"data,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// LiveQueryManager runs ephemeral live queries entirely in-process.
// Fan-out subscriber state is held inline — no separate Hub type.
type LiveQueryManager struct {
	timeout time.Duration

	// active query state
	next   atomic.Int64
	mu     sync.RWMutex
	active map[int64]*liveQuery

	// fan-out subscriber state (inlined from Hub)
	subsMu  sync.RWMutex
	subs    map[int64]map[int64]chan LiveQueryEvent
	subNext atomic.Int64
}

type liveQuery struct {
	id        int64
	sql       string
	startedAt time.Time
	pending   map[int64]struct{}
	timer     *time.Timer
	stopped   bool
}

// NewLiveQueryManager returns a manager that times out individual live queries
// after the given duration.
func NewLiveQueryManager(timeout time.Duration) *LiveQueryManager {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &LiveQueryManager{
		timeout: timeout,
		active:  make(map[int64]*liveQuery),
		subs:    make(map[int64]map[int64]chan LiveQueryEvent),
	}
}

// Start registers a live query against a resolved host set and arms its
// timeout. The returned handle is what the admin sees in the create response.
func (m *LiveQueryManager) Start(sql string, hostIDs []int64) *LiveQueryHandle {
	id := m.next.Add(1)
	pending := make(map[int64]struct{}, len(hostIDs))
	for _, hostID := range hostIDs {
		pending[hostID] = struct{}{}
	}
	q := &liveQuery{
		id:        id,
		sql:       sql,
		startedAt: time.Now().UTC(),
		pending:   pending,
	}

	m.mu.Lock()
	m.active[id] = q
	if len(pending) == 0 {
		// No targets — synthesize an immediate completion so the SSE stream
		// closes cleanly.
		m.mu.Unlock()
		m.fanPublish(id, LiveQueryEvent{Status: "completed"})
		m.removeLocked(id)
		return &LiveQueryHandle{ID: id, SQL: sql, StartedAt: q.startedAt}
	}
	q.timer = time.AfterFunc(m.timeout, func() { m.expire(id) })
	m.mu.Unlock()

	return &LiveQueryHandle{
		ID:                id,
		SQL:               sql,
		StartedAt:         q.startedAt,
		ResolvedHostCount: len(hostIDs),
	}
}

// PendingForHost returns live queries currently targeting host.
func (m *LiveQueryManager) PendingForHost(hostID int64) []LiveQueryWork {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]LiveQueryWork, 0)
	for _, q := range m.active {
		if _, pending := q.pending[hostID]; !pending {
			continue
		}
		out = append(out, LiveQueryWork{QueryID: q.id, SQL: q.sql})
	}
	return out
}

// RecordResult marks a host as having responded for a live query, publishes
// the result event, and finishes the query if no hosts remain pending.
func (m *LiveQueryManager) RecordResult(
	queryID int64,
	hostID int64,
	hostName string,
	status LiveQueryStatus,
	data json.RawMessage,
	errMsg string,
) {
	m.mu.Lock()
	q, ok := m.active[queryID]
	if !ok || q.stopped {
		m.mu.Unlock()
		return
	}
	if _, pending := q.pending[hostID]; !pending {
		m.mu.Unlock()
		return
	}
	delete(q.pending, hostID)
	finished := len(q.pending) == 0
	if finished {
		q.stopped = true
		if q.timer != nil {
			q.timer.Stop()
		}
	}
	m.mu.Unlock()

	m.fanPublish(queryID, LiveQueryEvent{
		HostID:   hostID,
		HostName: hostName,
		Status:   string(status),
		Data:     data,
		Error:    errMsg,
	})
	if finished {
		m.fanPublish(queryID, LiveQueryEvent{Status: "completed"})
		m.removeLocked(queryID)
	}
}

// Subscribe returns the live event channel for queryID and a release function.
// Errors with ErrLiveQueryNotFound when the query has already completed/timed out.
func (m *LiveQueryManager) Subscribe(queryID int64) (<-chan LiveQueryEvent, func(), error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.active[queryID]; !ok {
		return nil, nil, ErrLiveQueryNotFound
	}

	ch, release := m.fanSubscribe(queryID)
	return ch, release, nil
}

func (m *LiveQueryManager) expire(queryID int64) {
	m.mu.Lock()
	q, ok := m.active[queryID]
	if !ok || q.stopped {
		m.mu.Unlock()
		return
	}
	q.stopped = true
	timedOut := make([]int64, 0, len(q.pending))
	for hostID := range q.pending {
		timedOut = append(timedOut, hostID)
	}
	q.pending = nil
	m.mu.Unlock()

	for _, hostID := range timedOut {
		m.fanPublish(queryID, LiveQueryEvent{
			HostID: hostID,
			Status: string(LiveStatusTimeout),
		})
	}
	m.fanPublish(queryID, LiveQueryEvent{Status: "completed"})
	m.removeLocked(queryID)
}

func (m *LiveQueryManager) removeLocked(queryID int64) {
	m.mu.Lock()
	delete(m.active, queryID)
	m.mu.Unlock()
}

// fanSubscribe registers a buffered event channel for queryID and returns a
// release function. The caller must invoke release when done.
func (m *LiveQueryManager) fanSubscribe(queryID int64) (<-chan LiveQueryEvent, func()) {
	id := m.subNext.Add(1)
	ch := make(chan LiveQueryEvent, 32)

	m.subsMu.Lock()
	if m.subs[queryID] == nil {
		m.subs[queryID] = make(map[int64]chan LiveQueryEvent)
	}
	m.subs[queryID][id] = ch
	m.subsMu.Unlock()

	return ch, func() {
		m.subsMu.Lock()
		defer m.subsMu.Unlock()
		subs := m.subs[queryID]
		if subs == nil {
			return
		}
		if _, ok := subs[id]; !ok {
			return
		}
		delete(subs, id)
		if len(subs) == 0 {
			delete(m.subs, queryID)
		}
		close(ch)
	}
}

// fanPublish delivers event to all current subscribers without blocking.
func (m *LiveQueryManager) fanPublish(queryID int64, event LiveQueryEvent) {
	m.subsMu.RLock()
	defer m.subsMu.RUnlock()
	for _, ch := range m.subs[queryID] {
		select {
		case ch <- event:
		default:
		}
	}
}
