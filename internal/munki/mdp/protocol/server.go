// Package protocol serves the Munki distribution point worker protocol.
package protocol

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/httpx"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/munki/mdp/wire"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// workerDownloadTTL bounds a mirror download URL. The worker fetches one as a
// job starts, so it only needs to cover establishing the transfer, not the whole
// stream: an in-flight download keeps going after the URL expires.
const workerDownloadTTL = 15 * time.Minute

// Server owns the worker-facing MDP protocol routes and live connection hub.
type Server struct {
	store    *mdp.Store
	hub      *Hub
	delivery objectDelivery
	version  string
	logger   *slog.Logger
}

// workerHandler is the MDP-facing server side: the WebSocket control endpoint a
// worker connects to, plus the per-job download-URL endpoint it pulls from.
type workerHandler struct {
	store    *mdp.Store
	hub      *Hub
	delivery objectDelivery
	version  string
	logger   *slog.Logger
}

type objectDelivery interface {
	DownloadURL(
		ctx context.Context,
		object storage.Object,
		ttl time.Duration,
		opts storage.DeliveryOptions,
	) (string, error)
}

type downloadURLResponse struct {
	DownloadURL string `json:"download_url"`
}

// NewServer returns a worker-facing protocol server.
func NewServer(
	ctx context.Context,
	store *mdp.Store,
	delivery objectDelivery,
	version string,
	logger *slog.Logger,
) (*Server, error) {
	if !wire.ValidBuildVersion(version) {
		return nil, fmt.Errorf("invalid Woodstar build version %q", version)
	}
	return &Server{
		store:    store,
		hub:      newHub(ctx, store, store.Presence(), logger),
		delivery: delivery,
		version:  version,
		logger:   logger,
	}, nil
}

// RegisterRoutes mounts the download endpoint on ordinary and the worker
// WebSocket endpoint on websocket.
func (s *Server) RegisterRoutes(ordinary chi.Router, websocket chi.Router) {
	h := workerHandler{
		store: s.store, hub: s.hub, delivery: s.delivery, version: s.version, logger: s.logger,
	}
	websocket.Get("/api/munki/distribution/connect", h.connect)
	ordinary.Get("/api/munki/distribution/packages/{id}/download-url", h.downloadURL)
}

// RefreshDesiredPackages pushes the current desired package set to connected
// distribution points.
func (s *Server) RefreshDesiredPackages() {
	s.hub.refreshDesiredPackages()
}

// Disconnect drops the current worker connection for a distribution point.
func (s *Server) Disconnect(id int64) {
	s.hub.Disconnect(id)
}

// Close drops connected workers and stops protocol background work.
func (s *Server) Close() {
	s.hub.Close()
}

// connect upgrades an authenticated worker to a WebSocket and hands it to the
// hub for the lifetime of the connection.
func (h workerHandler) connect(w http.ResponseWriter, r *http.Request) {
	dp, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	worker, ok := h.negotiate(w, r, dp.ID)
	if !ok {
		return
	}
	w.Header().Set(wire.BuildVersionHeader, h.version)
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{wire.Subprotocol},
	})
	if err != nil {
		h.log(r, "connect", err)
		return
	}
	if err := h.hub.Serve(r.Context(), ws, dp, worker); err != nil && !isExpectedClose(err) {
		h.log(r, "connect", err)
		_ = ws.Close(websocket.StatusInternalError, "serve error")
		return
	}
	_ = ws.Close(websocket.StatusNormalClosure, "")
}

func (h workerHandler) negotiate(
	w http.ResponseWriter,
	r *http.Request,
	pointID int64,
) (mdp.DistributionPointWorker, bool) {
	w.Header().Set(wire.BuildVersionHeader, h.version)

	protocols := offeredSubprotocols(r.Header)
	versionHeaders := r.Header.Values(wire.BuildVersionHeader)
	if len(protocols) != 1 || protocols[0] != wire.Subprotocol ||
		len(versionHeaders) != 1 || !wire.ValidBuildVersion(versionHeaders[0]) {
		w.Header().Set(wire.ProtocolHeader, wire.Subprotocol)
		h.store.Presence().Reject(pointID, incompatibleWorker(protocols, versionHeaders))
		http.Error(w, "incompatible MDP protocol", http.StatusUpgradeRequired)
		return mdp.DistributionPointWorker{}, false
	}
	protocolVersion := wire.ProtocolVersion
	return mdp.DistributionPointWorker{
		Compatible:      true,
		ProtocolVersion: &protocolVersion,
		BuildVersion:    versionHeaders[0],
	}, true
}

func incompatibleWorker(
	protocols []string,
	versionHeaders []string,
) mdp.DistributionPointWorker {
	worker := mdp.DistributionPointWorker{}
	if len(versionHeaders) == 1 && wire.ValidBuildVersion(versionHeaders[0]) {
		worker.BuildVersion = versionHeaders[0]
	}
	if len(protocols) == 1 {
		if version, ok := wire.ParseSubprotocolVersion(protocols[0]); ok {
			worker.ProtocolVersion = &version
		}
	}
	return worker
}

func offeredSubprotocols(header http.Header) []string {
	var protocols []string
	for _, value := range header.Values("Sec-WebSocket-Protocol") {
		for protocol := range strings.SplitSeq(value, ",") {
			protocols = append(protocols, strings.TrimSpace(protocol))
		}
	}
	return protocols
}

// downloadURL mints a fresh, short-lived URL for an authenticated worker to pull
// one package's installer bytes. The worker calls it as each mirror job starts.
func (h workerHandler) downloadURL(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticate(w, r); !ok {
		return
	}
	packageID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	object, err := h.store.InstallerObject(r.Context(), packageID)
	if errors.Is(err, dbutil.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		h.log(r, "download-url", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	url, err := h.delivery.DownloadURL(
		r.Context(),
		object,
		workerDownloadTTL,
		storage.DeliveryOptions{},
	)
	if err != nil {
		h.log(r, "download-url", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	httpx.Write(w, http.StatusOK, downloadURLResponse{DownloadURL: url})
}

func (h workerHandler) authenticate(w http.ResponseWriter, r *http.Request) (*mdp.DistributionPoint, bool) {
	token, ok := httpx.BearerToken(r.Header.Get("Authorization"))
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return nil, false
	}
	point, err := h.store.AuthenticateWorker(r.Context(), token)
	if errors.Is(err, dbutil.ErrNotFound) {
		w.WriteHeader(http.StatusUnauthorized)
		return nil, false
	}
	if err != nil {
		h.log(r, "authenticate", err)
		w.WriteHeader(http.StatusInternalServerError)
		return nil, false
	}
	return point, true
}

func (h workerHandler) log(r *http.Request, operation string, err error) {
	h.logger.WarnContext(r.Context(), "munki distribution protocol request failed",
		"operation", operation,
		"path", r.URL.Path,
		"err", err,
	)
}

// isExpectedClose reports whether err is a normal end of a connection rather
// than a server-side failure worth logging.
func isExpectedClose(err error) bool {
	if err == nil || errors.Is(err, errHubClosed) || errors.Is(err, context.Canceled) ||
		errors.Is(err, io.EOF) {
		return true
	}
	status := websocket.CloseStatus(err)
	return status == websocket.StatusNormalClosure ||
		status == websocket.StatusGoingAway ||
		status == websocket.StatusPolicyViolation
}
