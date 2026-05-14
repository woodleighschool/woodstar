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

// Status is the per-host outcome reported back to the SSE stream.
type Status string

const (
	StatusSuccess Status = "success"
	StatusError   Status = "error"
	StatusTimeout Status = "timeout"
)

// Work is one queued live query for a host (read by /distributed/read).
type Work struct {
	QueryID int64
	SQL     string
}

// Handle is the public summary of a started live query.
type Handle struct {
	ID                int64
	SQL               string
	StartedAt         time.Time
	ResolvedHostCount int
}

// Event is published to subscribers for SSE delivery.
type Event struct {
	HostID   int64           `json:"host_id,omitempty"`
	HostName string          `json:"host_name,omitempty"`
	Status   string          `json:"status"`
	Data     json.RawMessage `json:"data,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// Manager runs ephemeral live queries entirely in-process.
type Manager struct {
	timeout time.Duration

	next    atomic.Int64
	subNext atomic.Int64

	mu        sync.Mutex
	active    map[int64]*liveQuery
	completed map[int64]struct{}
	subs      map[int64]map[int64]chan Event
}

type liveQuery struct {
	id        int64
	sql       string
	startedAt time.Time
	pending   map[int64]struct{}
	timer     *time.Timer
	stopped   bool
}

// NewManager returns a manager that times out individual live queries after the
// given duration.
func NewManager(timeout time.Duration) *Manager {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Manager{
		timeout:   timeout,
		active:    make(map[int64]*liveQuery),
		completed: make(map[int64]struct{}),
		subs:      make(map[int64]map[int64]chan Event),
	}
}

// Start registers a live query against a resolved host set and arms its
// timeout. The returned handle is what the admin sees in the create response.
func (m *Manager) Start(sql string, hostIDs []int64) Handle {
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
	if len(pending) == 0 {
		m.completed[id] = struct{}{}
		m.mu.Unlock()
		m.forgetCompletedLater(id)
		return Handle{ID: id, SQL: sql, StartedAt: q.startedAt}
	}
	m.active[id] = q
	q.timer = time.AfterFunc(m.timeout, func() { m.expire(id) })
	m.mu.Unlock()

	return Handle{
		ID:                id,
		SQL:               sql,
		StartedAt:         q.startedAt,
		ResolvedHostCount: len(hostIDs),
	}
}

// PendingForHost returns live queries currently targeting host.
func (m *Manager) PendingForHost(hostID int64) []Work {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Work, 0)
	for _, q := range m.active {
		if _, pending := q.pending[hostID]; !pending {
			continue
		}
		out = append(out, Work{QueryID: q.id, SQL: q.sql})
	}
	return out
}

// RecordResult marks a host as having responded for a live query, publishes
// the result event, and finishes the query if no hosts remain pending.
func (m *Manager) RecordResult(
	queryID int64,
	hostID int64,
	hostName string,
	status Status,
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
		delete(m.active, queryID)
		m.completed[queryID] = struct{}{}
	}

	m.publishLocked(queryID, Event{
		HostID:   hostID,
		HostName: hostName,
		Status:   string(status),
		Data:     data,
		Error:    errMsg,
	})
	if finished {
		m.publishLocked(queryID, Event{Status: "completed"})
	}
	m.mu.Unlock()
	if finished {
		m.forgetCompletedLater(queryID)
	}
}

// Subscribe returns the live event channel for queryID and a release function.
// Already-completed queries replay a terminal completed event for late subscribers.
func (m *Manager) Subscribe(queryID int64) (<-chan Event, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.active[queryID]; ok {
		ch, release := m.subscribeLocked(queryID)
		return ch, release, nil
	}
	if _, ok := m.completed[queryID]; ok {
		ch := make(chan Event, 1)
		ch <- Event{Status: "completed"}
		close(ch)
		return ch, func() {}, nil
	}
	return nil, nil, ErrLiveQueryNotFound
}

func (m *Manager) expire(queryID int64) {
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
	delete(m.active, queryID)
	m.completed[queryID] = struct{}{}

	for _, hostID := range timedOut {
		m.publishLocked(queryID, Event{
			HostID: hostID,
			Status: string(StatusTimeout),
		})
	}
	m.publishLocked(queryID, Event{Status: "completed"})
	m.mu.Unlock()
	m.forgetCompletedLater(queryID)
}

func (m *Manager) forgetCompletedLater(queryID int64) {
	time.AfterFunc(m.timeout, func() {
		m.mu.Lock()
		delete(m.completed, queryID)
		m.mu.Unlock()
	})
}

func (m *Manager) fanSubscribe(queryID int64) (<-chan Event, func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.subscribeLocked(queryID)
}

func (m *Manager) subscribeLocked(queryID int64) (<-chan Event, func()) {
	id := m.subNext.Add(1)
	ch := make(chan Event, 32)

	if m.subs[queryID] == nil {
		m.subs[queryID] = make(map[int64]chan Event)
	}
	m.subs[queryID][id] = ch

	return ch, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
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

func (m *Manager) fanPublish(queryID int64, event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishLocked(queryID, event)
}

func (m *Manager) publishLocked(queryID int64, event Event) {
	for _, ch := range m.subs[queryID] {
		select {
		case ch <- event:
		default:
		}
	}
	if event.Status == "completed" {
		for id, ch := range m.subs[queryID] {
			close(ch)
			delete(m.subs[queryID], id)
		}
		delete(m.subs, queryID)
	}
}
