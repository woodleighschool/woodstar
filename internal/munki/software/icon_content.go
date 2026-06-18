package software

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	munkiupload "github.com/woodleighschool/woodstar/internal/munki/objectupload"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const iconCacheControl = "private, max-age=86400"

// RegisterIconContentRoute mounts the session-authenticated software icon proxy.
func RegisterIconContentRoute(
	r chi.Router,
	store *Store,
	objects *storage.ObjectStore,
	backend storage.Store,
) {
	h := iconContentHandler{store: store, objects: objects, backend: backend}
	r.Get(munkiSoftwareIDPath+"/icon", h.get)
}

type iconContentHandler struct {
	store   *Store
	objects *storage.ObjectStore
	backend storage.Store
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if obj.Prefix != munkiupload.IconObjectPrefix || !obj.Available() {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	storage.ServeObject(w, r, h.backend, obj.Key(), storage.ServeOptions{
		ContentType:  obj.ContentType,
		Filename:     obj.Filename,
		CacheControl: iconCacheControl,
		ETag:         objectETag(obj),
	})
}

func objectETag(obj *storage.Object) string {
	if obj == nil || obj.SHA256 == nil || *obj.SHA256 == "" {
		return ""
	}
	return `"` + *obj.SHA256 + `"`
}
