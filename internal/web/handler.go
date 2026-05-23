package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type runtimeConfig struct {
	Version string `json:"version"`
}

// HandlerOptions configures the embedded web UI handler.
type HandlerOptions struct {
	FS      fs.FS
	Version string
	Logger  *slog.Logger
}

// Handler serves the embedded frontend bundle and runtime config.
type Handler struct {
	assets   http.Handler
	index    []byte
	indexErr error
	logger   *slog.Logger
}

// NewHandler returns an HTTP handler for the embedded web UI.
func NewHandler(opts HandlerOptions) *Handler {
	h := &Handler{
		logger: opts.Logger,
	}
	if opts.FS != nil {
		h.assets = http.FileServer(http.FS(opts.FS))
		h.index, h.indexErr = renderIndex(opts.FS, opts.Version)
	}
	return h
}

// RegisterRoutes attaches static asset and SPA fallback routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/assets/*", h.serveAsset)
	r.Get("/favicon.ico", h.serveAsset)
	r.Get("/favicon.png", h.serveAsset)
	r.Get("/", h.serveIndex)
	r.Get("/*", h.serveIndex)
}

func (h *Handler) serveAsset(w http.ResponseWriter, r *http.Request) {
	if h.assets == nil {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	h.assets.ServeHTTP(w, r)
}

func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
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

func renderIndex(fsys fs.FS, version string) ([]byte, error) {
	content, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(runtimeConfig{
		Version: version,
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
