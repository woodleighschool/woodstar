package worker

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/munki/mdp/grant"
	"github.com/woodleighschool/woodstar/internal/storage/capability"
)

type server struct {
	mirror *mirror
	key    []byte
	logger *slog.Logger
}

func (s *server) handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/munki/pkgs/*", s.serve)
	return r
}

// serve streams a mirrored installer to a Munki client after verifying its
// grant. Status codes: 401 invalid grant, 410 expired, 404 not mirrored, 409
// stale or mismatched bytes, 416 bad range (via ServeContent).
func (s *server) serve(w http.ResponseWriter, r *http.Request) {
	claims, err := grant.Verify(s.key, r.URL.Query().Get("cap"), time.Now())
	switch {
	case errors.Is(err, capability.ErrExpired):
		s.reject(w, r, http.StatusGone, "grant expired")
		return
	case err != nil:
		s.reject(w, r, http.StatusUnauthorized, "invalid grant")
		return
	}
	if claims.InstallerItemLocation != chi.URLParam(r, "*") {
		s.reject(w, r, http.StatusUnauthorized, "grant path mismatch")
		return
	}

	state, ok := s.mirror.get(claims.PackageID)
	if !ok {
		s.reject(w, r, http.StatusNotFound, "not mirrored")
		return
	}
	// The grant must bind to the bytes the worker has verified. A mismatch means
	// the mirror is stale relative to what Woodstar expects.
	if claims.SHA256 != state.SHA256 || claims.SizeBytes != state.SizeBytes {
		s.reject(w, r, http.StatusConflict, "mirror stale")
		return
	}

	path := s.mirror.localPath(claims.PackageID, state.Filename)
	file, err := os.Open(path) //nolint:gosec // mirror.localPath confines the file to the configured data directory.
	if errors.Is(err, os.ErrNotExist) {
		s.reject(w, r, http.StatusNotFound, "file missing")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if info.Size() != state.SizeBytes {
		s.reject(w, r, http.StatusConflict, "size mismatch")
		return
	}

	s.logger.DebugContext(r.Context(), "serving package",
		"package_id", claims.PackageID,
	)
	// Serve-path integrity is the grant/mirror SHA match plus this size stat, not
	// a full re-hash. Installers are large, their bytes were hashed at download
	// time, and Munki re-verifies installer_item_hash on the client.
	http.ServeContent(w, r, state.Filename, info.ModTime(), file)
}

// reject answers a request that cannot be served and records why, since the
// status code alone does not say which gate failed.
func (s *server) reject(w http.ResponseWriter, r *http.Request, status int, reason string) {
	s.logger.DebugContext(r.Context(), "serve rejected",
		"path", chi.URLParam(r, "*"),
		"status", status,
		"reason", reason,
	)
	w.WriteHeader(status)
}

// fail answers an unexpected serve error.
func (s *server) fail(w http.ResponseWriter, r *http.Request, err error) {
	s.logger.WarnContext(r.Context(), "serve failed", "path", r.URL.Path, "err", err)
	w.WriteHeader(http.StatusInternalServerError)
}
