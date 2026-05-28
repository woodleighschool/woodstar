package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/CAFxX/httpcompression"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/woodleighschool/woodstar/internal/api/middleware"
)

func routes(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.ClientIPFromRemoteAddr)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Timeout(120 * time.Second))

	r.Get("/api/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("alive\n"))
	})
	r.Get("/api/readyz", func(w http.ResponseWriter, req *http.Request) {
		if err := deps.Runtime.DB.Ping(req.Context()); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ready\n"))
	})

	r.Group(func(r chi.Router) {
		protocolRoutes(r, deps)
	})
	r.Group(func(r chi.Router) {
		browserRoutes(r, deps)
	})

	return r
}

func browserRoutes(r chi.Router, deps Dependencies) {
	r.Use(middleware.RequestLogger(deps.Runtime.Logger))
	r.Use(compressionMiddleware(deps.Runtime.Logger))
	if deps.Runtime.SessionManager != nil {
		r.Use(deps.Runtime.SessionManager.LoadAndSave)
	}
	r.Use(middleware.CrossOriginProtection())
	Mount(r, deps)
	if deps.Runtime.WebHandler != nil {
		deps.Runtime.WebHandler.RegisterRoutes(r)
	}
}

func compressionMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	compressor, err := httpcompression.DefaultAdapter(
		httpcompression.MinSize(1024),
		httpcompression.GzipCompressionLevel(2),
		httpcompression.Prefer(httpcompression.PreferServer),
	)
	if err != nil {
		logger.Error("failed to create HTTP compression adapter", "operation", "compression_middleware", "err", err)
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return compressor
}
