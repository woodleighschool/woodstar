package web

import (
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// HandlerOptions configures the embedded web UI handler.
type HandlerOptions struct {
	FS      fs.FS
	BaseURL string
	Version string
}

// Handler serves the embedded frontend bundle and runtime config.
type Handler struct {
	fs       fs.FS
	baseURL  string
	basePath string
	version  string
}

var contentTypesOnce sync.Once

func registerContentTypes() {
	for ext, contentType := range map[string]string{
		".js":   "application/javascript",
		".css":  "text/css",
		".html": "text/html",
		".json": "application/json",
		".svg":  "image/svg+xml",
	} {
		if err := mime.AddExtensionType(ext, contentType); err != nil {
			log.Warn().Err(err).Str("extension", ext).Msg("register web content type")
		}
	}
}

// NewHandler returns an HTTP handler for the embedded web UI.
func NewHandler(opts HandlerOptions) *Handler {
	contentTypesOnce.Do(registerContentTypes)
	return &Handler{
		fs:       opts.FS,
		baseURL:  normalizeBaseURL(opts.BaseURL),
		basePath: basePath(opts.BaseURL),
		version:  opts.Version,
	}
}

// RegisterRoutes attaches static asset and SPA fallback routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/assets/*", h.serveAsset)
	r.Get("/favicon.ico", h.serveAsset)
	r.Get("/favicon.png", h.serveAsset)
	r.Get("/index.html", h.redirectIndex)
	r.Get("/", h.serveIndex)
	r.Get("/*", h.serveIndex)
}

func (h *Handler) serveAsset(w http.ResponseWriter, r *http.Request) {
	if h.fs == nil {
		http.NotFound(w, r)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if h.basePath != "/" {
		path = strings.TrimPrefix(path, strings.TrimPrefix(h.basePath, "/")+"/")
	}
	file, err := h.fs.Open(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("web asset not found")
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if strings.HasPrefix(path, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}

	readSeeker, ok := file.(io.ReadSeeker)
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, path, stat.ModTime(), readSeeker)
}

func (h *Handler) redirectIndex(w http.ResponseWriter, r *http.Request) {
	target := h.basePath
	if target == "" {
		target = "/"
	}
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	//nolint:gosec // target is a normalized local base path, not user-controlled absolute URL input.
	http.Redirect(w, r, target, http.StatusMovedPermanently)
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
	_, _ = w.Write(h.injectRuntime(content))
}

func (h *Handler) injectRuntime(content []byte) []byte {
	scriptTag := fmt.Sprintf(
		`<script>window.__WOODSTAR__={apiBaseURL:%q,baseURL:%q,version:%q};</script>`,
		h.baseURL,
		h.baseURL,
		h.version,
	)
	html := strings.Replace(string(content), "</head>", scriptTag+"</head>", 1)
	if h.basePath != "/" {
		html = strings.ReplaceAll(html, `src="/assets/`, `src="`+h.basePath+`/assets/`)
		html = strings.ReplaceAll(html, `href="/assets/`, `href="`+h.basePath+`/assets/`)
		html = strings.ReplaceAll(html, `href="/favicon.ico"`, `href="`+h.basePath+`/favicon.ico"`)
		html = strings.ReplaceAll(html, `href="/favicon.png"`, `href="`+h.basePath+`/favicon.png"`)
	}
	return []byte(html)
}

func normalizeBaseURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "http://localhost:8080"
	}
	return strings.TrimRight(value, "/")
}

func basePath(baseURL string) string {
	parsed, err := url.Parse(normalizeBaseURL(baseURL))
	if err != nil || parsed.Path == "" {
		return "/"
	}
	path := strings.TrimRight(parsed.Path, "/")
	if path == "" {
		return "/"
	}
	return path
}
