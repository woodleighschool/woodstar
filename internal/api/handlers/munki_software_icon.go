package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const iconCacheControl = "private, max-age=86400"

func registerMunkiSoftwareIconContent(
	r chi.Router,
	store *munkisoftware.Store,
	objects *storage.ObjectStore,
	backend storage.Store,
	logger *slog.Logger,
) {
	h := iconContentHandler{store: store, objects: objects, backend: backend, logger: logger}
	r.Get(munkiSoftwareIDPath+"/icon", h.get)
}

type iconContentHandler struct {
	store   *munkisoftware.Store
	objects *storage.ObjectStore
	backend storage.Store
	logger  *slog.Logger
}

func (h iconContentHandler) get(w http.ResponseWriter, r *http.Request) {
	softwareID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || softwareID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	title, err := h.store.GetByID(r.Context(), softwareID)
	if errors.Is(err, dbutil.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		_ = handlerError(r.Context(), h.logger, "get-munki-software-icon-content", err, "software_id", softwareID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if title.IconObjectID == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	obj, err := h.objects.GetByID(r.Context(), *title.IconObjectID)
	if errors.Is(err, dbutil.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		_ = handlerError(
			r.Context(),
			h.logger,
			"get-munki-software-icon-content",
			err,
			"software_id", softwareID,
			"object_id", *title.IconObjectID,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if obj.Prefix != munkisoftware.IconObjectPrefix || !obj.Available() {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := storage.ServeObject(w, r, h.backend, obj.Key(), storage.ServeOptions{
		ContentType:  obj.ContentType,
		Filename:     obj.Filename,
		CacheControl: iconCacheControl,
		ETag:         objectETag(obj),
	}); err != nil {
		_ = handlerError(
			r.Context(),
			h.logger,
			"get-munki-software-icon-content",
			err,
			"software_id", softwareID,
			"object_id", obj.ID,
		)
	}
}

func objectETag(obj *storage.Object) string {
	if obj == nil || obj.SHA256 == nil || *obj.SHA256 == "" {
		return ""
	}
	return `"` + *obj.SHA256 + `"`
}
