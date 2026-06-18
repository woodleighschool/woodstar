package mdp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/httpauth"
	"github.com/woodleighschool/woodstar/internal/storage"
)

type agent struct {
	store     *Store
	hub       *Hub
	presigner storage.Presigner
	logger    *slog.Logger
}

// RegisterProtocolRoutes mounts the MDP-facing connect and content endpoints.
// Both authenticate the worker by its per-DP bearer key.
func RegisterProtocolRoutes(
	r chi.Router,
	hub *Hub,
	store *Store,
	presigner storage.Presigner,
	logger *slog.Logger,
) {
	a := agent{store: store, hub: hub, presigner: presigner, logger: logger}
	r.Get("/api/munki/distribution/connect", a.connect)
	r.Get("/api/munki/distribution/packages/{package_id}/content", a.content)
}

// connect upgrades an authenticated worker to a WebSocket and hands it to the
// hub for the lifetime of the connection.
func (a agent) connect(w http.ResponseWriter, r *http.Request) {
	dp, ok := a.authenticate(w, r)
	if !ok {
		return
	}
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		a.log(r, "connect", err)
		return
	}
	if err := a.hub.Serve(ws, dp); err != nil && !isExpectedClose(err) {
		a.log(r, "connect", err)
		_ = ws.Close(websocket.StatusInternalError, "serve error")
		return
	}
	_ = ws.Close(websocket.StatusNormalClosure, "")
}

// content streams a mirrored installer's bytes to the worker by redirecting to a
// short-lived storage URL; the worker never holds storage credentials.
func (a agent) content(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.authenticate(w, r); !ok {
		return
	}
	packageID, err := strconv.ParseInt(chi.URLParam(r, "package_id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	key, err := a.store.InstallerObjectKey(r.Context(), packageID)
	if errors.Is(err, dbutil.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		a.log(r, "content", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	url, err := a.presigner.PresignGet(r.Context(), key, 0, storage.GetOptions{})
	if err != nil {
		a.log(r, "content", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

func (a agent) authenticate(w http.ResponseWriter, r *http.Request) (*DistributionPoint, bool) {
	token, ok := httpauth.BearerToken(r.Header.Get("Authorization"))
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return nil, false
	}
	point, err := a.store.AuthenticateWorker(r.Context(), token)
	if errors.Is(err, dbutil.ErrNotFound) {
		w.WriteHeader(http.StatusUnauthorized)
		return nil, false
	}
	if err != nil {
		a.log(r, "authenticate", err)
		w.WriteHeader(http.StatusInternalServerError)
		return nil, false
	}
	return point, true
}

func (a agent) log(r *http.Request, operation string, err error) {
	a.logger.WarnContext(r.Context(), "munki distribution protocol request failed",
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
