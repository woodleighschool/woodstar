// Package protocol serves the Munki distribution point worker protocol.
package protocol

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/httpx"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// workerDownloadTTL bounds a mirror download URL. The worker fetches one as a
// job starts, so it only needs to cover establishing the transfer, not the whole
// stream: an in-flight download keeps going after the URL expires.
const workerDownloadTTL = 15 * time.Minute

// Server owns the worker-facing MDP protocol routes and live connection hub.
type Server struct {
	store     *mdp.Store
	hub       *Hub
	presigner storage.Presigner
	logger    *slog.Logger
}

// workerHandler is the MDP-facing server side: the WebSocket control endpoint a
// worker connects to, plus the per-job download-URL endpoint it pulls from.
type workerHandler struct {
	store     *mdp.Store
	hub       *Hub
	presigner storage.Presigner
	logger    *slog.Logger
}

type downloadURLResponse struct {
	DownloadURL string `json:"download_url"`
}

// NewServer returns a worker-facing protocol server.
func NewServer(
	ctx context.Context,
	store *mdp.Store,
	presigner storage.Presigner,
	logger *slog.Logger,
) *Server {
	return &Server{
		store:     store,
		hub:       newHub(ctx, store, store.Presence(), logger),
		presigner: presigner,
		logger:    logger,
	}
}

// RegisterRoutes mounts the MDP worker-facing endpoints.
func (s *Server) RegisterRoutes(r chi.Router) {
	h := workerHandler{store: s.store, hub: s.hub, presigner: s.presigner, logger: s.logger}
	r.Get("/api/munki/distribution/connect", h.connect)
	r.Get("/api/munki/distribution/packages/{id}/download-url", h.downloadURL)
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
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.log(r, "connect", err)
		return
	}
	if err := h.hub.Serve(r.Context(), ws, dp); err != nil && !isExpectedClose(err) {
		h.log(r, "connect", err)
		_ = ws.Close(websocket.StatusInternalError, "serve error")
		return
	}
	_ = ws.Close(websocket.StatusNormalClosure, "")
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
	url, err := h.presigner.PresignGet(
		r.Context(),
		object.Key,
		workerDownloadTTL,
		storage.GetOptions{ContentType: object.ContentType},
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
	switch websocket.CloseStatus(err) {
	case websocket.StatusNormalClosure, websocket.StatusGoingAway, websocket.StatusPolicyViolation:
		return true
	default:
		return false
	}
}
