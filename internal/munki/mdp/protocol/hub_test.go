package protocol

import (
	"log/slog"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/munki/mdp"
)

func TestNewServerRejectsInvalidBuildVersion(t *testing.T) {
	server, err := NewServer(t.Context(), nil, nil, " invalid ", slog.Default())
	if err == nil || server != nil {
		t.Fatalf("NewServer = (%v, %v), want nil server and error", server, err)
	}
}

func TestPresenceTransitionsHoldTheHubLock(t *testing.T) {
	presence := &blockingPresence{
		connectStarted:     make(chan struct{}),
		continueConnect:    make(chan struct{}),
		disconnectStarted:  make(chan struct{}),
		continueDisconnect: make(chan struct{}),
	}
	hub := &Hub{
		presence: presence,
		conns:    make(map[int64]*connection),
	}
	conn := &connection{}

	registered := make(chan bool, 1)
	go func() {
		registered <- hub.register(1, conn, mdp.DistributionPointWorker{})
	}()
	waitForSignal(t, presence.connectStarted, "presence connect")
	if hub.mu.TryLock() {
		hub.mu.Unlock()
		t.Fatal("hub lock released before presence connect completed")
	}
	close(presence.continueConnect)
	if !<-registered {
		t.Fatal("register rejected an open hub")
	}

	unregistered := make(chan struct{})
	go func() {
		hub.unregister(1, conn)
		close(unregistered)
	}()
	waitForSignal(t, presence.disconnectStarted, "presence disconnect")
	if hub.mu.TryLock() {
		hub.mu.Unlock()
		t.Fatal("hub lock released before presence disconnect completed")
	}
	close(presence.continueDisconnect)
	waitForSignal(t, unregistered, "unregister")
}

func TestDisconnectClearsRejectedWorkerWithoutConnection(t *testing.T) {
	presence := mdp.NewPresence()
	presence.Reject(1, mdp.DistributionPointWorker{})
	hub := &Hub{
		presence: presence,
		conns:    make(map[int64]*connection),
	}

	hub.Disconnect(1)

	if worker, ok := presence.Worker(1); ok {
		t.Fatalf("Worker = %+v after disconnect, want no worker state", worker)
	}
}

type blockingPresence struct {
	connectStarted     chan struct{}
	continueConnect    chan struct{}
	disconnectStarted  chan struct{}
	continueDisconnect chan struct{}
}

func (p *blockingPresence) Connect(int64, mdp.DistributionPointWorker) {
	close(p.connectStarted)
	<-p.continueConnect
}

func (p *blockingPresence) Disconnect(int64) {
	close(p.disconnectStarted)
	<-p.continueDisconnect
}

func (p *blockingPresence) Clear(int64) {}

func waitForSignal(t *testing.T, signal <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}
