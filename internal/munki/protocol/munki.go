// Package protocol exposes Munki repository endpoints.
package protocol

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/munki"
)

const plistContentType = "application/x-plist"

// AgentSecretVerifier checks shared agent secrets for protocol access.
type AgentSecretVerifier interface {
	Verify(context.Context, agentauth.Agent, string) (bool, error)
}

// Repository loads raw Munki repository objects.
type Repository interface {
	Manifest(context.Context, string) ([]byte, error)
	Catalog(context.Context, string) ([]byte, error)
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
}

func (h handler) manifest(w http.ResponseWriter, r *http.Request) {
	h.writePlist(w, r, "manifest", func(ctx context.Context, name string) ([]byte, error) {
		if h.repository == nil {
			return nil, munki.ErrNotFound
		}
		return h.repository.Manifest(ctx, name)
	})
}

func (h handler) catalog(w http.ResponseWriter, r *http.Request) {
	h.writePlist(w, r, "catalog", func(ctx context.Context, name string) ([]byte, error) {
		if h.repository == nil {
			return nil, munki.ErrNotFound
		}
		return h.repository.Catalog(ctx, name)
	})
}

func (h handler) writePlist(
	w http.ResponseWriter,
	r *http.Request,
	operation string,
	load func(context.Context, string) ([]byte, error),
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
	body, err := load(r.Context(), chi.URLParam(r, "name"))
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
	token, ok := agentauth.BearerToken(r.Header.Get("Authorization"))
	if !ok || h.secretVerifier == nil {
		return false, nil
	}
	ok, err := h.secretVerifier.Verify(r.Context(), agentauth.AgentMunki, token)
	if err != nil {
		return false, err
	}
	return ok, nil
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
