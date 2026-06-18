package mdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	// messageHello is sent once when a distribution point connects: its identity
	// plus the full desired set. Every connect is a fresh reconciliation boundary.
	messageHello = "hello"
	// messageDesiredChanged is pushed whenever the desired installer set changes.
	messageDesiredChanged = "desired_changed"
	// messageState is the distribution point's reported state, inbound only.
	messageState = "state"

	pingInterval = 20 * time.Second
	pingTimeout  = 10 * time.Second
)

// errHubClosed is returned by Serve when the hub is shutting down.
var errHubClosed = errors.New("hub closed")

type pointIdentity struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type helloMessage struct {
	Type              string           `json:"type"`
	DistributionPoint pointIdentity    `json:"distribution_point"`
	Packages          []DesiredPackage `json:"packages"`
}

type desiredChangedMessage struct {
	Type     string           `json:"type"`
	Packages []DesiredPackage `json:"packages"`
}

type stateMessage struct {
	Type     string                `json:"type"`
	Packages []statePackageMessage `json:"packages"`
}

type statePackageMessage struct {
	PackageID int64         `json:"package_id"`
	SHA256    string        `json:"sha256,omitempty"`
	Status    PackageStatus `json:"status,omitempty"`
	Error     string        `json:"error,omitempty"`
}

// Hub tracks live distribution point connections. It is the source of truth for
// presence and the fan-out for desired-set changes.
type Hub struct {
	store  *Store
	logger *slog.Logger

	mu     sync.RWMutex
	conns  map[int64]*connection
	closed bool
}

type connection struct {
	ws   *websocket.Conn
	send chan []byte
}

// NewHub returns a connection hub backed by store.
func NewHub(store *Store, logger *slog.Logger) *Hub {
	return &Hub{store: store, logger: logger, conns: map[int64]*connection{}}
}

// Online reports whether a distribution point holds a live connection.
func (h *Hub) Online(id int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.conns[id]
	return ok
}

// Close drops every live connection and refuses new ones, unblocking the serve
// loops so the HTTP server can shut down.
func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	conns := make([]*connection, 0, len(h.conns))
	for _, c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	for _, c := range conns {
		_ = c.ws.Close(websocket.StatusGoingAway, "server shutting down")
	}
}

// Serve runs one distribution point connection: it sends hello, relays
// desired-set changes outbound, and records reported state inbound, until the
// connection closes. Every call is a full reconciliation boundary for the point.
func (h *Hub) Serve(ws *websocket.Conn, dp *DistributionPoint) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := h.sendHello(ctx, ws, dp); err != nil {
		return err
	}

	conn := &connection{ws: ws, send: make(chan []byte, 1)}
	if !h.register(dp.ID, conn) {
		return errHubClosed
	}
	defer h.unregister(dp.ID, conn)

	go h.writeLoop(ctx, cancel, conn)
	return h.readLoop(ctx, ws, dp.ID)
}

func (h *Hub) sendHello(ctx context.Context, ws *websocket.Conn, dp *DistributionPoint) error {
	desired, err := h.store.DesiredPackages(ctx)
	if err != nil {
		return err
	}
	return writeJSON(ctx, ws, helloMessage{
		Type:              messageHello,
		DistributionPoint: pointIdentity{ID: dp.ID, Name: dp.Name},
		Packages:          desired,
	})
}

func (h *Hub) readLoop(ctx context.Context, ws *websocket.Conn, dpID int64) error {
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			return err
		}
		var msg stateMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return fmt.Errorf("decode state message: %w", err)
		}
		if msg.Type != messageState {
			return fmt.Errorf("unexpected message type %q", msg.Type)
		}
		if err := h.store.RecordState(ctx, dpID, stateReportFromMessage(msg)); err != nil {
			h.logger.WarnContext(ctx, "munki distribution record state failed",
				"operation", "state", "distribution_point_id", dpID, "err", err)
		}
	}
}

func (h *Hub) writeLoop(ctx context.Context, cancel context.CancelFunc, conn *connection) {
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
			err := conn.ws.Ping(pingCtx)
			pingCancel()
			if err != nil {
				return
			}
		}
	}
}

// DesiredChanged recomputes the desired set and pushes it to every connection.
// It is fire-and-forget: a package mutation is not blocked on fan-out.
func (h *Hub) DesiredChanged() {
	go h.broadcastDesired()
}

func (h *Hub) broadcastDesired() {
	ctx := context.Background()
	desired, err := h.store.DesiredPackages(ctx)
	if err != nil {
		h.logger.WarnContext(ctx, "munki distribution desired broadcast failed",
			"operation", "desired_changed", "err", err)
		return
	}
	msg, err := json.Marshal(desiredChangedMessage{
		Type:     messageDesiredChanged,
		Packages: desired,
	})
	if err != nil {
		return
	}
	h.mu.RLock()
	conns := make([]*connection, 0, len(h.conns))
	for _, c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.RUnlock()
	for _, c := range conns {
		select {
		case c.send <- msg:
		default:
			_ = c.ws.Close(websocket.StatusPolicyViolation, "stale desired state")
		}
	}
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
}

func stateReportFromMessage(msg stateMessage) StateReport {
	packages := make([]ReportedPackage, len(msg.Packages))
	for i, pkg := range msg.Packages {
		packages[i] = ReportedPackage(pkg)
	}
	return StateReport{
		Packages: packages,
	}
}

func writeJSON(ctx context.Context, ws *websocket.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return ws.Write(ctx, websocket.MessageText, data)
}
