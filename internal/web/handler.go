package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type runtimeConfig struct {
	APIBaseURL string `json:"apiBaseURL"`
	Version    string `json:"version"`
	CSRFToken  string `json:"csrfToken"`
}

// HandlerOptions configures the embedded web UI handler.
type HandlerOptions struct {
	FS        fs.FS
	PublicURL string
	Version   string
	CSRFToken func(*http.Request) string
}

// Handler serves the embedded frontend bundle and runtime config.
type Handler struct {
	fs        fs.FS
	publicURL string
	version   string
	csrfToken func(*http.Request) string
	assets    http.Handler
}

// NewHandler returns an HTTP handler for the embedded web UI.
func NewHandler(opts HandlerOptions) *Handler {
	h := &Handler{
		fs:        opts.FS,
		publicURL: strings.TrimRight(opts.PublicURL, "/"),
		version:   opts.Version,
		csrfToken: opts.CSRFToken,
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
		log.Error().Err(err).Msg("embedded web index missing")
		http.Error(w, "web bundle missing", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(h.injectRuntime(r, content))
}

func (h *Handler) injectRuntime(r *http.Request, content []byte) []byte {
	csrf := ""
	if h.csrfToken != nil {
		csrf = h.csrfToken(r)
	}

	data, err := json.Marshal(runtimeConfig{
		APIBaseURL: h.publicURL,
		Version:    h.version,
		CSRFToken:  csrf,
	})
	if err != nil {
		return content
	}

	scriptTag := fmt.Appendf(nil, `<script>window.__WOODSTAR__=%s;</script></head>`, data)

	return bytes.Replace(content, []byte("</head>"), scriptTag, 1)
}
