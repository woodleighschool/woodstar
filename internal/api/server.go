// Package api wires Woodstar's HTTP server and app API surface.
package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/CAFxX/httpcompression"
	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/api/handlers"
	"github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/inventory"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	mdpprotocol "github.com/woodleighschool/woodstar/internal/munki/mdp/protocol"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkiprotocol "github.com/woodleighschool/woodstar/internal/munki/protocol"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/orbit"
	orbitprotocol "github.com/woodleighschool/woodstar/internal/orbit/protocol"
	"github.com/woodleighschool/woodstar/internal/osquery"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	osqueryprotocol "github.com/woodleighschool/woodstar/internal/osquery/protocol"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
	santaprotocol "github.com/woodleighschool/woodstar/internal/santa/protocol"
	"github.com/woodleighschool/woodstar/internal/santa/references"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/webui"
)

func init() {
	//nolint:reassign // huma exposes array nullability only as a package global
	huma.DefaultArrayNullable = false
}

// Server owns the HTTP listener and router.
type Server struct {
	httpServer        *http.Server
	config            config.Config
	logger            *slog.Logger
	version           string
	orbit             *orbitprotocol.Server
	osquery           *osqueryprotocol.Server
	munki             *munkiprotocol.Server
	munkiDistribution *mdpprotocol.Server
	santa             *santaprotocol.Server
}

// Dependencies is everything the HTTP server needs. Package main constructs
// stores and services; package api owns how they become routes.
type Dependencies struct {
	Config         config.Config
	DB             *database.DB
	Version        string
	Logger         *slog.Logger
	WebHandler     *webui.Handler
	SessionManager *scs.SessionManager

	App       AppDependencies
	Protocols ProtocolDependencies
}

// AppDependencies are stores and services used by Woodstar's browser/admin API.
type AppDependencies struct {
	AuthService *auth.Service
	Users       *directory.UserService
	Directory   *directory.Store
	Hosts       *hosts.Store
	PrimaryUser *hosts.PrimaryUserStore
	Secrets     *agentauth.Store
	Software    *inventory.Store
	Labels      *labels.Store

	Reports     *reports.Store
	Checks      *checks.Store
	LiveQueries *livequery.Manager

	StorageBackend storage.Backend
	StorageKey     []byte
	StorageObjects *storage.ObjectStore

	MunkiPackages     *packages.Store
	MunkiSoftware     *munkisoftware.Store
	MunkiHostState    *munki.Store
	MunkiDistribution *mdp.Store

	SantaConfigurations *configurations.Store
	SantaEvents         *events.Store
	SantaRules          *rules.Store
	SantaReferences     *references.Store
	SantaState          *santa.HostStateService
}

// ProtocolDependencies are services mounted as agent-facing wire protocols on
// the same HTTP server as the app API.
type ProtocolDependencies struct {
	AgentAuth agentauth.SecretVerifier
	Orbit     *orbit.EnrollmentService
	Osquery   *osquery.AgentService
	Munki     MunkiProtocolDependencies
	Santa     *santa.SyncService
}

// MunkiProtocolDependencies are the services backing Munki client and
// distribution-point protocols.
type MunkiProtocolDependencies struct {
	Repository   *munki.RepositoryService
	Distribution *mdp.Store
	Storage      storage.Backend
}

// NewServer returns an HTTP server.
func NewServer(deps *Dependencies) *Server {
	server := &Server{
		config:  deps.Config,
		logger:  deps.Logger.With("component", "server"),
		version: deps.Version,
		orbit: orbitprotocol.NewServer(
			deps.Protocols.Orbit,
			deps.Logger.With("component", "orbit"),
		),
		osquery: osqueryprotocol.NewServer(
			deps.Protocols.Osquery,
			deps.Logger.With("component", "osquery"),
		),
		munki: munkiprotocol.NewServer(
			deps.Protocols.AgentAuth,
			deps.Protocols.Munki.Repository,
			deps.Protocols.Munki.Distribution,
			deps.Protocols.Munki.Storage,
			deps.Logger.With("component", "munki"),
		),
		munkiDistribution: mdpprotocol.NewServer(
			deps.Protocols.Munki.Distribution,
			deps.Protocols.Munki.Storage,
			deps.Logger.With("component", "munki_distribution"),
		),
		santa: santaprotocol.NewServer(
			deps.Protocols.AgentAuth,
			deps.Protocols.Santa,
			deps.Logger.With("component", "santa"),
		),
	}
	server.httpServer = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", deps.Config.Host, deps.Config.Port),
		Handler:           server.routes(deps),
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       180 * time.Second,
	}
	return server
}

// Addr returns the configured HTTP listen address.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// Serve starts the HTTP server on listener and blocks until shutdown or failure.
func (s *Server) Serve(listener net.Listener) error {
	defer s.munkiDistribution.Close()
	s.logger.Info(
		"starting woodstar",
		"operation", "start",
		"addr", s.httpServer.Addr,
		"public_url", s.config.PublicURL,
		"version", s.version,
	)
	if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.InfoContext(ctx, "stopping woodstar", "operation", "shutdown")
	s.munkiDistribution.Close()
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) routes(deps *Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(clientIPMiddleware(deps.Config))
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Timeout(120 * time.Second))
	r.Use(compressionMiddleware(deps.Logger))
	r.Use(corsMiddleware(deps.Config))

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

	r.Group(func(r chi.Router) {
		s.protocolRoutes(r, deps)
	})
	r.Group(func(r chi.Router) {
		s.browserRoutes(r, deps)
	})

	return r
}

// protocolRoutes mounts every agent-facing protocol endpoint. They are wire
// protocols that live beside the app API on the same HTTP server.
func (s *Server) protocolRoutes(r chi.Router, deps *Dependencies) {
	r.Use(middleware.RequestLogger(deps.Logger))
	s.orbit.RegisterRoutes(r)
	s.osquery.RegisterRoutes(r)
	s.munki.RegisterRoutes(r)
	s.munkiDistribution.RegisterRoutes(r)
	s.santa.RegisterRoutes(r)
}

func (s *Server) browserRoutes(r chi.Router, deps *Dependencies) {
	r.Use(middleware.RequestLogger(deps.Logger))
	r.Group(func(r chi.Router) {
		handlers.RegisterBlobStorage(
			r,
			deps.App.StorageBackend,
			deps.App.StorageKey,
			deps.Logger.With("component", "storage"),
		)
	})
	r.Group(func(r chi.Router) {
		r.Use(deps.SessionManager.LoadAndSave)
		r.Use(middleware.CrossOriginProtection(deps.Config.CORSAllowedOrigins))
		s.mount(r, deps)
		deps.WebHandler.RegisterRoutes(r)
	})
}

func (s *Server) mount(r chi.Router, deps *Dependencies) {
	humaAPI := humachi.New(r, humaConfig(deps.Version))
	registerAppRoutes(r, humaAPI, deps, s.munkiDistribution.RefreshDesiredPackages)
}

func registerAppRoutes(
	r chi.Router,
	humaAPI huma.API,
	deps *Dependencies,
	refreshMunkiDistribution func(),
) {
	session := huma.NewGroup(humaAPI)
	session.UseMiddleware(middleware.OptionalHumaAuth(humaAPI, deps.App.AuthService))

	protected := huma.NewGroup(humaAPI)
	protected.UseMiddleware(middleware.RequireHumaAuth(humaAPI, deps.App.AuthService))

	// Authn is request-time middleware; admin posture is an operation modifier
	// so generated schemas advertise 403 on the routes that can return it.
	ordinary := huma.NewGroup(protected)
	ordinary.UseModifier(middleware.RequireAdminForMutations(humaAPI))
	sensitive := huma.NewGroup(protected)
	sensitive.UseModifier(middleware.RequireAdminForAll(humaAPI))

	apiLogger := deps.Logger.With("component", "api")
	munkiPackages := munki.NewPackageService(munki.PackageServiceDependencies{
		Packages:               deps.App.MunkiPackages,
		DesiredPackagesChanged: refreshMunkiDistribution,
	})

	// Create handlers
	handlers.RegisterAuth(humaAPI, session, protected, r, deps.App.AuthService, deps.App.Users, apiLogger)
	handlers.RegisterDirectory(ordinary, deps.App.Users, deps.App.Directory, apiLogger)
	handlers.RegisterHosts(ordinary, deps.App.Hosts, deps.App.PrimaryUser, apiLogger)
	handlers.RegisterInventory(ordinary, deps.App.Software, deps.App.Hosts, apiLogger)
	handlers.RegisterLabels(ordinary, deps.App.Labels, apiLogger)
	handlers.RegisterAgentAuth(sensitive, deps.App.Secrets, apiLogger)
	handlers.RegisterOsquery(
		ordinary,
		sensitive,
		deps.App.Reports,
		deps.App.Checks,
		deps.App.LiveQueries,
		deps.App.Hosts,
		apiLogger,
	)
	handlers.RegisterMunki(
		ordinary,
		r,
		deps.App.AuthService,
		deps.App.MunkiHostState,
		deps.App.Hosts,
		deps.App.MunkiSoftware,
		munkiPackages,
		deps.App.StorageObjects,
		deps.App.StorageBackend,
		deps.App.MunkiDistribution,
		apiLogger,
	)
	handlers.RegisterSanta(
		ordinary,
		deps.App.SantaState,
		deps.App.Hosts,
		deps.App.SantaConfigurations,
		deps.App.SantaRules,
		deps.App.SantaEvents,
		deps.App.SantaReferences,
		apiLogger,
	)
}

// humaConfig returns the Huma config shared by serve and openapi.
func humaConfig(version string) huma.Config {
	cfg := huma.DefaultConfig("Woodstar API", version)
	cfg.Info.Description = "Typed admin and frontend API."
	cfg.Info.License = &huma.License{Name: "Apache-2.0"}

	// Don't emit docs or schema routes, useless for us.
	cfg.OpenAPIPath = ""
	cfg.DocsPath = ""
	cfg.SchemasPath = ""
	cfg.CreateHooks = nil

	cfg.Components = &huma.Components{
		Schemas: huma.NewMapRegistry("#/components/schemas/", schemaNamer),
		SecuritySchemes: map[string]*huma.SecurityScheme{
			"cookieAuth": {
				Type: "apiKey",
				In:   "cookie",
				Name: "woodstar_session",
			},
		},
	}

	return cfg
}

// BuildSchemaAPI builds the app API for OpenAPI schema generation only. Route
// registration binds handlers but never invokes them, so nil stores and
// services are valid here.
func BuildSchemaAPI(version string, deps *Dependencies) huma.API {
	r := chi.NewRouter()
	humaAPI := humachi.New(r, humaConfig(version))
	deps.Version = version
	registerAppRoutes(r, humaAPI, deps, func() {})
	return humaAPI
}

// clientIPMiddleware maps the configured client-IP source to its chi middleware.
// config owns the source enum and its validation; api owns this switch so
// config never imports chi. The default trusts the connection's remote address.
func clientIPMiddleware(cfg config.Config) func(http.Handler) http.Handler {
	switch cfg.ClientIPSource {
	case config.ClientIPSourceHeader:
		return chimiddleware.ClientIPFromHeader(cfg.ClientIPHeader)
	case config.ClientIPSourceXFFTrustedCIDRs:
		return chimiddleware.ClientIPFromXFF(cfg.ClientIPTrustedCIDRs...)
	case config.ClientIPSourceXFFTrustedProxies:
		return chimiddleware.ClientIPFromXFFTrustedProxies(cfg.ClientIPTrustedProxies)
	default:
		return chimiddleware.ClientIPFromRemoteAddr
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
	return func(next http.Handler) http.Handler {
		compressed := compressor(next)
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.Contains(req.Header.Get("Accept"), "text/event-stream") ||
				strings.HasPrefix(req.URL.Path, "/storage/") ||
				req.Header.Get("Range") != "" {
				next.ServeHTTP(w, req)
				return
			}
			compressed.ServeHTTP(w, req)
		})
	}
}

func corsMiddleware(cfg config.Config) func(http.Handler) http.Handler {
	if len(cfg.CORSAllowedOrigins) == 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return cors.New(cors.Options{
		AllowCredentials: true,
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodOptions,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "Range", "X-API-Key", "X-Requested-With"},
		ExposedHeaders: []string{"Accept-Ranges", "Content-Length", "Content-Range", "Content-Type"},
		MaxAge:         300,
	}).Handler
}
