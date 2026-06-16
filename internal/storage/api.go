package storage

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

func RegisterContentAdminRoutes(r chi.Router, objects *ObjectStore, store Store) {
	r.Put("/api/objects/{id}/content", func(w http.ResponseWriter, req *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(req, "id"), 10, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		obj, err := objects.GetByID(req.Context(), id)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		if obj.Available() {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if err := store.Put(
			req.Context(),
			obj.Key(),
			req.Body,
			PutOptions{ContentType: obj.ContentType},
		); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func ServeContent(ctx huma.Context, store Store, key string) {
	if presigner, ok := store.(Presigner); ok {
		url, err := presigner.PresignGet(ctx.Context(), key, 0)
		if err != nil {
			ctx.SetStatus(http.StatusInternalServerError)
			return
		}
		ctx.SetHeader("Location", url)
		ctx.SetStatus(http.StatusFound)
		return
	}
	reader, info, err := store.Open(ctx.Context(), key)
	if errors.Is(err, ErrObjectNotFound) {
		ctx.SetStatus(http.StatusNotFound)
		return
	}
	if err != nil {
		ctx.SetStatus(http.StatusInternalServerError)
		return
	}
	defer reader.Close()
	if info.ContentType != "" {
		ctx.SetHeader("Content-Type", info.ContentType)
	}
	ctx.SetStatus(http.StatusOK)
	_, _ = io.Copy(ctx.BodyWriter(), reader)
}
