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

	"github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/enrollment"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/orbit"
	orbitprotocol "github.com/woodleighschool/woodstar/internal/orbit/protocol"
	"github.com/woodleighschool/woodstar/internal/osquery"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	osqueryprotocol "github.com/woodleighschool/woodstar/internal/osquery/protocol"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/santa"
	santaprotocol "github.com/woodleighschool/woodstar/internal/santa/protocol"
	"github.com/woodleighschool/woodstar/internal/software"
	"github.com/woodleighschool/woodstar/internal/users"
	"github.com/woodleighschool/woodstar/internal/web"
)

// Dependencies contains runtime dependencies for [Server].
type Dependencies struct {
	Runtime   RuntimeDependencies
	Auth      AuthDependencies
	Inventory InventoryDependencies
	Orbit     OrbitDependencies
	Santa     SantaDependencies
}

type RuntimeDependencies struct {
	Config         config.Config
	DB             *database.DB
	Version        string
	Logger         *slog.Logger
	WebHandler     *web.Handler
	SessionManager *scs.SessionManager
}

type AuthDependencies struct {
	AuthService *auth.Service
	UserService *users.Service
}

type InventoryDependencies struct {
	HostStore     *hosts.Store
	SecretStore   *enrollment.Store
	SoftwareStore *software.Store
	LabelStore    *labels.Store
	ReportStore   *reports.Store
	CheckStore    *checks.Store
}

type OrbitDependencies struct {
	LiveQueryManager *livequery.Manager
	Service          *orbit.Service
	OsqueryService   *osquery.Service
}

type SantaDependencies struct {
	Store   *santa.Store
	Service *santa.Service
}

// Server owns the HTTP listener and router.
type Server struct {
	httpServer *http.Server
	config     config.Config
	logger     *slog.Logger
	version    string
}

// NewServer returns an HTTP server.
func NewServer(deps Dependencies) *Server {
	server := &Server{
		config:  deps.Runtime.Config,
		logger:  deps.Runtime.Logger,
		version: deps.Runtime.Version,
	}
	server.httpServer = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", deps.Runtime.Config.Host, deps.Runtime.Config.Port),
		Handler:           routes(deps),
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       180 * time.Second,
	}
	return server
}

// ListenAndServe starts the HTTP listener and blocks until shutdown or failure.
func (s *Server) ListenAndServe() error {
	s.logger.Info(
		"starting woodstar",
		"component", "server",
		"operation", "start",
		"addr", s.httpServer.Addr,
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
		orbitRoutes(r, deps)
	})
	r.Group(func(r chi.Router) {
		browserRoutes(r, deps)
	})

	return r
}

func orbitRoutes(r chi.Router, deps Dependencies) {
	r.Use(middleware.RequestLogger(deps.Runtime.Logger))
	orbitprotocol.RegisterOrbitRoutes(r, deps.Orbit.Service, deps.Runtime.Logger.With("component", "orbit"))
	osqueryprotocol.RegisterOsqueryRoutes(
		r,
		deps.Orbit.OsqueryService,
		deps.Runtime.Logger.With("component", "osquery"),
	)
	if deps.Santa.Store != nil && deps.Santa.Service != nil {
		santaprotocol.RegisterSantaRoutes(
			r,
			deps.Santa.Store,
			deps.Santa.Service,
			deps.Runtime.Logger.With("component", "santa"),
		)
	}
}

func browserRoutes(r chi.Router, deps Dependencies) {
	r.Use(middleware.RequestLogger(deps.Runtime.Logger))
	if deps.Runtime.SessionManager != nil {
		r.Use(deps.Runtime.SessionManager.LoadAndSave)
	}
	r.Use(middleware.CrossOriginProtection())
	Mount(r, deps)
	if deps.Runtime.WebHandler != nil {
		deps.Runtime.WebHandler.RegisterRoutes(r)
	}
}
