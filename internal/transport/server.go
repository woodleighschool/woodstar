// Package transport provides Woodstar's HTTP server and route composition.
package transport

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

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/db"
	"github.com/woodleighschool/woodstar/internal/models"
	"github.com/woodleighschool/woodstar/internal/orbit"
	"github.com/woodleighschool/woodstar/internal/osquery"
	queryinfra "github.com/woodleighschool/woodstar/internal/queries"
	"github.com/woodleighschool/woodstar/internal/transport/admin"
	transportorbit "github.com/woodleighschool/woodstar/internal/transport/orbit"
	transportosquery "github.com/woodleighschool/woodstar/internal/transport/osquery"
	"github.com/woodleighschool/woodstar/internal/web"
)

// SessionLifetime is the browser session lifetime.
const SessionLifetime = 14 * 24 * time.Hour

// Dependencies contains runtime dependencies for [Server].
type Dependencies struct {
	Config           config.Config
	DB               *db.DB
	Version          string
	Logger           *slog.Logger
	WebHandler       *web.Handler
	AuthService      *auth.Service
	SessionManager   *scs.SessionManager
	HostStore        *models.HostStore
	DeviceMappings   *models.DeviceMappingStore
	SecretStore      *models.SecretStore
	SoftwareStore    *models.SoftwareStore
	LabelStore       *models.LabelStore
	QueryStore       *models.QueryStore
	CheckStore       *models.CheckStore
	LiveQueryManager *queryinfra.LiveQueryManager
	OrbitService     *orbit.Service
	OsqueryService   *osquery.Service
}

// Server owns the HTTP listener and router.
type Server struct {
	httpServer     *http.Server
	config         config.Config
	db             *db.DB
	version        string
	logger         *slog.Logger
	webHandler     *web.Handler
	authService    *auth.Service
	sessionManager *scs.SessionManager
	hostStore      *models.HostStore
	deviceMappings *models.DeviceMappingStore
	secretStore    *models.SecretStore
	softwareStore  *models.SoftwareStore
	labelStore     *models.LabelStore
	queryStore     *models.QueryStore
	checkStore     *models.CheckStore
	liveQueries    *queryinfra.LiveQueryManager
	orbitService   *orbit.Service
	osqueryService *osquery.Service
	started        time.Time
}

// NewServer returns an HTTP server.
func NewServer(deps Dependencies) *Server {
	return &Server{
		httpServer: &http.Server{
			ReadHeaderTimeout: 15 * time.Second,
			ReadTimeout:       60 * time.Second,
			WriteTimeout:      120 * time.Second,
			IdleTimeout:       180 * time.Second,
		},
		config:         deps.Config,
		db:             deps.DB,
		version:        deps.Version,
		logger:         deps.Logger,
		webHandler:     deps.WebHandler,
		authService:    deps.AuthService,
		sessionManager: deps.SessionManager,
		hostStore:      deps.HostStore,
		deviceMappings: deps.DeviceMappings,
		secretStore:    deps.SecretStore,
		softwareStore:  deps.SoftwareStore,
		labelStore:     deps.LabelStore,
		queryStore:     deps.QueryStore,
		checkStore:     deps.CheckStore,
		liveQueries:    deps.LiveQueryManager,
		orbitService:   deps.OrbitService,
		osqueryService: deps.OsqueryService,
		started:        time.Now().UTC(),
	}
}

// ListenAndServe starts the HTTP listener and blocks until shutdown or failure.
func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.httpServer.Addr = addr
	s.httpServer.Handler = s.routes()

	s.logger.Info(
		"starting woodstar",
		"component", "server",
		"operation", "start",
		"addr", addr,
		"public_url", s.config.PublicURL,
		"version", s.version,
	)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.InfoContext(ctx, "stopping woodstar", "component", "server", "operation", "shutdown")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Timeout(120 * time.Second))

	r.Group(func(agent chi.Router) {
		agent.Use(requestLogger(s.logger, slog.LevelDebug))
		transportorbit.RegisterRoutes(agent, s.orbitService, s.logger.With("component", "orbit"))
		transportosquery.RegisterRoutes(agent, s.osqueryService, s.logger.With("component", "osquery"))
	})

	r.Group(func(browser chi.Router) {
		browser.Use(requestLogger(s.logger, slog.LevelDebug))
		if s.sessionManager != nil {
			browser.Use(s.sessionManager.LoadAndSave)
		}
		if !s.config.IsHTTPS() {
			browser.Use(admin.PlaintextHTTP)
		}
		browser.Use(admin.CSRF(s.config, SessionLifetime))
		admin.Mount(browser, admin.Dependencies{
			DB:               s.db,
			Version:          s.version,
			Started:          s.started,
			AuthService:      s.authService,
			HostStore:        s.hostStore,
			DeviceMappings:   s.deviceMappings,
			SecretStore:      s.secretStore,
			SoftwareStore:    s.softwareStore,
			LabelStore:       s.labelStore,
			QueryStore:       s.queryStore,
			CheckStore:       s.checkStore,
			LiveQueryManager: s.liveQueries,
		})
		if s.webHandler != nil {
			s.webHandler.RegisterRoutes(browser)
		}
	})

	return r
}
