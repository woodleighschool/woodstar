package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/goodies/bloby"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
)

const munkiAssetCacheControl = "private, max-age=86400"

type munkiContentHandler struct {
	objects *bloby.Service
	logger  *slog.Logger
}

func registerMunkiContentRoutes(
	softwareRouter chi.Router,
	packagesRouter chi.Router,
	clientResourcesRouter chi.Router,
	objects *bloby.Service,
	logger *slog.Logger,
) {
	h := munkiContentHandler{
		objects: objects,
		logger:  logger,
	}
	softwareRouter.Get(munkiIconPath+"/{id}/content", h.object(munkisoftware.IconObjectPrefix, munkiAssetCacheControl))
	packagesRouter.Get(munkiPackageInstallerPath+"/{id}/content", h.object(packages.ObjectPrefix, ""))
	clientResourcesRouter.Get(
		clientResourcesBannerUploadPath+"/{id}/content",
		h.object(clientresources.BannerObjectPrefix, munkiAssetCacheControl),
	)
	clientResourcesRouter.Get(
		clientResourcesArchiveUploadPath+"/{id}/content",
		h.object(clientresources.ArchiveObjectPrefix, ""),
	)
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
	if errors.Is(err, bloby.ErrNotFound) {
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
	if err := h.objects.Deliver(w, r, *object, bloby.DeliveryOptions{
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
	_ = api.HandlerError(r.Context(), h.logger, operation, err, attrs...)
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

func contentURL(basePath string, objectID int64) string {
	if objectID <= 0 {
		return ""
	}
	return basePath + "/" + strconv.FormatInt(objectID, 10) + "/content"
}
