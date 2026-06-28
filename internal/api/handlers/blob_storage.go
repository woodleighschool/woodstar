package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/storage/capability"
)

// RegisterBlobStorage mounts capability-authenticated raw blob transfer routes.
func RegisterBlobStorage(r chi.Router, store storage.Store, key []byte, logger *slog.Logger) {
	h := blobStorageHandler{store: store, key: key, logger: logger}
	r.Get("/storage/*", h.get)
	r.Put("/storage/*", h.put)
}

type blobStorageHandler struct {
	store  storage.Store
	key    []byte
	logger *slog.Logger
}

func (h blobStorageHandler) get(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.verify(w, r, capability.OpGet)
	if !ok {
		return
	}
	if err := storage.ServeObject(
		w,
		r,
		h.store,
		claims.Key,
		storage.ServeOptions{ContentType: claims.ContentType},
	); err != nil {
		h.logError(r, "get-storage-object", err, "key", claims.Key)
	}
}

func (h blobStorageHandler) put(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.verify(w, r, capability.OpPut)
	if !ok {
		return
	}
	if err := h.store.Put(r.Context(), claims.Key, r.Body, storage.PutOptions{
		ContentType: claims.ContentType,
	}); err != nil {
		h.logError(r, "put-storage-object", err, "key", claims.Key)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h blobStorageHandler) logError(r *http.Request, operation string, err error, attrs ...any) {
	args := []any{"operation", operation, "status", http.StatusInternalServerError}
	args = append(args, attrs...)
	args = append(args, "err", err)
	h.logger.ErrorContext(r.Context(), "storage blob handler failed", args...)
}

func (h blobStorageHandler) verify(
	w http.ResponseWriter,
	r *http.Request,
	op string,
) (storage.BlobCapabilityClaims, bool) {
	claims, err := capability.Verify[storage.BlobCapabilityClaims](h.key, r.URL.Query().Get("cap"), op, time.Now())
	requestKey := chi.URLParam(r, "*")
	switch {
	case errors.Is(err, capability.ErrExpired):
		w.WriteHeader(http.StatusGone)
		return storage.BlobCapabilityClaims{}, false
	case err != nil || claims.Key == "" || requestKey != claims.Key:
		w.WriteHeader(http.StatusUnauthorized)
		return storage.BlobCapabilityClaims{}, false
	}
	return claims, true
}
