// Package transport provides Woodstar's HTTP server and route composition.
package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/models"
	"github.com/woodleighschool/woodstar/internal/orbit"
	"github.com/woodleighschool/woodstar/internal/osquery"
	"github.com/woodleighschool/woodstar/internal/transport/admin"
	transportorbit "github.com/woodleighschool/woodstar/internal/transport/orbit"
	transportosquery "github.com/woodleighschool/woodstar/internal/transport/osquery"
	"github.com/woodleighschool/woodstar/internal/web"
)

// SessionLifetime is the browser session lifetime.
const SessionLifetime = 14 * 24 * time.Hour

// Dependencies contains runtime dependencies for [Server].
type Dependencies struct {
	Config         config.Config
	DB             *database.DB
	Version        string
	WebHandler     *web.Handler
	AuthService    *auth.Service
	SessionManager *scs.SessionManager
	HostStore      *models.HostStore
	DeviceMappings *models.DeviceMappingStore
	SecretStore    *models.SecretStore
	SoftwareStore  *models.SoftwareStore
	OrbitService   *orbit.Service
	OsqueryService *osquery.Service
}

// Server owns the HTTP listener and router.
type Server struct {
	httpServer     *http.Server
	config         config.Config
	db             *database.DB
	version        string
	webHandler     *web.Handler
	authService    *auth.Service
	sessionManager *scs.SessionManager
	hostStore      *models.HostStore
	deviceMappings *models.DeviceMappingStore
	secretStore    *models.SecretStore
	softwareStore  *models.SoftwareStore
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
		webHandler:     deps.WebHandler,
		authService:    deps.AuthService,
		sessionManager: deps.SessionManager,
		hostStore:      deps.HostStore,
		deviceMappings: deps.DeviceMappings,
		secretStore:    deps.SecretStore,
		softwareStore:  deps.SoftwareStore,
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

	log.Info().Str("addr", addr).Str("public_url", s.config.PublicURL).Msg("starting woodstar")
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("stopping woodstar")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(120 * time.Second))

	if s.sessionManager != nil {
		r.Use(s.sessionManager.LoadAndSave)
	}

	transportorbit.RegisterRoutes(r, s.orbitService)
	transportosquery.RegisterRoutes(r, s.osqueryService)

	r.Group(func(browser chi.Router) {
		if !s.config.IsHTTPS() {
			browser.Use(admin.PlaintextHTTP)
		}
		browser.Use(admin.CSRF(s.config, SessionLifetime))
		admin.Mount(browser, admin.Dependencies{
			DB:             s.db,
			Version:        s.version,
			Started:        s.started,
			AuthService:    s.authService,
			HostStore:      s.hostStore,
			DeviceMappings: s.deviceMappings,
			SecretStore:    s.secretStore,
			SoftwareStore:  s.softwareStore,
		})
		if s.webHandler != nil {
			s.webHandler.RegisterRoutes(browser)
		}
	})

	return r
}
