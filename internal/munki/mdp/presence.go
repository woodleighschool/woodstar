package mdp

import "sync"

// Presence holds the latest worker state observed by this Woodstar process.
// The protocol records connections and rejected upgrades, while the store uses
// connected workers for redirects and includes every state in the admin view.
type Presence struct {
	mu        sync.RWMutex
	connected map[int64]DistributionPointWorker
	rejected  map[int64]DistributionPointWorker
}

// NewPresence returns an empty worker-state registry.
func NewPresence() *Presence {
	return &Presence{
		connected: map[int64]DistributionPointWorker{},
		rejected:  map[int64]DistributionPointWorker{},
	}
}

// Worker returns the latest worker state observed by this process. A nil
// receiver reports no worker so zero-value wiring for schema generation is safe.
func (p *Presence) Worker(id int64) (DistributionPointWorker, bool) {
	if p == nil {
		return DistributionPointWorker{}, false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if worker, ok := p.connected[id]; ok {
		return worker, true
	}
	worker, ok := p.rejected[id]
	return worker, ok
}

// Connect records a distribution point's current worker connection and
// supersedes any incompatibility observed before that connection.
func (p *Presence) Connect(id int64, worker DistributionPointWorker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connected[id] = worker
	delete(p.rejected, id)
}

// Reject records the latest incompatible worker. A live compatible worker
// remains the visible state until it disconnects.
func (p *Presence) Reject(id int64, worker DistributionPointWorker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rejected[id] = worker
}

// Disconnect clears a distribution point's current connection, revealing any
// incompatible worker observed while that connection was live.
func (p *Presence) Disconnect(id int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.connected, id)
}

// Clear removes every worker state recorded for a distribution point.
func (p *Presence) Clear(id int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.connected, id)
	delete(p.rejected, id)
}
