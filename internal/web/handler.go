package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type runtimeConfig struct {
	Version   string `json:"version"`
	CSRFToken string `json:"csrfToken"`
}

// HandlerOptions configures the embedded web UI handler.
type HandlerOptions struct {
	FS        fs.FS
	Version   string
	CSRFToken func(*http.Request) string
	Logger    *slog.Logger
}

// Handler serves the embedded frontend bundle and runtime config.
type Handler struct {
	fs        fs.FS
	version   string
	csrfToken func(*http.Request) string
	assets    http.Handler
	logger    *slog.Logger
}

// NewHandler returns an HTTP handler for the embedded web UI.
func NewHandler(opts HandlerOptions) *Handler {
	h := &Handler{
		fs:        opts.FS,
		version:   opts.Version,
		csrfToken: opts.CSRFToken,
		logger:    opts.Logger,
	}
	if opts.FS != nil {
		h.assets = http.FileServer(http.FS(opts.FS))
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
	if h.fs == nil {
		http.NotFound(w, r)
		return
	}

	file, err := h.fs.Open("index.html")
	if err != nil {
		h.logger.ErrorContext(
			r.Context(),
			"embedded web index missing", "operation",
			"serve_index",
			"err",
			err,
		)
		http.Error(w, "web bundle missing", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		h.logger.ErrorContext(
			r.Context(),
			"embedded web index read failed", "operation",
			"serve_index",
			"err",
			err,
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if _, err := w.Write(h.injectRuntime(r, content)); err != nil {
		h.logger.DebugContext(r.Context(), "embedded web index write failed", "operation", "serve_index", "err", err)
	}
}

func (h *Handler) injectRuntime(r *http.Request, content []byte) []byte {
	csrf := ""
	if h.csrfToken != nil {
		csrf = h.csrfToken(r)
	}

	data, err := json.Marshal(runtimeConfig{
		Version:   h.version,
		CSRFToken: csrf,
	})
	if err != nil {
		return content
	}

	scriptTag := fmt.Appendf(nil, `<script>window.__WOODSTAR__=%s;</script></head>`, data)

	return bytes.Replace(content, []byte("</head>"), scriptTag, 1)
}
