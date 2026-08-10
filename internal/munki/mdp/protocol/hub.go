package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/munki/mdp/wire"
	"github.com/woodleighschool/woodstar/internal/randtoken"
)

const (
	pingInterval        = 5 * time.Second
	pingTimeout         = 10 * time.Second
	desiredPollInterval = 20 * time.Second
	connectionIDBytes   = 16

	// sendBuffer absorbs a burst of desired-set pushes without dropping a
	// momentarily-busy worker. A worker whose buffer still overflows is genuinely
	// stuck and is closed so it reconnects clean.
	sendBuffer = 16
)

// errHubClosed is returned by Serve when the hub is shutting down.
var errHubClosed = errors.New("hub closed")

// Hub owns this process's WebSocket connections and ordered desired-set fan-out.
// PostgreSQL owns worker session authority across processes.
type Hub struct {
	store  *mdp.Store
	logger *slog.Logger

	mu     sync.Mutex
	conns  map[int64]*connection
	closed bool

	wake   chan struct{}
	done   chan struct{}
	cancel context.CancelFunc
}

type connection struct {
	id      string
	ws      *websocket.Conn
	send    chan []byte
	desired []byte
}

// newHub returns a connection hub backed by store. One fan-out goroutine keeps
// local workers converged with the authoritative desired package set.
func newHub(ctx context.Context, store *mdp.Store, logger *slog.Logger) *Hub {
	ctx, cancel := context.WithCancel(ctx)
	h := &Hub{
		store:  store,
		logger: logger,
		conns:  map[int64]*connection{},
		wake:   make(chan struct{}, 1),
		done:   make(chan struct{}),
		cancel: cancel,
	}
	go h.fanoutLoop(ctx)
	return h
}

// Close drops every live connection, stops the fan-out loop, and refuses new
// connections, unblocking the serve loops so the HTTP server can shut down.
func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		<-h.done
		return
	}
	h.closed = true
	conns := make([]*connection, 0, len(h.conns))
	for _, c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	h.cancel()
	for _, c := range conns {
		_ = c.ws.Close(websocket.StatusGoingAway, "server shutting down")
	}
	<-h.done
}

// Serve runs one distribution point connection: it sends hello and the desired
// set, relays later desired-set changes outbound, and records reported package
// state inbound, until the connection closes.
func (h *Hub) Serve(
	parent context.Context,
	ws *websocket.Conn,
	dp *mdp.DistributionPoint,
	key string,
	worker mdp.DistributionPointWorker,
) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	connectionID, err := randtoken.Generate(connectionIDBytes)
	if err != nil {
		return err
	}
	if err := h.store.ClaimWorkerSession(ctx, dp.ID, key, connectionID, worker); err != nil {
		_ = ws.Close(websocket.StatusPolicyViolation, "distribution point session rejected")
		return err
	}

	conn := &connection{id: connectionID, ws: ws, send: make(chan []byte, sendBuffer)}
	if !h.register(dp.ID, conn) {
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), pingTimeout)
		_ = h.store.ReleaseWorkerSession(releaseCtx, dp.ID, connectionID)
		releaseCancel()
		return errHubClosed
	}
	defer h.unregister(dp.ID, conn)

	if err := h.sendHello(ctx, ws, dp); err != nil {
		return err
	}

	go h.writeLoop(ctx, cancel, dp.ID, conn)

	if msg, err := h.desiredSetBytes(ctx); err != nil {
		h.logger.WarnContext(ctx, "munki distribution desired set failed",
			"operation", "desired_set", "distribution_point_id", dp.ID, "err", err)
	} else {
		h.enqueueDesired(conn, msg)
	}

	return h.readLoop(ctx, ws, dp.ID, connectionID)
}

func (h *Hub) sendHello(ctx context.Context, ws *websocket.Conn, dp *mdp.DistributionPoint) error {
	return writeJSON(ctx, ws, wire.ServerMessage{
		Type:              wire.MessageHello,
		DistributionPoint: wire.PointIdentity{ID: dp.ID, Name: dp.Name},
	})
}

func (h *Hub) readLoop(ctx context.Context, ws *websocket.Conn, dpID int64, connectionID string) error {
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			return err
		}
		var event wire.PackageEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("decode package event: %w", err)
		}
		status, ok := statusForEvent(event.Type)
		if !ok {
			return fmt.Errorf("unexpected message type %q", event.Type)
		}
		if err := h.store.RecordPackageState(
			ctx, dpID, connectionID, event.PackageID, status, event.SHA256, event.Error,
		); err != nil {
			if errors.Is(err, mdp.ErrWorkerSessionInvalid) {
				_ = ws.Close(websocket.StatusPolicyViolation, "distribution point session replaced")
				return err
			}
			// A record failure (e.g. the package was just deleted) is the worker's
			// problem to retry, not a reason to drop an otherwise healthy connection.
			h.logger.WarnContext(ctx, "munki distribution record state failed",
				"operation", "state", "distribution_point_id", dpID,
				"package_id", event.PackageID, "err", err)
		}
	}
}

func (h *Hub) writeLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	dpID int64,
	conn *connection,
) {
	defer cancel()
	ping := time.NewTicker(pingInterval)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-conn.send:
			if err := conn.ws.Write(ctx, websocket.MessageText, msg); err != nil {
				return
			}
		case <-ping.C:
			pingCtx, pingCancel := context.WithTimeout(ctx, pingTimeout)
			err := h.store.RenewWorkerSession(pingCtx, dpID, conn.id)
			if err == nil {
				err = conn.ws.Ping(pingCtx)
			}
			pingCancel()
			if err != nil {
				if errors.Is(err, mdp.ErrWorkerSessionInvalid) {
					_ = conn.ws.Close(websocket.StatusPolicyViolation, "distribution point session replaced")
				} else if ctx.Err() == nil {
					h.logger.WarnContext(ctx, "munki distribution session renewal failed",
						"operation", "renew", "distribution_point_id", dpID, "err", err)
				}
				return
			}
		}
	}
}

// refreshDesiredPackages wakes the fan-out loop to re-push the desired set. It
// is fire-and-forget and coalescing: a burst of mutations collapses into one
// broadcast of the final state.
func (h *Hub) refreshDesiredPackages() {
	select {
	case h.wake <- struct{}{}:
	default:
	}
}

func (h *Hub) fanoutLoop(ctx context.Context) {
	defer close(h.done)
	poll := time.NewTicker(desiredPollInterval)
	defer poll.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.wake:
			h.broadcastDesired(ctx)
		case <-poll.C:
			if h.hasConnections() {
				h.broadcastDesired(ctx)
			}
		}
	}
}

func (h *Hub) hasConnections() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.conns) > 0
}

func (h *Hub) broadcastDesired(ctx context.Context) {
	msg, err := h.desiredSetBytes(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		h.logger.WarnContext(ctx, "munki distribution desired broadcast failed",
			"operation", "desired_set", "err", err)
		return
	}
	h.mu.Lock()
	conns := make([]*connection, 0, len(h.conns))
	for _, c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	for _, c := range conns {
		h.enqueueDesired(c, msg)
	}
}

func (h *Hub) desiredSetBytes(ctx context.Context) ([]byte, error) {
	desired, err := h.store.DesiredPackages(ctx)
	if err != nil {
		return nil, err
	}
	packages := make([]wire.DesiredPackage, len(desired))
	for i, d := range desired {
		packages[i] = wire.DesiredPackage(d)
	}
	return json.Marshal(wire.ServerMessage{Type: wire.MessageDesiredSet, Packages: packages})
}

// enqueue hands a message to a connection's writer, closing a connection whose
// buffer has overflowed since that worker is no longer keeping up.
func (h *Hub) enqueue(c *connection, msg []byte) {
	select {
	case c.send <- msg:
	default:
		_ = c.ws.Close(websocket.StatusPolicyViolation, "distribution point fell behind")
	}
}

func (h *Hub) enqueueDesired(c *connection, msg []byte) {
	h.mu.Lock()
	if bytes.Equal(c.desired, msg) {
		h.mu.Unlock()
		return
	}
	c.desired = bytes.Clone(msg)
	h.mu.Unlock()
	h.enqueue(c, msg)
}

func (h *Hub) register(id int64, conn *connection) bool {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return false
	}
	old := h.conns[id]
	h.conns[id] = conn
	h.mu.Unlock()
	if old != nil {
		_ = old.ws.Close(websocket.StatusPolicyViolation, "replaced by a new connection")
	}
	return true
}

func (h *Hub) unregister(id int64, conn *connection) {
	h.mu.Lock()
	if h.conns[id] == conn {
		delete(h.conns, id)
	}
	h.mu.Unlock()
	releaseCtx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := h.store.ReleaseWorkerSession(releaseCtx, id, conn.id); err != nil {
		h.logger.WarnContext(releaseCtx, "munki distribution session release failed",
			"operation", "release", "distribution_point_id", id, "err", err)
	}
}

// Disconnect drops the current connection for one distribution point. Key
// rotation, disabling, and deletion must invalidate the live worker as well as
// subsequent HTTP authentication.
func (h *Hub) Disconnect(id int64) {
	h.mu.Lock()
	conn := h.conns[id]
	if conn == nil {
		h.mu.Unlock()
		return
	}
	delete(h.conns, id)
	h.mu.Unlock()
	_ = conn.ws.Close(websocket.StatusPolicyViolation, "distribution point credentials changed")
}

func statusForEvent(eventType string) (mdp.PackageStatus, bool) {
	switch eventType {
	case wire.EventPackageSyncing:
		return mdp.PackageStatusSyncing, true
	case wire.EventPackageCurrent:
		return mdp.PackageStatusCurrent, true
	case wire.EventPackageError:
		return mdp.PackageStatusError, true
	default:
		return "", false
	}
}

func writeJSON(ctx context.Context, ws *websocket.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return ws.Write(ctx, websocket.MessageText, data)
}
