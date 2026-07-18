package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const munkiAssetCacheControl = "private, max-age=86400"

type munkiContentHandler struct {
	software *munkisoftware.Store
	objects  *storage.ObjectStore
	delivery storage.Deliverer
	logger   *slog.Logger
}

func registerMunkiContentRoutes(
	r chi.Router,
	software *munkisoftware.Store,
	objects *storage.ObjectStore,
	delivery storage.Deliverer,
	logger *slog.Logger,
) {
	h := munkiContentHandler{
		software: software,
		objects:  objects,
		delivery: delivery,
		logger:   logger,
	}
	r.Get(munkiSoftwareIDPath+"/icon", h.softwareIcon)
	r.Get(munkiIconPath+"/{id}/content", h.object(munkisoftware.IconObjectPrefix, munkiAssetCacheControl))
	r.Get(munkiPackageInstallerPath+"/{id}/content", h.object(packages.ObjectPrefix, ""))
	r.Get(
		clientResourcesBannerPath+"/{id}/content",
		h.object(clientresources.BannerObjectPrefix, munkiAssetCacheControl),
	)
}

func (h munkiContentHandler) softwareIcon(w http.ResponseWriter, r *http.Request) {
	softwareID, ok := contentObjectID(w, r)
	if !ok {
		return
	}
	software, err := h.software.GetByID(r.Context(), softwareID)
	if errors.Is(err, dbutil.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		h.fail(w, r, "get-munki-software-icon", err, "software_id", softwareID)
		return
	}
	if software.IconObjectID == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	h.deliver(w, r, *software.IconObjectID, munkisoftware.IconObjectPrefix, munkiAssetCacheControl)
}

func (h munkiContentHandler) object(prefix, cacheControl string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		objectID, ok := contentObjectID(w, r)
		if !ok {
			return
		}
		h.deliver(w, r, objectID, prefix, cacheControl)
	}
}

func (h munkiContentHandler) deliver(
	w http.ResponseWriter,
	r *http.Request,
	objectID int64,
	prefix string,
	cacheControl string,
) {
	object, err := h.objects.GetByID(r.Context(), objectID)
	if errors.Is(err, dbutil.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		h.fail(w, r, "get-munki-content", err, "object_id", objectID)
		return
	}
	if object.Prefix != prefix || !object.Available() {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err := h.delivery.Deliver(w, r, *object, storage.DeliveryOptions{
		CacheControl: cacheControl,
	}); err != nil {
		h.logger.ErrorContext(
			r.Context(),
			"munki content delivery failed",
			"operation", "deliver-munki-content",
			"object_id", object.ID,
			"err", err,
		)
	}
}

func (h munkiContentHandler) fail(
	w http.ResponseWriter,
	r *http.Request,
	operation string,
	err error,
	attrs ...any,
) {
	_ = handlerError(r.Context(), h.logger, operation, err, attrs...)
	w.WriteHeader(http.StatusInternalServerError)
}

func contentObjectID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func munkiPackageInstallerContentURL(objectID int64) string {
	return contentURL(munkiPackageInstallerPath, objectID)
}

func munkiIconContentURL(objectID int64) string {
	return contentURL(munkiIconPath, objectID)
}

func clientResourcesBannerContentURL(objectID int64) string {
	return contentURL(clientResourcesBannerPath, objectID)
}

func contentURL(basePath string, objectID int64) string {
	if objectID <= 0 {
		return ""
	}
	return basePath + "/" + strconv.FormatInt(objectID, 10) + "/content"
}
