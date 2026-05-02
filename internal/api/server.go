package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"

	"github.com/woodleighschool/woodstar/internal/api/handlers"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/web"
)

type Dependencies struct {
	Config     config.Config
	DB         *database.DB
	Version    string
	WebHandler *web.Handler
}

type Server struct {
	httpServer *http.Server
	config     config.Config
	db         *database.DB
	version    string
	webHandler *web.Handler
	started    time.Time
}

func NewServer(deps Dependencies) *Server {
	return &Server{
		httpServer: &http.Server{
			ReadHeaderTimeout: 15 * time.Second,
			ReadTimeout:       60 * time.Second,
			WriteTimeout:      120 * time.Second,
			IdleTimeout:       180 * time.Second,
		},
		config:     deps.Config,
		db:         deps.DB,
		version:    deps.Version,
		webHandler: deps.WebHandler,
		started:    time.Now().UTC(),
	}
}

func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.httpServer.Addr = addr
	s.httpServer.Handler = s.routes()

	log.Info().Str("addr", addr).Msg("starting woodstar")
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("stopping woodstar")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(120 * time.Second))

	r.Route("/api", func(r chi.Router) {
		health := handlers.NewHealthHandler(s.db)
		version := handlers.NewVersionHandler(s.version, s.started)

		r.Get("/healthz", health.Liveness)
		r.Get("/readyz", health.Readiness)
		r.Get("/version", version.Current)
	})

	if s.webHandler != nil {
		s.webHandler.RegisterRoutes(r)
	}

	return r
}
