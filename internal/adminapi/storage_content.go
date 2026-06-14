package adminapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/adminctx"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// registerStorageContentUpload mounts a streaming receiver for storage backends
// without presigned uploads (file). The request body streams straight to the
// store, so large installers never buffer in memory the way Huma's RawBody
// would. Auth mirrors the admin Huma group: a Bearer API key or session cookie,
// administrators only.
func registerStorageContentUpload(r chi.Router, deps Dependencies) {
	objects := deps.Munki.Objects
	store := deps.Munki.Store
	authService := deps.Auth.AuthService

	r.Put("/api/objects/{id}/content", func(w http.ResponseWriter, req *http.Request) {
		user, err := authService.Authenticate(req.Context(), req.Header.Get("Authorization"))
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, auth.ErrNotAuthenticated) {
				status = http.StatusUnauthorized
			}
			http.Error(w, http.StatusText(status), status)
			return
		}
		if _, err := adminctx.RequireAdmin(adminctx.WithUser(req.Context(), user)); err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

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
			storage.PutOptions{ContentType: obj.ContentType},
		); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
