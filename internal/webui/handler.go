// Package webui serves the embedded Woodstar frontend.
package webui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

type runtimeConfig struct {
	Version   string `json:"version"`
	PublicURL string `json:"public_url"`
}

// HandlerOptions configures the embedded web UI handler.
type HandlerOptions struct {
	FS        fs.FS
	Version   string
	PublicURL string
	Logger    *slog.Logger
}

// Handler serves the embedded frontend bundle and runtime config.
type Handler struct {
	fs       fs.FS
	index    []byte
	indexErr error
	logger   *slog.Logger
}

func init() {
	_ = mime.AddExtensionType(".css", "text/css; charset=utf-8")
	_ = mime.AddExtensionType(".js", "application/javascript; charset=utf-8")
	_ = mime.AddExtensionType(".svg", "image/svg+xml")
}

// NewHandler returns an HTTP handler for the embedded web UI.
func NewHandler(opts HandlerOptions) *Handler {
	h := &Handler{
		fs:     opts.FS,
		logger: opts.Logger,
	}
	if opts.FS != nil {
		h.index, h.indexErr = renderIndex(opts.FS, opts.Version, opts.PublicURL)
	}
	return h
}

// RegisterRoutes attaches static asset and SPA fallback routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/assets/*", h.serveAsset)
	r.Get("/index.html", redirectIndex)
	r.Get("/", h.serveIndex)
	r.Get("/*", h.serveIndex)
}

func (h *Handler) serveAsset(w http.ResponseWriter, r *http.Request) {
	if h.fs == nil {
		http.NotFound(w, r)
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/")
	file, err := h.fs.Open(name)
	if err != nil {
		h.logger.DebugContext(
			r.Context(),
			"embedded web asset missing",
			"operation", "serve_asset",
			"path", r.URL.Path,
			"err", err,
		)
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		h.logger.ErrorContext(
			r.Context(),
			"embedded web asset stat failed",
			"operation", "serve_asset",
			"path", name,
			"err", err,
		)
		http.Error(w, "web asset unavailable", http.StatusInternalServerError)
		return
	}
	if stat.IsDir() {
		http.NotFound(w, r)
		return
	}

	setAssetHeaders(w, name)
	if seeker, ok := file.(io.ReadSeeker); ok {
		http.ServeContent(w, r, name, stat.ModTime(), seeker)
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		h.logger.ErrorContext(
			r.Context(),
			"embedded web asset read failed",
			"operation", "serve_asset",
			"path", name,
			"err", err,
		)
		http.Error(w, "web asset unavailable", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, name, stat.ModTime(), bytes.NewReader(content))
}

func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	if isAssetPath(r.URL.Path) {
		h.serveAsset(w, r)
		return
	}
	if h.index == nil && h.indexErr == nil {
		http.NotFound(w, r)
		return
	}
	if h.indexErr != nil {
		h.logger.ErrorContext(
			r.Context(),
			"embedded web index missing",
			"operation", "serve_index",
			"err", h.indexErr,
		)
		http.Error(w, "web bundle missing", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if _, err := w.Write(h.index); err != nil {
		h.logger.DebugContext(r.Context(), "embedded web index write failed", "operation", "serve_index", "err", err)
	}
}

func redirectIndex(w http.ResponseWriter, r *http.Request) {
	target := "/"
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

func setAssetHeaders(w http.ResponseWriter, name string) {
	if contentType := mime.TypeByExtension(filepath.Ext(name)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
}

func isAssetPath(path string) bool {
	return strings.HasPrefix(path, "/assets/") || filepath.Ext(path) != ""
}

func renderIndex(fsys fs.FS, version string, publicURL string) ([]byte, error) {
	content, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(runtimeConfig{
		Version:   version,
		PublicURL: publicURL,
	})
	if err != nil {
		return nil, err
	}

	headClose := []byte("</head>")
	index := bytes.LastIndex(content, headClose)
	if index < 0 {
		return content, nil
	}

	out := make([]byte, 0, len(content)+len(data)+len(`<script>window.__WOODSTAR__=;</script>`))
	out = append(out, content[:index]...)
	out = fmt.Appendf(out, `<script>window.__WOODSTAR__=%s;</script>`, data)
	out = append(out, content[index:]...)
	return out, nil
}
