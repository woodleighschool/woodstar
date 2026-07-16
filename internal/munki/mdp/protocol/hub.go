package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/woodleighschool/woodstar/internal/munki/mdp"
)

const (
	// messageHello is sent once when a distribution point connects: its identity.
	// The desired set follows in its own message so a large list never blocks the
	// hello handshake.
	messageHello = "hello"
	// messageDesiredSet is the full authoritative installer list. It is sent on
	// connect and re-sent whenever the desired set changes, so a worker always
	// reconciles against current truth and ordering of deltas never matters.
	messageDesiredSet = "desired_set"

	// Worker-to-server package events. Only a current event with a matching hash
	// makes a point current; syncing and error are advisory for the admin view.
	eventPackageSyncing = "package_syncing"
	eventPackageCurrent = "package_current"
	eventPackageError   = "package_error"

	pingInterval = 20 * time.Second
	pingTimeout  = 10 * time.Second

	// sendBuffer absorbs a burst of desired-set pushes without dropping a
	// momentarily-busy worker. A worker whose buffer still overflows is genuinely
	// stuck and is closed so it reconnects clean.
	sendBuffer = 16
)

// errHubClosed is returned by Serve when the hub is shutting down.
var errHubClosed = errors.New("hub closed")

type pointIdentity struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type helloMessage struct {
	Type              string        `json:"type"`
	DistributionPoint pointIdentity `json:"distribution_point"`
}

type desiredSetMessage struct {
	Type     string                  `json:"type"`
	Packages []desiredPackageMessage `json:"packages"`
}

// desiredPackageMessage is one installer the worker should mirror. It carries no
// download URL: the worker requests a fresh one per job as it starts.
type desiredPackageMessage struct {
	PackageID int64  `json:"package_id"`
	Filename  string `json:"filename"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

// packageEvent is one package's mirror state, reported by the worker as each job
// settles. It is the inbound half of the protocol.
type packageEvent struct {
	Type      string `json:"type"`
	PackageID int64  `json:"package_id"`
	SHA256    string `json:"sha256"`
	Error     string `json:"error"`
}

// Hub tracks live distribution point connections. It is the writer of presence
// and the ordered fan-out for desired-set changes.
type Hub struct {
	store    *mdp.Store
	presence presenceWriter
	logger   *slog.Logger

	mu     sync.Mutex
	conns  map[int64]*connection
	closed bool

	wake   chan struct{}
	done   chan struct{}
	cancel context.CancelFunc
}

type presenceWriter interface {
	Connect(pointID int64)
	Disconnect(pointID int64)
}

type connection struct {
	ws   *websocket.Conn
	send chan []byte
}

// newHub returns a connection hub backed by store, writing presence as workers
// connect and disconnect. It runs one fan-out goroutine so desired-set pushes
// reach every worker in a single, ordered sequence.
func newHub(ctx context.Context, store *mdp.Store, presence *mdp.Presence, logger *slog.Logger) *Hub {
	ctx, cancel := context.WithCancel(ctx)
	h := &Hub{
		store:    store,
		presence: presence,
		logger:   logger,
		conns:    map[int64]*connection{},
		wake:     make(chan struct{}, 1),
		done:     make(chan struct{}),
		cancel:   cancel,
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
func (h *Hub) Serve(parent context.Context, ws *websocket.Conn, dp *mdp.DistributionPoint) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	if err := h.sendHello(ctx, ws, dp); err != nil {
		return err
	}

	conn := &connection{ws: ws, send: make(chan []byte, sendBuffer)}
	if !h.register(dp.ID, conn) {
		return errHubClosed
	}
	defer h.unregister(dp.ID, conn)

	go h.writeLoop(ctx, cancel, conn)

	if msg, err := h.desiredSetBytes(ctx); err != nil {
		h.logger.WarnContext(ctx, "munki distribution desired set failed",
			"operation", "desired_set", "distribution_point_id", dp.ID, "err", err)
	} else {
		h.enqueue(conn, msg)
	}

	return h.readLoop(ctx, ws, dp.ID)
}

func (h *Hub) sendHello(ctx context.Context, ws *websocket.Conn, dp *mdp.DistributionPoint) error {
	return writeJSON(ctx, ws, helloMessage{
		Type:              messageHello,
		DistributionPoint: pointIdentity{ID: dp.ID, Name: dp.Name},
	})
}

func (h *Hub) readLoop(ctx context.Context, ws *websocket.Conn, dpID int64) error {
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			return err
		}
		var event packageEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("decode package event: %w", err)
		}
		status, ok := statusForEvent(event.Type)
		if !ok {
			return fmt.Errorf("unexpected message type %q", event.Type)
		}
		if err := h.store.RecordPackageState(
			ctx, dpID, event.PackageID, status, event.SHA256, event.Error,
		); err != nil {
			// A record failure (e.g. the package was just deleted) is the worker's
			// problem to retry, not a reason to drop an otherwise healthy connection.
			h.logger.WarnContext(ctx, "munki distribution record state failed",
				"operation", "state", "distribution_point_id", dpID,
				"package_id", event.PackageID, "err", err)
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
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.wake:
			h.broadcastDesired(ctx)
		}
	}
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
		h.enqueue(c, msg)
	}
}

func (h *Hub) desiredSetBytes(ctx context.Context) ([]byte, error) {
	desired, err := h.store.DesiredPackages(ctx)
	if err != nil {
		return nil, err
	}
	packages := make([]desiredPackageMessage, len(desired))
	for i, d := range desired {
		packages[i] = desiredPackageMessage(d)
	}
	return json.Marshal(desiredSetMessage{Type: messageDesiredSet, Packages: packages})
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

func (h *Hub) register(id int64, conn *connection) bool {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return false
	}
	old := h.conns[id]
	h.conns[id] = conn
	h.presence.Connect(id)
	h.mu.Unlock()
	if old != nil {
		_ = old.ws.Close(websocket.StatusPolicyViolation, "replaced by a new connection")
	}
	return true
}

func (h *Hub) unregister(id int64, conn *connection) {
	h.mu.Lock()
	removed := h.conns[id] == conn
	if removed {
		delete(h.conns, id)
		h.presence.Disconnect(id)
	}
	h.mu.Unlock()
	// Only a replaced or genuinely-closed current connection clears presence; a
	// superseded old connection's late unregister must not knock the new one out.
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
	h.presence.Disconnect(id)
	h.mu.Unlock()
	_ = conn.ws.Close(websocket.StatusPolicyViolation, "distribution point credentials changed")
}

func statusForEvent(eventType string) (mdp.PackageStatus, bool) {
	switch eventType {
	case eventPackageSyncing:
		return mdp.PackageStatusSyncing, true
	case eventPackageCurrent:
		return mdp.PackageStatusCurrent, true
	case eventPackageError:
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
