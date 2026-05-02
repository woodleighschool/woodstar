package web

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Options struct {
	FS fs.FS
}

type Handler struct {
	fs fs.FS
}

func init() {
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".css", "text/css")
	mime.AddExtensionType(".html", "text/html")
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".svg", "image/svg+xml")
}

func NewHandler(opts Options) *Handler {
	return &Handler{fs: opts.FS}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/assets/*", h.serveAsset)
	r.Get("/favicon.ico", h.serveAsset)
	r.Get("/favicon.png", h.serveAsset)
	r.Get("/", h.placeholder)
	r.Get("/*", h.placeholder)
}

func (h *Handler) serveAsset(w http.ResponseWriter, r *http.Request) {
	if h.fs == nil {
		http.NotFound(w, r)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
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

func (h *Handler) placeholder(w http.ResponseWriter, r *http.Request) {
	if h.fs != nil {
		file, err := h.fs.Open("index.html")
		if err == nil {
			defer file.Close()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.Copy(w, file)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>Woodstar</title></head><body><div id="root"></div></body></html>`))
}
