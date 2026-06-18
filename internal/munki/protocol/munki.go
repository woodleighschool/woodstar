// Package protocol exposes Munki repository endpoints.
package protocol

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/httpauth"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const plistContentType = "application/x-plist"

// Repository loads raw Munki repository objects.
type Repository interface {
	ResolveClient(context.Context, string) (munki.ClientHost, error)
	Manifest(context.Context, munki.ClientHost, string) ([]byte, error)
	Catalog(context.Context, munki.ClientHost, string) ([]byte, error)
	ResolvePackageFile(context.Context, munki.ClientHost, string) (munki.PackageInstaller, error)
	ResolveIconFile(context.Context, munki.ClientHost, string) (string, error)
}

// Selector redirects a package download to an eligible distribution point. A nil
// selector keeps every download Woodstar-direct.
type Selector interface {
	SelectRedirect(ctx context.Context, req mdp.SelectionRequest) (string, bool)
}

type handler struct {
	secretVerifier agentauth.SecretVerifier
	repository     Repository
	selector       Selector
	store          storage.Presigner
	logger         *slog.Logger
}

// RegisterMunkiRoutes mounts Munki client repository endpoints. selector may be
// nil, in which case package downloads are always served Woodstar-direct.
func RegisterMunkiRoutes(
	r chi.Router,
	secretVerifier agentauth.SecretVerifier,
	repository Repository,
	selector Selector,
	store storage.Presigner,
	logger *slog.Logger,
) {
	h := handler{
		secretVerifier: secretVerifier,
		repository:     repository,
		selector:       selector,
		store:          store,
		logger:         logger,
	}
	r.Get("/munki/manifests/{name}", h.manifest)
	r.Get("/munki/catalogs/{name}", h.catalog)
	r.Get("/munki/pkgs/*", h.packageFile)
	r.Get("/munki/icons/*", h.iconFile)
}

func (h handler) manifest(w http.ResponseWriter, r *http.Request) {
	h.writePlist(w, r, "manifest", func(ctx context.Context, client munki.ClientHost, name string) ([]byte, error) {
		if h.repository == nil {
			return nil, munki.ErrNotFound
		}
		return h.repository.Manifest(ctx, client, name)
	})
}

func (h handler) catalog(w http.ResponseWriter, r *http.Request) {
	h.writePlist(w, r, "catalog", func(ctx context.Context, client munki.ClientHost, name string) ([]byte, error) {
		if h.repository == nil {
			return nil, munki.ErrNotFound
		}
		return h.repository.Catalog(ctx, client, name)
	})
}

func (h handler) packageFile(w http.ResponseWriter, r *http.Request) {
	client, ok := h.authorizedClient(w, r)
	if !ok {
		return
	}
	installer, err := h.repository.ResolvePackageFile(r.Context(), client, chi.URLParam(r, "*"))
	if errors.Is(err, munki.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		h.log(r, "package", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if url, ok := h.redirectToDistributionPoint(r, client, installer); ok {
		// Target is the admin-configured distribution point base URL plus a
		// server-signed grant, not client input.
		http.Redirect(w, r, url, http.StatusFound) //nolint:gosec // server-minted redirect target
		return
	}
	h.deliver(w, r, installer.Key)
}

func (h handler) iconFile(w http.ResponseWriter, r *http.Request) {
	client, ok := h.authorizedClient(w, r)
	if !ok {
		return
	}
	key, err := h.repository.ResolveIconFile(r.Context(), client, chi.URLParam(r, "*"))
	if errors.Is(err, munki.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		h.log(r, "icon", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.deliver(w, r, key)
}

// redirectToDistributionPoint asks the selector for an eligible distribution
// point. It reports false to fall back to Woodstar-direct delivery.
func (h handler) redirectToDistributionPoint(
	r *http.Request,
	client munki.ClientHost,
	installer munki.PackageInstaller,
) (string, bool) {
	if h.selector == nil {
		return "", false
	}
	return h.selector.SelectRedirect(r.Context(), mdp.SelectionRequest{
		ClientIP:  chimiddleware.GetClientIP(r.Context()),
		HostID:    client.ID,
		Serial:    client.Serial,
		PackageID: installer.PackageID,
		SHA256:    installer.SHA256,
		SizeBytes: installer.SizeBytes,
	})
}

func (h handler) authorizedClient(w http.ResponseWriter, r *http.Request) (munki.ClientHost, bool) {
	authorized, err := h.authorized(r)
	if err != nil {
		h.log(r, "file", err)
		w.WriteHeader(http.StatusInternalServerError)
		return munki.ClientHost{}, false
	}
	if !authorized {
		w.WriteHeader(http.StatusUnauthorized)
		return munki.ClientHost{}, false
	}
	client, err := h.clientHost(r)
	if errors.Is(err, munki.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return munki.ClientHost{}, false
	}
	if err != nil {
		h.log(r, "file", err)
		w.WriteHeader(http.StatusInternalServerError)
		return munki.ClientHost{}, false
	}
	return client, true
}

// deliver serves the blob Woodstar-direct through a short-lived transfer URL.
func (h handler) deliver(w http.ResponseWriter, r *http.Request, key string) {
	if h.store == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	url, err := h.store.PresignGet(r.Context(), key, 0, storage.GetOptions{})
	if err != nil {
		h.log(r, "file", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Target is a storage-backend presigned URL, not client input.
	http.Redirect(w, r, url, http.StatusFound) //nolint:gosec // server-minted redirect target
}

func (h handler) writePlist(
	w http.ResponseWriter,
	r *http.Request,
	operation string,
	load func(context.Context, munki.ClientHost, string) ([]byte, error),
) {
	authorized, err := h.authorized(r)
	if err != nil {
		h.log(r, operation, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !authorized {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	client, err := h.clientHost(r)
	if errors.Is(err, munki.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		h.log(r, operation, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	body, err := load(r.Context(), client, chi.URLParam(r, "name"))
	if errors.Is(err, munki.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		h.log(r, operation, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", plistContentType)
	_, err = w.Write(body)
	if err != nil {
		h.log(r, operation, err)
	}
}

func (h handler) authorized(r *http.Request) (bool, error) {
	token, ok := httpauth.BearerToken(r.Header.Get("Authorization"))
	if !ok || h.secretVerifier == nil {
		return false, nil
	}
	ok, err := h.secretVerifier.Verify(r.Context(), agentauth.AgentMunki, token)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (h handler) clientHost(r *http.Request) (munki.ClientHost, error) {
	if h.repository == nil {
		return munki.ClientHost{}, munki.ErrNotFound
	}
	serial := strings.TrimSpace(r.Header.Get("Serial"))
	if serial == "" {
		return munki.ClientHost{}, munki.ErrNotFound
	}
	return h.repository.ResolveClient(r.Context(), serial)
}

func (h handler) log(r *http.Request, operation string, err error) {
	if h.logger == nil {
		return
	}
	h.logger.WarnContext(
		r.Context(),
		"munki protocol request failed",
		"operation", operation,
		"status", http.StatusInternalServerError,
		"path", r.URL.Path,
		"err", err,
	)
}
