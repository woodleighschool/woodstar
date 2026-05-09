// Package queries contains cross-cutting query execution infrastructure.
package queries

import (
	"sync"
	"sync/atomic"
)

// Hub fans live query result events out to connected SSE subscribers.
// All state is in-memory; on process restart any in-flight live queries
// are lost.
type Hub struct {
	mu   sync.RWMutex
	subs map[int64]map[int64]chan LiveQueryEvent
	next atomic.Int64
}

// NewHub returns an empty live query event hub.
func NewHub() *Hub {
	return &Hub{subs: make(map[int64]map[int64]chan LiveQueryEvent)}
}

// Subscribe registers a buffered live query event channel and a release
// function. The caller must invoke release when done; release closes the
// channel.
func (h *Hub) Subscribe(queryID int64) (<-chan LiveQueryEvent, func()) {
	if h == nil {
		ch := make(chan LiveQueryEvent)
		close(ch)
		return ch, func() {}
	}
	id := h.next.Add(1)
	ch := make(chan LiveQueryEvent, 32)

	h.mu.Lock()
	if h.subs[queryID] == nil {
		h.subs[queryID] = make(map[int64]chan LiveQueryEvent)
	}
	h.subs[queryID][id] = ch
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		subs := h.subs[queryID]
		if subs == nil {
			return
		}
		if _, ok := subs[id]; !ok {
			return
		}
		delete(subs, id)
		if len(subs) == 0 {
			delete(h.subs, queryID)
		}
		close(ch)
	}
}

// Publish delivers event to current subscribers without blocking. If a
// subscriber's buffer is full the event is dropped for that subscriber —
// slow consumers don't stall the publisher.
func (h *Hub) Publish(queryID int64, event LiveQueryEvent) {
	if h == nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subs[queryID] {
		select {
		case ch <- event:
		default:
		}
	}
}
