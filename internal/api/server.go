// Package api wires Woodstar's HTTP server and app API surface.
package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	pathpkg "path"
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
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	mdpprotocol "github.com/woodleighschool/woodstar/internal/munki/mdp/protocol"
	munkiupload "github.com/woodleighschool/woodstar/internal/munki/objectupload"
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

// Server owns the listener and router.
type Server struct {
	httpServer        *http.Server
	config            config.Config
	logger            *slog.Logger
	version           string
	orbit             *orbitprotocol.Server
	osquery           *osqueryprotocol.Server
	munki             *munkiprotocol.Server
	munkiPackages     *munki.PackageService
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
	PrimaryUser *hosts.PrimaryUserService
	Secrets     *agentauth.Store
	Software    *inventory.Store
	Labels      *labels.Store

	Reports     *reports.Store
	Checks      *checks.Store
	LiveQueries *livequery.Manager

	StorageBackend storage.Backend
	StorageKey     []byte
	StorageObjects *storage.ObjectStore
	MunkiUploads   *munkiupload.Service

	MunkiPackages          *munki.PackageService
	MunkiClientResources   *clientresources.Service
	MunkiSoftware          *munkisoftware.Store
	MunkiSoftwareDeletions *munki.SoftwareDeletionService
	MunkiHostState         *munki.Store
	MunkiDistribution      *mdp.Store

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
	Repository           *munki.RepositoryService
	Distribution         *mdp.Store
	DistributionProtocol *mdpprotocol.Server
	Storage              storage.Backend
}

// NewServer returns an HTTP server.
func NewServer(deps *Dependencies) (*Server, error) {
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
		munkiPackages:     deps.App.MunkiPackages,
		munkiDistribution: deps.Protocols.Munki.DistributionProtocol,
		santa: santaprotocol.NewServer(
			deps.Protocols.AgentAuth,
			deps.Protocols.Santa,
			deps.Logger.With("component", "santa"),
		),
	}
	handler, err := server.routes(deps)
	if err != nil {
		return nil, err
	}
	server.httpServer = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", deps.Config.Host, deps.Config.Port),
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       180 * time.Second,
	}
	return server, nil
}

// Addr returns the configured HTTP listen address.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// Serve starts the server on listener and blocks until shutdown or failure.
func (s *Server) Serve(listener net.Listener) error {
	defer s.munkiDistribution.Close()
	transport := "http"
	serve := func() error { return s.httpServer.Serve(listener) }
	if s.config.TLSConfigured() {
		transport = "https"
		serve = func() error {
			return s.httpServer.ServeTLS(listener, s.config.TLSCertFile, s.config.TLSKeyFile)
		}
	}
	s.logger.Info(
		"starting woodstar",
		"operation", "start",
		"addr", s.httpServer.Addr,
		"server_url", s.config.ServerURL,
		"transport", transport,
		"version", s.version,
	)
	if err := serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

func (s *Server) routes(deps *Dependencies) (http.Handler, error) {
	compression, err := compressionMiddleware()
	if err != nil {
		return nil, fmt.Errorf("create HTTP compression adapter: %w", err)
	}

	r := chi.NewRouter()
	r.Use(clientIPMiddleware(deps.Config))
	r.Use(chimiddleware.RequestID)
	r.Use(requestTimeoutMiddleware(120 * time.Second))
	r.Use(compression)
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

	return r, nil
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
	registerAppRoutes(r, humaAPI, deps, s.munkiPackages)
}

func registerAppRoutes(
	r chi.Router,
	humaAPI huma.API,
	deps *Dependencies,
	munkiPackages *munki.PackageService,
) {
	session := huma.NewGroup(humaAPI)
	session.UseMiddleware(middleware.OptionalHumaAuth(humaAPI, deps.App.AuthService))

	protected := huma.NewGroup(humaAPI)
	protected.UseMiddleware(middleware.RequireHumaAuth(humaAPI, deps.App.AuthService))
	protected.UseModifier(middleware.ProtectedOperation(humaAPI))

	// Authn is request-time middleware; admin posture is an operation modifier
	// so generated schemas advertise 403 on the routes that can return it.
	ordinary := huma.NewGroup(protected)
	ordinary.UseModifier(middleware.RequireAdminForMutations(humaAPI))
	sensitive := huma.NewGroup(protected)
	sensitive.UseModifier(middleware.RequireAdminForAll(humaAPI))

	apiLogger := deps.Logger.With("component", "api")

	// Create handlers
	handlers.RegisterAuth(handlers.AuthHandlerDeps{
		Public:      humaAPI,
		Session:     session,
		Protected:   protected,
		Router:      r,
		AuthService: deps.App.AuthService,
		Users:       deps.App.Users,
		Logger:      apiLogger,
	})
	handlers.RegisterDirectory(ordinary, deps.App.Users, deps.App.Directory, apiLogger)
	handlers.RegisterHosts(ordinary, deps.App.Hosts, deps.App.PrimaryUser, apiLogger)
	handlers.RegisterInventory(ordinary, deps.App.Software, apiLogger)
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
	handlers.RegisterMunki(handlers.MunkiHandlerDeps{
		API:             ordinary,
		Router:          r,
		AuthService:     deps.App.AuthService,
		HostState:       deps.App.MunkiHostState,
		Software:        deps.App.MunkiSoftware,
		DeleteSoftware:  deps.App.MunkiSoftwareDeletions,
		Packages:        munkiPackages,
		ClientResources: deps.App.MunkiClientResources,
		Objects:         deps.App.StorageObjects,
		Uploads:         deps.App.MunkiUploads,
		Storage:         deps.App.StorageBackend,
		Distribution:    deps.App.MunkiDistribution,
		Connections:     deps.Protocols.Munki.DistributionProtocol,
		Logger:          apiLogger,
	})
	handlers.RegisterSanta(
		ordinary,
		deps.App.SantaState,
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
			"bearerAuth": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "API key",
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
	registerAppRoutes(r, humaAPI, deps, nil)
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

func compressionMiddleware() (func(http.Handler) http.Handler, error) {
	compressor, err := httpcompression.DefaultAdapter(
		httpcompression.MinSize(1024),
		httpcompression.GzipCompressionLevel(2),
		httpcompression.Prefer(httpcompression.PreferServer),
	)
	if err != nil {
		return nil, err
	}
	return func(next http.Handler) http.Handler {
		compressed := compressor(next)
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if isStorageTransfer(req) || isLiveQueryStream(req) || isDistributionWebSocket(req) {
				next.ServeHTTP(w, req)
				return
			}
			compressed.ServeHTTP(w, req)
		})
	}, nil
}

func requestTimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	withTimeout := chimiddleware.Timeout(timeout)
	withPackageInstallerTimeout := chimiddleware.Timeout(packageInstallerTimeout)
	return func(next http.Handler) http.Handler {
		timed := withTimeout(next)
		packageInstallerTimed := withPackageInstallerTimeout(next)
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if isStorageTransfer(req) || isLiveQueryStream(req) || isDistributionWebSocket(req) {
				next.ServeHTTP(w, req)
				return
			}
			requestTimeout := timeout
			requestHandler := timed
			if isPackageInstallerMutation(req) {
				requestTimeout = packageInstallerTimeout
				requestHandler = packageInstallerTimed
			}
			deadline := time.Now().Add(requestTimeout)
			controller := http.NewResponseController(w)
			_ = controller.SetReadDeadline(deadline)
			_ = controller.SetWriteDeadline(deadline)
			requestHandler.ServeHTTP(w, req)
		})
	}
}

const packageInstallerTimeout = time.Hour

func isStorageTransfer(req *http.Request) bool {
	return strings.HasPrefix(req.URL.Path, "/storage/")
}

func isLiveQueryStream(req *http.Request) bool {
	matched, _ := pathpkg.Match("/api/live-queries/*/stream", req.URL.Path)
	return req.Method == http.MethodGet && matched
}

func isDistributionWebSocket(req *http.Request) bool {
	return req.Method == http.MethodGet &&
		req.URL.Path == "/api/munki/distribution/connect" &&
		strings.EqualFold(req.Header.Get("Upgrade"), "websocket")
}

func isPackageInstallerMutation(req *http.Request) bool {
	if req.Method == http.MethodPut {
		matched, _ := pathpkg.Match("/api/munki/package-installers/*", req.URL.Path)
		return matched
	}
	if req.Method == http.MethodPost {
		matched, _ := pathpkg.Match("/api/munki/package-installers/*/multipart/complete", req.URL.Path)
		return matched
	}
	return false
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
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "Range", "X-Requested-With"},
		ExposedHeaders: []string{"Accept-Ranges", "Content-Length", "Content-Range", "Content-Type"},
		MaxAge:         300,
	}).Handler
}
