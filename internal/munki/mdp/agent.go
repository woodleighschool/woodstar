package mdp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/httpx"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// workerDownloadTTL bounds a mirror download URL. The worker fetches one as a
// job starts, so it only needs to cover establishing the transfer, not the whole
// stream: an in-flight download keeps going after the URL expires.
const workerDownloadTTL = 15 * time.Minute

// workerHandler is the MDP-facing server side: the WebSocket control endpoint a
// worker connects to, plus the per-job download-URL endpoint it pulls from.
type workerHandler struct {
	store     *Store
	hub       *Hub
	presigner storage.Presigner
	logger    *slog.Logger
}

type downloadURLResponse struct {
	DownloadURL string `json:"download_url"`
}

// RegisterProtocolRoutes mounts the MDP worker-facing endpoints.
func RegisterProtocolRoutes(
	r chi.Router,
	hub *Hub,
	store *Store,
	presigner storage.Presigner,
	logger *slog.Logger,
) {
	h := workerHandler{store: store, hub: hub, presigner: presigner, logger: logger}
	r.Get("/api/munki/distribution/connect", h.connect)
	r.Get("/api/munki/distribution/packages/{id}/download-url", h.downloadURL)
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
	if err := h.hub.Serve(ws, dp); err != nil && !isExpectedClose(err) {
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
	key, err := h.store.InstallerObjectKey(r.Context(), packageID)
	if errors.Is(err, dbutil.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		h.log(r, "download-url", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	url, err := h.presigner.PresignGet(r.Context(), key, workerDownloadTTL, storage.GetOptions{})
	if err != nil {
		h.log(r, "download-url", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	httpx.Write(w, http.StatusOK, downloadURLResponse{DownloadURL: url})
}

func (h workerHandler) authenticate(w http.ResponseWriter, r *http.Request) (*DistributionPoint, bool) {
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
	if err == nil || errors.Is(err, errHubClosed) || errors.Is(err, context.Canceled) {
		return true
	}
	switch websocket.CloseStatus(err) {
	case websocket.StatusNormalClosure, websocket.StatusGoingAway:
		return true
	default:
		return false
	}
}
