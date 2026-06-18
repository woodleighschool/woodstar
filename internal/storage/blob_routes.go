package storage

import (
	"errors"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/storage/capability"
)

// blobClaims is storage's capability payload: the operation, the object key it
// authorizes, the codec-enforced expiry, and an optional content type to serve.
type blobClaims struct {
	Op          string `json:"op"`
	Key         string `json:"key"`
	Exp         int64  `json:"exp"`
	ContentType string `json:"content_type,omitempty"`
}

// RegisterBlobRoutes mounts capability-authenticated raw blob transfer routes.
func RegisterBlobRoutes(r chi.Router, store Store, key []byte) {
	h := blobHandler{store: store, key: key}
	r.Get("/storage/blob", h.get)
	r.Put("/storage/blob", h.put)
	r.Options("/storage/blob", h.options)
}

type blobHandler struct {
	store Store
	key   []byte
}

func (h blobHandler) get(w http.ResponseWriter, r *http.Request) {
	writeBlobCORS(w)
	claims, ok := h.verify(w, r, capability.OpGet)
	if !ok {
		return
	}
	reader, info, err := h.store.Open(r.Context(), claims.Key)
	if errors.Is(err, ErrObjectNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	seeker, ok := reader.(io.ReadSeeker)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	contentType := claims.ContentType
	if contentType == "" {
		contentType = info.ContentType
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(w, r, path.Base(claims.Key), time.Time{}, seeker)
}

func (h blobHandler) put(w http.ResponseWriter, r *http.Request) {
	writeBlobCORS(w)
	claims, ok := h.verify(w, r, capability.OpPut)
	if !ok {
		return
	}
	if err := h.store.Put(r.Context(), claims.Key, r.Body, PutOptions{
		ContentType: claims.ContentType,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h blobHandler) options(w http.ResponseWriter, _ *http.Request) {
	writeBlobCORS(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h blobHandler) verify(
	w http.ResponseWriter,
	r *http.Request,
	op string,
) (blobClaims, bool) {
	claims, err := capability.Verify[blobClaims](h.key, r.URL.Query().Get("cap"), op, time.Now())
	switch {
	case errors.Is(err, capability.ErrExpired):
		w.WriteHeader(http.StatusGone)
		return blobClaims{}, false
	case err != nil || claims.Key == "":
		w.WriteHeader(http.StatusUnauthorized)
		return blobClaims{}, false
	}
	return claims, true
}

func writeBlobCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Range")
	w.Header().Set("Access-Control-Expose-Headers", "Accept-Ranges, Content-Length, Content-Range, Content-Type")
}
