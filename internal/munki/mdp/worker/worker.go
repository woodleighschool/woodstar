package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/woodleighschool/woodstar/internal/munki/mdp/wire"
)

const (
	shutdownTimeout           = 15 * time.Second
	websocketHandshakeTimeout = 30 * time.Second

	initialReconnectDelay = 1 * time.Second
	maxReconnectDelay     = 30 * time.Second
	// connectionStableAfter is how long a connection must last before its
	// successor starts again from the initial reconnect delay.
	connectionStableAfter = time.Minute

	// maxMessageBytes bounds an inbound hello/desired_set message so a large
	// fleet's package list is not truncated by the default read limit.
	maxMessageBytes = 8 << 20

	// initialJobRetry is the first backoff after a failed mirror job. The job
	// keeps retrying with exponential backoff so a transient failure does not
	// strand a package, without hammering the server on a persistent one.
	initialJobRetry = 5 * time.Second
)

var errProtocolMismatch = errors.New("MDP protocol mismatch")

// Worker is the woodstar mdp face: a WebSocket control channel that mirrors
// installers on demand and a serve node that hands them to redirected Munki
// clients.
type Worker struct {
	cfg     Config
	version string
	logger  *slog.Logger
	mirror  *mirror
	client  *woodstarClient
	server  *server

	controlConnected atomic.Bool
}

// New restores the worker's mirror and wires its collaborators.
func New(cfg Config, version string, logger *slog.Logger) (*Worker, error) {
	if !wire.ValidBuildVersion(version) {
		return nil, errors.New("invalid Woodstar build version")
	}
	m, err := loadMirror(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	client, err := newWoodstarClient(cfg.ServerURL, cfg.Key, cfg.ServerCAFile)
	if err != nil {
		return nil, fmt.Errorf("configure Woodstar client: %w", err)
	}
	return &Worker{
		cfg:     cfg,
		version: version,
		logger:  logger,
		mirror:  m,
		client:  client,
		server:  &server{mirror: m, key: []byte(cfg.Key), logger: logger},
	}, nil
}

// Run starts the serve node and the Woodstar connection loop. It returns when
// ctx is cancelled, the serve node exits, or the control connection encounters
// a terminal protocol error. Failure in either half stops the other so the
// worker never lingers partially available.
func (w *Worker) Run(ctx context.Context) error {
	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", w.cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", w.cfg.ListenAddr, err)
	}

	httpServer := &http.Server{Handler: w.handler(), ReadHeaderTimeout: 10 * time.Second}
	transport := "http"
	serve := func() error { return httpServer.Serve(listener) }
	if w.cfg.TLSConfigured() {
		transport = "https"
		serve = func() error {
			return httpServer.ServeTLS(listener, w.cfg.TLSCertFile, w.cfg.TLSKeyFile)
		}
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- serve() }()

	w.logger.InfoContext(ctx, "started",
		"listen_addr", w.cfg.ListenAddr,
		"server_url", w.cfg.ServerURL,
		"data_dir", w.cfg.DataDir,
		"transport", transport,
	)

	connCtx, cancelConn := context.WithCancel(ctx)
	defer cancelConn()
	loopDone := make(chan error, 1)
	go func() {
		loopDone <- w.connectLoop(connCtx)
	}()

	var (
		runErr       error
		loopFinished bool
	)
	select {
	case <-ctx.Done():
	case err := <-serveErr:
		runErr = fmt.Errorf("serve node: %w", err)
	case err := <-loopDone:
		loopFinished = true
		if err != nil {
			runErr = fmt.Errorf("control connection: %w", err)
		}
	}

	w.logger.InfoContext(ctx, "shutting down")
	cancelConn()
	if !loopFinished {
		<-loopDone
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil && runErr == nil {
		runErr = fmt.Errorf("shutdown serve node: %w", err)
	}
	if runErr == nil {
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			runErr = fmt.Errorf("serve node: %w", err)
		}
	}
	return runErr
}

// connectLoop keeps a Woodstar connection up, reconnecting with capped backoff.
// Each reconnect receives the whole desired set, so a missed change is recovered.
func (w *Worker) connectLoop(ctx context.Context) error {
	backoff := initialReconnectDelay
	for {
		if ctx.Err() != nil {
			return nil //nolint:nilerr // Parent cancellation is a clean worker shutdown.
		}
		start := time.Now()
		err := w.connectOnce(ctx)
		if ctx.Err() != nil {
			return nil //nolint:nilerr // Parent cancellation supersedes the connection error.
		}
		if errors.Is(err, errProtocolMismatch) {
			return err
		}
		if time.Since(start) >= connectionStableAfter {
			backoff = initialReconnectDelay
		}
		w.logger.WarnContext(ctx, "connection lost", "retry_in", backoff.String(), "err", err)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, maxReconnectDelay)
	}
}

// connectOnce dials Woodstar and processes hello and desired_set messages until
// the connection ends. The read loop only updates state and kicks the session's
// reconciler; downloads and reporting run on the session's own goroutines, so a
// large download never blocks the control channel or its ping responses.
func (w *Worker) connectOnce(ctx context.Context) error {
	w.logger.DebugContext(ctx, "connecting", "url", w.connectURL())
	dialCtx, cancelDial := context.WithTimeout(ctx, websocketHandshakeTimeout)
	ws, response, err := websocket.Dial( //nolint:bodyclose // websocket.Dial always closes the handshake response body.
		dialCtx, w.connectURL(), &websocket.DialOptions{
			HTTPClient: w.client.websocketHTTP,
			HTTPHeader: http.Header{
				"Authorization":         {"Bearer " + w.cfg.Key},
				wire.BuildVersionHeader: {w.version},
			},
			Subprotocols: []string{wire.Subprotocol},
		})
	cancelDial()
	if err != nil {
		if isProtocolMismatchResponse(response) {
			return fmt.Errorf(
				"%w: worker protocol %q, server protocol %q, server version %q",
				errProtocolMismatch,
				wire.Subprotocol,
				response.Header.Get(wire.ProtocolHeader),
				response.Header.Get(wire.BuildVersionHeader),
			)
		}
		return err
	}
	defer func() { _ = ws.Close(websocket.StatusNormalClosure, "") }()
	defer w.controlConnected.Store(false)
	serverVersion := response.Header.Get(wire.BuildVersionHeader)
	if ws.Subprotocol() != wire.Subprotocol || !wire.ValidBuildVersion(serverVersion) {
		return fmt.Errorf(
			"%w: selected protocol %q, server version %q",
			errProtocolMismatch,
			ws.Subprotocol(),
			serverVersion,
		)
	}
	ws.SetReadLimit(maxMessageBytes)
	w.logger.InfoContext(ctx, "connected",
		"server_url", w.cfg.ServerURL,
		"protocol_version", wire.ProtocolVersion,
		"server_version", serverVersion,
	)

	connCtx, cancel := context.WithCancel(ctx)
	session := newSession(w.mirror, w.client, w.logger, w.cfg.DownloadConcurrency, initialJobRetry)
	defer func() {
		cancel()
		session.wait()
	}()
	go session.reconcileLoop(connCtx)
	go session.writeEvents(connCtx, ws)

	for {
		_, data, err := ws.Read(connCtx)
		if err != nil {
			return err
		}
		var msg wire.ServerMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return fmt.Errorf("decode server message: %w", err)
		}
		switch msg.Type {
		case wire.MessageHello:
			w.logger.DebugContext(connCtx, "received identity",
				"id", msg.DistributionPoint.ID, "name", msg.DistributionPoint.Name)
			w.mirror.setIdentity(msg.DistributionPoint)
			w.controlConnected.Store(true)
		case wire.MessageDesiredSet:
			session.submitDesired(msg.Packages)
		default:
			return fmt.Errorf("unexpected message type %q", msg.Type)
		}
	}
}

func isProtocolMismatchResponse(response *http.Response) bool {
	if response == nil || response.StatusCode != http.StatusUpgradeRequired {
		return false
	}
	_, protocolOK := wire.ParseSubprotocolVersion(response.Header.Get(wire.ProtocolHeader))
	return protocolOK && wire.ValidBuildVersion(response.Header.Get(wire.BuildVersionHeader))
}

func (w *Worker) connectURL() string {
	return w.cfg.ServerURL + "/api/munki/distribution/connect"
}
