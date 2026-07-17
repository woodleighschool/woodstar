// Package protocol exposes Munki repository endpoints.
package protocol

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/httpx"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const plistContentType = "application/x-plist"

// Repository loads raw Munki repository objects.
type Repository interface {
	Manifest(ctx context.Context, name string) ([]byte, error)
	Catalog(ctx context.Context, name string) ([]byte, error)
	IconHashes(ctx context.Context) ([]byte, error)
	ResolvePackageFile(ctx context.Context, name string) (munki.PackageInstaller, error)
	ResolveIconFile(ctx context.Context, name string) (munki.RepositoryFile, error)
	ResolveClientResources(ctx context.Context, name string) (munki.RepositoryFile, error)
}

// Selector redirects a package download to a matching distribution point.
type Selector interface {
	SelectRedirect(ctx context.Context, req mdp.SelectionRequest) (string, bool)
}

type handler struct {
	secretVerifier agentauth.SecretVerifier
	repository     Repository
	selector       Selector
	storage        storage.Presigner
	logger         *slog.Logger
}

// Server owns Munki client repository routes.
type Server struct {
	secretVerifier agentauth.SecretVerifier
	repository     Repository
	selector       Selector
	storage        storage.Presigner
	logger         *slog.Logger
}

// NewServer returns a Munki client repository protocol server.
func NewServer(
	secretVerifier agentauth.SecretVerifier,
	repository Repository,
	selector Selector,
	storage storage.Presigner,
	logger *slog.Logger,
) *Server {
	return &Server{
		secretVerifier: secretVerifier,
		repository:     repository,
		selector:       selector,
		storage:        storage,
		logger:         logger,
	}
}

// RegisterRoutes mounts Munki client repository endpoints.
func (s *Server) RegisterRoutes(r chi.Router) {
	h := handler{
		secretVerifier: s.secretVerifier,
		repository:     s.repository,
		selector:       s.selector,
		storage:        s.storage,
		logger:         s.logger,
	}
	r.Get("/munki/manifests/{name}", h.manifest)
	r.Get("/munki/catalogs/{name}", h.catalog)
	r.Get("/munki/pkgs/*", h.packageFile)
	r.Get("/munki/icons/_icon_hashes.plist", h.iconHashes)
	r.Get("/munki/icons/*", h.iconFile)
	r.Get("/munki/client_resources/*", h.clientResources)
}

func (h handler) manifest(w http.ResponseWriter, r *http.Request) {
	h.writePlist(w, r, "manifest", func(ctx context.Context) ([]byte, error) {
		return h.repository.Manifest(ctx, chi.URLParam(r, "name"))
	})
}

func (h handler) catalog(w http.ResponseWriter, r *http.Request) {
	h.writePlist(w, r, "catalog", func(ctx context.Context) ([]byte, error) {
		return h.repository.Catalog(ctx, chi.URLParam(r, "name"))
	})
}

func (h handler) iconHashes(w http.ResponseWriter, r *http.Request) {
	h.writePlist(w, r, "icon hashes", h.repository.IconHashes)
}

func (h handler) packageFile(w http.ResponseWriter, r *http.Request) {
	if ok := h.authorizedRequest(w, r, "package"); !ok {
		return
	}
	installer, err := h.repository.ResolvePackageFile(r.Context(), chi.URLParam(r, "*"))
	if errors.Is(err, munki.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		h.log(r, "package", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if url, ok := h.redirectToDistributionPoint(r, installer); ok {
		// Target is the admin-configured distribution point base URL plus a
		// server-signed grant, not client input.
		http.Redirect(w, r, url, http.StatusFound) //nolint:gosec // Server-minted distribution URL.
		return
	}
	h.deliver(w, r, munki.RepositoryFile{Key: installer.Key, ContentType: installer.ContentType})
}

func (h handler) iconFile(w http.ResponseWriter, r *http.Request) {
	if ok := h.authorizedRequest(w, r, "icon"); !ok {
		return
	}
	file, err := h.repository.ResolveIconFile(r.Context(), chi.URLParam(r, "*"))
	if errors.Is(err, munki.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		h.log(r, "icon", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.deliver(w, r, file)
}

func (h handler) clientResources(w http.ResponseWriter, r *http.Request) {
	if ok := h.authorizedRequest(w, r, "client resources"); !ok {
		return
	}
	file, err := h.repository.ResolveClientResources(r.Context(), chi.URLParam(r, "*"))
	if errors.Is(err, munki.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		h.log(r, "client resources", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.deliver(w, r, file)
}

func (h handler) authorizedRequest(w http.ResponseWriter, r *http.Request, operation string) bool {
	authorized, err := h.authorized(r)
	if err != nil {
		h.log(r, operation, err)
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}
	if !authorized {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

// redirectToDistributionPoint asks the selector for a matching distribution point.
func (h handler) redirectToDistributionPoint(
	r *http.Request,
	installer munki.PackageInstaller,
) (string, bool) {
	return h.selector.SelectRedirect(r.Context(), mdp.SelectionRequest{
		ClientIP:              chimiddleware.GetClientIP(r.Context()),
		PackageID:             installer.PackageID,
		InstallerItemLocation: installer.InstallerItemLocation,
		SHA256:                installer.SHA256,
		SizeBytes:             installer.SizeBytes,
	})
}

// deliver serves the blob Woodstar-direct through a short-lived transfer URL.
func (h handler) deliver(w http.ResponseWriter, r *http.Request, file munki.RepositoryFile) {
	url, err := h.storage.PresignGet(
		r.Context(),
		file.Key,
		0,
		storage.GetOptions{ContentType: file.ContentType},
	)
	if err != nil {
		h.log(r, "file", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Target is a storage-backend presigned URL, not client input.
	http.Redirect(w, r, url, http.StatusFound)
}

func (h handler) writePlist(
	w http.ResponseWriter,
	r *http.Request,
	operation string,
	load func(context.Context) ([]byte, error),
) {
	if ok := h.authorizedRequest(w, r, operation); !ok {
		return
	}
	body, err := load(r.Context())
	if errors.Is(err, munki.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		h.log(r, operation, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	etag := responseETag(body)
	w.Header().Set("ETag", etag)
	if requestETagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", plistContentType)
	_, err = w.Write(body)
	if err != nil {
		h.log(r, operation, err)
	}
}

func responseETag(body []byte) string {
	sum := sha256.Sum256(body)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func requestETagMatches(header string, etag string) bool {
	for value := range strings.SplitSeq(header, ",") {
		value = strings.TrimSpace(value)
		if value == "*" || value == etag {
			return true
		}
	}
	return false
}

func (h handler) authorized(r *http.Request) (bool, error) {
	token, ok := httpx.BearerToken(r.Header.Get("Authorization"))
	if !ok {
		return false, nil
	}
	ok, err := h.secretVerifier.Verify(r.Context(), agentauth.AgentMunki, token)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (h handler) log(r *http.Request, operation string, err error) {
	h.logger.WarnContext(
		r.Context(),
		"munki protocol request failed",
		"operation", operation,
		"status", http.StatusInternalServerError,
		"path", r.URL.Path,
		"err", err,
	)
}
