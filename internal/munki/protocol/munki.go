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
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/heartbeats"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/httpx"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	"github.com/woodleighschool/woodstar/internal/storage"
)

const (
	plistContentType   = "application/x-plist"
	serialNumberHeader = "X-Woodstar-Serial-Number"
)

type hostResolver interface {
	GetByHardwareSerial(context.Context, string) (*hosts.Host, error)
}

type heartbeatRecorder interface {
	Record(context.Context, int64, heartbeats.Source, heartbeats.Contact) error
}

// Repository loads raw Munki repository objects.
type Repository interface {
	Manifest(context.Context, int64) ([]byte, error)
	Catalog(context.Context, int64, string) ([]byte, error)
	IconHashes(context.Context, int64) ([]byte, error)
	ResolvePackageFile(context.Context, int64, string) (munki.PackageInstaller, error)
	ResolveIconFile(context.Context, int64, string) (storage.Object, error)
	ResolveClientResources(context.Context, int64, string) (storage.Object, error)
}

// Selector redirects a package download to a matching distribution point.
type Selector interface {
	SelectRedirect(ctx context.Context, req mdp.SelectionRequest) (string, bool)
}

type handler struct {
	secretVerifier agentauth.SecretVerifier
	hostResolver   hostResolver
	repository     Repository
	heartbeats     heartbeatRecorder
	selector       Selector
	delivery       storage.Deliverer
	logger         *slog.Logger
}

// Server owns Munki client repository routes.
type Server struct {
	secretVerifier agentauth.SecretVerifier
	hostResolver   hostResolver
	repository     Repository
	heartbeats     heartbeatRecorder
	selector       Selector
	delivery       storage.Deliverer
	logger         *slog.Logger
}

// NewServer returns a Munki client repository protocol server.
func NewServer(
	secretVerifier agentauth.SecretVerifier,
	hostResolver hostResolver,
	repository Repository,
	heartbeats heartbeatRecorder,
	selector Selector,
	delivery storage.Deliverer,
	logger *slog.Logger,
) *Server {
	return &Server{
		secretVerifier: secretVerifier,
		hostResolver:   hostResolver,
		repository:     repository,
		heartbeats:     heartbeats,
		selector:       selector,
		delivery:       delivery,
		logger:         logger,
	}
}

// RegisterRoutes mounts Munki metadata endpoints on ordinary and byte-transfer
// endpoints on transfers.
func (s *Server) RegisterRoutes(ordinary chi.Router, transfers chi.Router) {
	h := handler{
		secretVerifier: s.secretVerifier,
		hostResolver:   s.hostResolver,
		repository:     s.repository,
		heartbeats:     s.heartbeats,
		selector:       s.selector,
		delivery:       s.delivery,
		logger:         s.logger,
	}
	ordinary.Get("/munki/manifests/{name}", h.manifest)
	ordinary.Get("/munki/catalogs/{name}", h.catalog)
	ordinary.Get("/munki/icons/_icon_hashes.plist", h.iconHashes)
	transfers.Get("/munki/pkgs/*", h.packageFile)
	transfers.Get("/munki/icons/*", h.iconFile)
	transfers.Get("/munki/client_resources/*", h.clientResources)
}

func (h handler) manifest(w http.ResponseWriter, r *http.Request) {
	h.writePlist(w, r, "manifest", func(ctx context.Context, hostID int64) ([]byte, error) {
		return h.repository.Manifest(ctx, hostID)
	})
}

func (h handler) catalog(w http.ResponseWriter, r *http.Request) {
	h.writePlist(w, r, "catalog", func(ctx context.Context, hostID int64) ([]byte, error) {
		return h.repository.Catalog(ctx, hostID, httpx.PathParam(r, "name"))
	})
}

func (h handler) iconHashes(w http.ResponseWriter, r *http.Request) {
	h.writePlist(w, r, "icon hashes", h.repository.IconHashes)
}

func (h handler) packageFile(w http.ResponseWriter, r *http.Request) {
	hostID, ok := h.authorizedHost(w, r, "package")
	if !ok {
		return
	}
	installer, err := h.repository.ResolvePackageFile(r.Context(), hostID, httpx.PathParam(r, "*"))
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
	h.deliver(w, r, installer.Object)
}

func (h handler) iconFile(w http.ResponseWriter, r *http.Request) {
	hostID, ok := h.authorizedHost(w, r, "icon")
	if !ok {
		return
	}
	file, err := h.repository.ResolveIconFile(r.Context(), hostID, httpx.PathParam(r, "*"))
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
	hostID, ok := h.authorizedHost(w, r, "client resources")
	if !ok {
		return
	}
	file, err := h.repository.ResolveClientResources(r.Context(), hostID, httpx.PathParam(r, "*"))
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

func (h handler) authorizedHost(w http.ResponseWriter, r *http.Request, operation string) (int64, bool) {
	authorized, err := h.authorized(r)
	if err != nil {
		h.log(r, operation, err)
		w.WriteHeader(http.StatusInternalServerError)
		return 0, false
	}
	if !authorized {
		w.WriteHeader(http.StatusUnauthorized)
		return 0, false
	}
	host, err := h.hostResolver.GetByHardwareSerial(r.Context(), strings.TrimSpace(r.Header.Get(serialNumberHeader)))
	if errors.Is(err, fault.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return 0, false
	}
	if err != nil {
		h.log(r, operation, err)
		w.WriteHeader(http.StatusInternalServerError)
		return 0, false
	}
	if err := h.heartbeats.Record(r.Context(), host.ID, heartbeats.SourceMunki, heartbeats.Contact{
		RemoteIP:  chimiddleware.GetClientIP(r.Context()),
		UserAgent: r.UserAgent(),
	}); err != nil {
		h.log(r, operation, err)
		w.WriteHeader(http.StatusInternalServerError)
		return 0, false
	}
	return host.ID, true
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
		SHA256:                installer.Object.SHA256Value(),
		SizeBytes:             installer.Object.SizeBytesValue(),
	})
}

// deliver hands the resolved object to storage for backend-appropriate delivery.
func (h handler) deliver(w http.ResponseWriter, r *http.Request, object storage.Object) {
	if err := h.delivery.Deliver(w, r, object, storage.DeliveryOptions{}); err != nil {
		h.log(r, "file", err)
	}
}

func (h handler) writePlist(
	w http.ResponseWriter,
	r *http.Request,
	operation string,
	load func(context.Context, int64) ([]byte, error),
) {
	hostID, ok := h.authorizedHost(w, r, operation)
	if !ok {
		return
	}
	body, err := load(r.Context(), hostID)
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
