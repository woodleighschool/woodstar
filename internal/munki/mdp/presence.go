package mdp

import "sync"

// Presence is the live set of distribution points holding a worker connection.
// The hub writes it on connect and disconnect; the store reads it to gate client
// redirects and to mark the admin view online. It is owned by the wiring layer
// and shared by both, so neither constructs the other.
type Presence struct {
	mu     sync.RWMutex
	online map[int64]struct{}
}

// NewPresence returns an empty presence set.
func NewPresence() *Presence {
	return &Presence{online: map[int64]struct{}{}}
}

// Online reports whether a distribution point currently holds a connection. A
// nil receiver reports false so zero-value wiring (schema generation) is safe.
func (p *Presence) Online(id int64) bool {
	if p == nil {
		return false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.online[id]
	return ok
}

// Connect marks a distribution point online. The hub calls it on register.
func (p *Presence) Connect(id int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.online[id] = struct{}{}
}

// Disconnect marks a distribution point offline. The hub calls it on unregister.
func (p *Presence) Disconnect(id int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.online, id)
}
