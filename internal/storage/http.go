package storage

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/storage/capability"
)

type transferRouteRegistrar interface {
	registerTransferRoutes(r chi.Router, logger *slog.Logger)
}

// RegisterTransferRoutes mounts Woodstar-hosted transfers for the file backend.
// S3 transfers are served exclusively by provider-signed URLs.
func RegisterTransferRoutes(r chi.Router, backend Backend, logger *slog.Logger) {
	backend.registerTransferRoutes(r, logger)
}

func (s *fileStore) registerTransferRoutes(r chi.Router, logger *slog.Logger) {
	h := transferHandler{
		store:  s,
		key:    s.capabilityKey,
		logger: logger,
	}
	r.Get("/storage/*", h.get)
	r.Put("/storage/*", h.put)
}

func (*s3Store) registerTransferRoutes(chi.Router, *slog.Logger) {}

type transferHandler struct {
	store  Store
	key    []byte
	logger *slog.Logger
}

func (h transferHandler) get(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.verify(w, r, capability.OpGet)
	if !ok {
		return
	}
	if err := serveKey(
		w,
		r,
		h.store,
		claims.Key,
		serveOptions{ContentType: claims.ContentType},
	); err != nil {
		h.logError(r, "get-storage-object", err, "key", claims.Key)
	}
}

func (h transferHandler) put(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.verify(w, r, capability.OpPut)
	if !ok {
		return
	}
	if err := h.store.Put(
		r.Context(),
		claims.Key,
		r.Body,
		PutOptions{},
	); err != nil {
		h.logError(r, "put-storage-object", err, "key", claims.Key)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h transferHandler) logError(r *http.Request, operation string, err error, attrs ...any) {
	args := make([]any, 0, 6+len(attrs))
	args = append(args, "operation", operation, "status", http.StatusInternalServerError)
	args = append(args, attrs...)
	args = append(args, "err", err)
	h.logger.ErrorContext(r.Context(), "storage blob handler failed", args...)
}

func (h transferHandler) verify(
	w http.ResponseWriter,
	r *http.Request,
	op string,
) (BlobCapabilityClaims, bool) {
	claims, err := capability.Verify[BlobCapabilityClaims](h.key, r.URL.Query().Get("cap"), op, time.Now())
	requestKey := chi.URLParam(r, "*")
	switch {
	case errors.Is(err, capability.ErrExpired):
		w.WriteHeader(http.StatusGone)
		return BlobCapabilityClaims{}, false
	case err != nil || claims.Key == "" || requestKey != claims.Key:
		w.WriteHeader(http.StatusUnauthorized)
		return BlobCapabilityClaims{}, false
	}
	return claims, true
}
