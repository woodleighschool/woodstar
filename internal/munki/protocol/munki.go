// Package protocol exposes Munki repository endpoints.
package protocol

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/httpauth"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/artifacts"
	"github.com/woodleighschool/woodstar/internal/munki/artifacts/storage"
)

const plistContentType = "application/x-plist"

// AgentSecretVerifier checks shared agent secrets for protocol access.
type AgentSecretVerifier interface {
	Verify(context.Context, agentauth.Agent, string) (bool, error)
}

// Repository loads raw Munki repository objects.
type Repository interface {
	ResolveClient(context.Context, string) (munki.ClientHost, error)
	Manifest(context.Context, munki.ClientHost, string) ([]byte, error)
	Catalog(context.Context, munki.ClientHost, string) ([]byte, error)
	ArtifactRedirect(context.Context, munki.ClientHost, artifacts.ArtifactKind, string) (string, error)
}

type handler struct {
	secretVerifier AgentSecretVerifier
	repository     Repository
	logger         *slog.Logger
}

// RegisterMunkiRoutes mounts Munki client repository endpoints.
func RegisterMunkiRoutes(
	r chi.Router,
	secretVerifier AgentSecretVerifier,
	repository Repository,
	logger *slog.Logger,
) {
	h := handler{
		secretVerifier: secretVerifier,
		repository:     repository,
		logger:         logger,
	}
	r.Get("/munki/manifests/{name}", h.manifest)
	r.Get("/munki/catalogs/{name}", h.catalog)
	r.Get("/munki/pkgs/*", h.packageArtifact)
	r.Get("/munki/icons/*", h.iconArtifact)
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

func (h handler) packageArtifact(w http.ResponseWriter, r *http.Request) {
	h.artifact(w, r, artifacts.ArtifactKindPackage)
}

func (h handler) iconArtifact(w http.ResponseWriter, r *http.Request) {
	h.artifact(w, r, artifacts.ArtifactKindIcon)
}

func (h handler) artifact(w http.ResponseWriter, r *http.Request, kind artifacts.ArtifactKind) {
	authorized, err := h.authorized(r)
	if err != nil {
		h.log(r, http.StatusInternalServerError, "artifact", err)
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
		h.log(r, http.StatusInternalServerError, "artifact", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if h.repository == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	location, err := h.repository.ArtifactRedirect(r.Context(), client, kind, chi.URLParam(r, "*"))
	if errors.Is(err, munki.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if errors.Is(err, storage.ErrUnavailable) {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		h.log(r, http.StatusInternalServerError, "artifact", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, location, http.StatusFound)
}

func (h handler) writePlist(
	w http.ResponseWriter,
	r *http.Request,
	operation string,
	load func(context.Context, munki.ClientHost, string) ([]byte, error),
) {
	authorized, err := h.authorized(r)
	if err != nil {
		h.log(r, http.StatusInternalServerError, operation, err)
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
		h.log(r, http.StatusInternalServerError, operation, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	body, err := load(r.Context(), client, chi.URLParam(r, "name"))
	if errors.Is(err, munki.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		h.log(r, http.StatusInternalServerError, operation, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", plistContentType)
	_, err = w.Write(body)
	if err != nil {
		h.log(r, http.StatusInternalServerError, operation, err)
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

func (h handler) log(r *http.Request, statusCode int, operation string, err error) {
	if h.logger == nil {
		return
	}
	h.logger.WarnContext(
		r.Context(),
		"munki protocol request failed",
		"operation", operation,
		"status", statusCode,
		"path", r.URL.Path,
		"err", err,
	)
}
