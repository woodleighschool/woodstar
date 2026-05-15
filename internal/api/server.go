// Package api provides Woodstar's HTTP server and route composition.
package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/woodleighschool/woodstar/internal/agents/checks"
	"github.com/woodleighschool/woodstar/internal/agents/livequery"
	"github.com/woodleighschool/woodstar/internal/agents/orbit"
	"github.com/woodleighschool/woodstar/internal/agents/osquery"
	"github.com/woodleighschool/woodstar/internal/agents/protocol"
	"github.com/woodleighschool/woodstar/internal/agents/queries"
	"github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/secrets"
	"github.com/woodleighschool/woodstar/internal/software"
	"github.com/woodleighschool/woodstar/internal/users"
	"github.com/woodleighschool/woodstar/internal/web"
)

// Dependencies contains runtime dependencies for [Server].
type Dependencies struct {
	Config           config.Config
	DB               *database.DB
	Version          string
	Logger           *slog.Logger
	WebHandler       *web.Handler
	AuthService      *auth.Service
	UserService      *users.Service
	SessionManager   *scs.SessionManager
	HostStore        *hosts.Store
	DeviceMappings   *hosts.DeviceMappingStore
	SecretStore      *secrets.Store
	SoftwareStore    *software.Store
	LabelStore       *labels.Store
	QueryStore       *queries.Store
	CheckStore       *checks.Store
	LiveQueryManager *livequery.Manager
	OrbitService     *orbit.Service
	OsqueryService   *osquery.Service
}

// Server owns the HTTP listener and router.
type Server struct {
	httpServer *http.Server
	deps       Dependencies
}

// NewServer returns an HTTP server.
func NewServer(deps Dependencies) *Server {
	installHumaErrorHandler(deps.Logger)

	server := &Server{deps: deps}
	server.httpServer = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", deps.Config.Host, deps.Config.Port),
		Handler:           server.routes(),
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       180 * time.Second,
	}
	return server
}

// ListenAndServe starts the HTTP listener and blocks until shutdown or failure.
func (s *Server) ListenAndServe() error {
	s.deps.Logger.Info(
		"starting woodstar",
		"component", "server",
		"operation", "start",
		"addr", s.httpServer.Addr,
		"public_url", s.deps.Config.PublicURL,
		"version", s.deps.Version,
	)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.deps.Logger.InfoContext(ctx, "stopping woodstar", "component", "server", "operation", "shutdown")
	return s.httpServer.Shutdown(ctx)
}

// Config returns the runtime configuration used by the server.
func (s *Server) Config() config.Config {
	return s.deps.Config
}

func (s *Server) routes() http.Handler {
	deps := s.deps
	r := chi.NewRouter()
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Timeout(120 * time.Second))

	r.Get("/api/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("alive\n"))
	})
	r.Get("/api/readyz", func(w http.ResponseWriter, req *http.Request) {
		if err := deps.DB.Ping(req.Context()); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ready\n"))
	})

	r.Group(s.agentRoutes)
	r.Group(s.browserRoutes)

	return r
}

func (s *Server) agentRoutes(r chi.Router) {
	deps := s.deps
	r.Use(middleware.RequestLogger(deps.Logger, slog.LevelDebug))
	protocol.RegisterOrbitRoutes(r, deps.OrbitService, deps.Logger.With("component", "orbit"))
	protocol.RegisterOsqueryRoutes(r, deps.OsqueryService, deps.Logger.With("component", "osquery"))
}

func (s *Server) browserRoutes(r chi.Router) {
	deps := s.deps
	r.Use(middleware.RequestLogger(deps.Logger, slog.LevelDebug))
	if deps.SessionManager != nil {
		r.Use(deps.SessionManager.LoadAndSave)
	}
	r.Use(middleware.CSRF(deps.Config, config.SessionLifetime))
	Mount(r, deps)
	if deps.WebHandler != nil {
		deps.WebHandler.RegisterRoutes(r)
	}
}
