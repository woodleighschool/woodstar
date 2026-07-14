package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

const (
	shutdownTimeout = 15 * time.Second

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

// Worker is the woodstar mdp face: a WebSocket control channel that mirrors
// installers on demand and a serve node that hands them to redirected Munki
// clients.
type Worker struct {
	cfg    Config
	logger *slog.Logger
	mirror *mirror
	client *woodstarClient
	server *server
}

// New restores the worker's mirror and wires its collaborators.
func New(cfg Config, logger *slog.Logger) (*Worker, error) {
	m, err := loadMirror(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	return &Worker{
		cfg:    cfg,
		logger: logger,
		mirror: m,
		client: newWoodstarClient(cfg.ServerURL, cfg.Key),
		server: &server{mirror: m, key: []byte(cfg.Key), logger: logger},
	}, nil
}

// Run starts the serve node and the Woodstar connection loop. It returns when
// ctx is cancelled or the serve node exits on its own; a serve-node failure
// stops the connection loop so the worker never lingers half-up.
func (w *Worker) Run(ctx context.Context) error {
	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", w.cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", w.cfg.ListenAddr, err)
	}

	httpServer := &http.Server{Handler: w.server.handler(), ReadHeaderTimeout: 10 * time.Second}
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
	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		w.connectLoop(connCtx)
	}()

	var runErr error
	select {
	case <-ctx.Done():
	case err := <-serveErr:
		runErr = fmt.Errorf("serve node: %w", err)
	}

	w.logger.InfoContext(ctx, "shutting down")
	cancelConn()
	<-loopDone

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
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
func (w *Worker) connectLoop(ctx context.Context) {
	backoff := initialReconnectDelay
	for {
		if ctx.Err() != nil {
			return
		}
		start := time.Now()
		err := w.connectOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		if time.Since(start) >= connectionStableAfter {
			backoff = initialReconnectDelay
		}
		w.logger.WarnContext(ctx, "connection lost", "retry_in", backoff.String(), "err", err)
		select {
		case <-ctx.Done():
			return
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
	// coder/websocket owns resp.Body (nil on success, a NopCloser on error), so we
	// neither read nor close it; doing so dereferences nil on the success path.
	//nolint:bodyclose // resp.Body lifecycle is managed by coder/websocket's Dial
	ws, _, err := websocket.Dial(ctx, w.connectURL(), &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + w.cfg.Key}},
	})
	if err != nil {
		return err
	}
	defer ws.Close(websocket.StatusNormalClosure, "")
	ws.SetReadLimit(maxMessageBytes)
	w.logger.InfoContext(ctx, "connected", "server_url", w.cfg.ServerURL)

	connCtx, cancel := context.WithCancel(ctx)
	session := newSession(connCtx, w.mirror, w.client, w.logger, w.cfg.DownloadConcurrency, initialJobRetry)
	defer func() {
		cancel()
		session.wait()
	}()
	go session.reconcileLoop()
	go session.writeEvents(ws)

	for {
		_, data, err := ws.Read(connCtx)
		if err != nil {
			return err
		}
		var msg serverMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return fmt.Errorf("decode server message: %w", err)
		}
		switch msg.Type {
		case messageHello:
			w.logger.DebugContext(connCtx, "received identity",
				"id", msg.DistributionPoint.ID, "name", msg.DistributionPoint.Name)
			w.mirror.setIdentity(msg.DistributionPoint)
		case messageDesiredSet:
			session.submitDesired(msg.Packages)
		default:
			return fmt.Errorf("unexpected message type %q", msg.Type)
		}
	}
}

func (w *Worker) connectURL() string {
	base := w.cfg.ServerURL
	switch {
	case strings.HasPrefix(base, "https://"):
		base = "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		base = "ws://" + strings.TrimPrefix(base, "http://")
	}
	return base + "/api/munki/distribution/connect"
}
