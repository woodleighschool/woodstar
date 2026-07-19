// Package api wires Woodstar's HTTP server and app API surface.
package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
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

// Huma exposes array nullability only through a package global.
//
//nolint:gochecknoinits // Set it before any schema registry can be built.
func init() {
	huma.DefaultArrayNullable = false //nolint:reassign // No per-API setting exists.
}

// Server owns the listener and router.
type Server struct {
	httpServer *http.Server
	deps       *Dependencies
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

	StorageBackend  storage.Backend
	StorageDelivery *storage.Delivery
	StorageKey      []byte
	StorageObjects  *storage.ObjectStore
	StorageIngestor *storage.Ingestor

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
	Delivery             storage.Deliverer
}

// NewServer returns an HTTP server.
func NewServer(deps *Dependencies) (*Server, error) {
	handler, err := routes(deps)
	if err != nil {
		return nil, err
	}
	return &Server{
		deps: deps,
		httpServer: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", deps.Config.Host, deps.Config.Port),
			Handler:           handler,
			ReadHeaderTimeout: 15 * time.Second,
			IdleTimeout:       180 * time.Second,
		},
	}, nil
}

// Addr returns the configured HTTP listen address.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// Serve starts the server on listener and blocks until shutdown or failure.
func (s *Server) Serve(listener net.Listener) error {
	defer s.deps.Protocols.Munki.DistributionProtocol.Close()
	cfg := s.deps.Config
	transport := "http"
	serve := func() error { return s.httpServer.Serve(listener) }
	if cfg.TLSConfigured() {
		transport = "https"
		serve = func() error {
			return s.httpServer.ServeTLS(listener, cfg.TLSCertFile, cfg.TLSKeyFile)
		}
	}
	s.deps.Logger.With("component", "server").Info(
		"starting woodstar",
		"operation", "start",
		"addr", s.httpServer.Addr,
		"server_url", cfg.ServerURL,
		"transport", transport,
		"version", s.deps.Version,
	)
	if err := serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.deps.Logger.With("component", "server").InfoContext(
		ctx,
		"stopping woodstar",
		"operation", "shutdown",
	)
	s.deps.Protocols.Munki.DistributionProtocol.Close()
	return s.httpServer.Shutdown(ctx)
}

func routes(deps *Dependencies) (http.Handler, error) {
	compression, err := compressionMiddleware()
	if err != nil {
		return nil, fmt.Errorf("create HTTP compression adapter: %w", err)
	}

	r := chi.NewRouter()
	r.Use(clientIPMiddleware(deps.Config))
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.SecurityHeaders(deps.App.StorageBackend.TransferOrigin()))
	r.Use(corsMiddleware(deps.Config))

	ordinary := r.With(requestTimeoutMiddleware(defaultRequestTimeout), compression)
	ordinary.Get("/api/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("alive\n"))
	})
	ordinary.Get("/api/readyz", func(w http.ResponseWriter, req *http.Request) {
		if err := deps.DB.Ping(req.Context()); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ready\n"))
	})

	r.Group(func(r chi.Router) {
		protocolRoutes(r, compression, deps)
	})
	r.Group(func(r chi.Router) {
		browserRoutes(r, compression, deps)
	})

	return r, nil
}

// protocolRoutes mounts every agent-facing protocol endpoint. They are wire
// protocols that live beside the app API on the same HTTP server.
func protocolRoutes(
	r chi.Router,
	compression func(http.Handler) http.Handler,
	deps *Dependencies,
) {
	requestLogger := middleware.RequestLogger(deps.Logger)
	ordinary := r.With(requestTimeoutMiddleware(defaultRequestTimeout), compression, requestLogger)
	transfers := r.With(requestLogger)
	websockets := r.With(requestLogger)

	orbitprotocol.NewServer(
		deps.Protocols.Orbit,
		deps.Logger.With("component", "orbit"),
	).RegisterRoutes(ordinary)
	osqueryprotocol.NewServer(
		deps.Protocols.Osquery,
		deps.Logger.With("component", "osquery"),
	).RegisterRoutes(ordinary)
	munkiServer := munkiprotocol.NewServer(
		deps.Protocols.AgentAuth,
		deps.Protocols.Munki.Repository,
		deps.Protocols.Munki.Distribution,
		deps.Protocols.Munki.Delivery,
		deps.Logger.With("component", "munki"),
	)
	munkiServer.RegisterRoutes(ordinary, transfers)
	deps.Protocols.Munki.DistributionProtocol.RegisterRoutes(ordinary, websockets)
	santaprotocol.NewServer(
		deps.Protocols.AgentAuth,
		deps.Protocols.Santa,
		deps.Logger.With("component", "santa"),
	).RegisterRoutes(ordinary)
}

func browserRoutes(
	r chi.Router,
	compression func(http.Handler) http.Handler,
	deps *Dependencies,
) {
	requestLogger := middleware.RequestLogger(deps.Logger)
	storage.RegisterTransferRoutes(
		r.With(requestLogger),
		deps.App.StorageBackend,
		deps.App.StorageKey,
		deps.Logger.With("component", "storage"),
	)

	sessionMiddleware := deps.SessionManager.LoadAndSave
	crossOriginProtection := middleware.CrossOriginProtection(deps.Config.CORSAllowedOrigins)
	transfers := r.With(requestLogger, sessionMiddleware, crossOriginProtection)
	streaming := r.With(requestLogger, sessionMiddleware, crossOriginProtection)
	ordinary := r.With(
		requestTimeoutMiddleware(defaultRequestTimeout),
		compression,
		requestLogger,
		sessionMiddleware,
		crossOriginProtection,
	)
	longRunning := r.With(
		requestTimeoutMiddleware(longRunningRequestTimeout),
		compression,
		requestLogger,
		sessionMiddleware,
		crossOriginProtection,
	)
	mount(ordinary, transfers, streaming, longRunning, deps)
	deps.WebHandler.RegisterRoutes(ordinary)
}

type appAPIs struct {
	ordinary    huma.API
	streaming   huma.API
	longRunning huma.API
}

func newAppAPIs(
	ordinary chi.Router,
	streaming chi.Router,
	longRunning chi.Router,
	version string,
) appAPIs {
	cfg := humaConfig(version)
	return appAPIs{
		ordinary:    humachi.New(ordinary, cfg),
		streaming:   humachi.New(streaming, cfg),
		longRunning: humachi.New(longRunning, cfg),
	}
}

func mount(
	ordinary chi.Router,
	transfers chi.Router,
	streaming chi.Router,
	longRunning chi.Router,
	deps *Dependencies,
) {
	apis := newAppAPIs(ordinary, streaming, longRunning, deps.Version)
	registerAppRoutes(ordinary, transfers, apis, deps)
}

func registerAppRoutes(
	ordinaryRouter chi.Router,
	transferRouter chi.Router,
	apis appAPIs,
	deps *Dependencies,
) {
	session := huma.NewGroup(apis.ordinary)
	session.UseMiddleware(middleware.OptionalHumaAuth(apis.ordinary, deps.App.AuthService))

	protected := newProtectedGroup(apis.ordinary, deps.App.AuthService)
	ordinary := newOrdinaryGroup(protected)
	sensitive := newSensitiveGroup(protected)

	streamingSensitive := newSensitiveGroup(
		newProtectedGroup(apis.streaming, deps.App.AuthService),
	)
	longRunningOrdinary := newOrdinaryGroup(
		newProtectedGroup(apis.longRunning, deps.App.AuthService),
	)

	apiLogger := deps.Logger.With("component", "api")

	// Create handlers
	handlers.RegisterAuth(handlers.AuthHandlerDeps{
		Public:      apis.ordinary,
		Session:     session,
		Protected:   protected,
		Router:      ordinaryRouter,
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
		streamingSensitive,
		deps.App.Reports,
		deps.App.Checks,
		deps.App.LiveQueries,
		deps.App.Hosts,
		apiLogger,
	)
	handlers.RegisterMunki(handlers.MunkiHandlerDeps{
		API:             ordinary,
		LongRunningAPI:  longRunningOrdinary,
		TransferRouter:  transferRouter,
		AuthService:     deps.App.AuthService,
		HostState:       deps.App.MunkiHostState,
		Software:        deps.App.MunkiSoftware,
		DeleteSoftware:  deps.App.MunkiSoftwareDeletions,
		Packages:        deps.App.MunkiPackages,
		ClientResources: deps.App.MunkiClientResources,
		Objects:         deps.App.StorageObjects,
		Ingestor:        deps.App.StorageIngestor,
		Delivery:        deps.App.StorageDelivery,
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

func newProtectedGroup(api huma.API, authService *auth.Service) *huma.Group {
	protected := huma.NewGroup(api)
	protected.UseMiddleware(middleware.RequireHumaAuth(api, authService))
	protected.UseModifier(middleware.ProtectedOperation(api))
	return protected
}

func newOrdinaryGroup(protected huma.API) *huma.Group {
	ordinary := huma.NewGroup(protected)
	ordinary.UseModifier(middleware.RequireAdminForMutations(protected))
	return ordinary
}

func newSensitiveGroup(protected huma.API) *huma.Group {
	sensitive := huma.NewGroup(protected)
	sensitive.UseModifier(middleware.RequireAdminForAll(protected))
	return sensitive
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
func BuildSchemaAPI(version string) huma.API {
	r := chi.NewRouter()
	apis := newAppAPIs(r, r, r, version)
	registerAppRoutes(r, r, apis, &Dependencies{
		Logger: slog.New(slog.DiscardHandler),
	})
	return apis.ordinary
}

// clientIPMiddleware maps the configured client-IP source to its chi middleware.
// config owns the source enum and its validation; api owns this switch so
// config never imports chi. The default trusts the connection's remote address.
func clientIPMiddleware(cfg config.Config) func(http.Handler) http.Handler {
	switch cfg.ClientIPSource {
	case config.ClientIPSourceRemoteAddr:
		return chimiddleware.ClientIPFromRemoteAddr
	case config.ClientIPSourceHeader:
		return chimiddleware.ClientIPFromHeader(cfg.ClientIPHeader)
	case config.ClientIPSourceXFFTrustedCIDRs:
		return chimiddleware.ClientIPFromXFF(cfg.ClientIPTrustedCIDRs...)
	case config.ClientIPSourceXFFTrustedProxies:
		return chimiddleware.ClientIPFromXFFTrustedProxies(cfg.ClientIPTrustedProxies)
	default:
		panic(fmt.Sprintf("unsupported client IP source %q", cfg.ClientIPSource))
	}
}

func compressionMiddleware() (func(http.Handler) http.Handler, error) {
	return httpcompression.DefaultAdapter(
		httpcompression.MinSize(1024),
		httpcompression.GzipCompressionLevel(2),
		httpcompression.Prefer(httpcompression.PreferServer),
	)
}

func requestTimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	withTimeout := chimiddleware.Timeout(timeout)
	return func(next http.Handler) http.Handler {
		timed := withTimeout(next)
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			deadline := time.Now().Add(timeout)
			controller := http.NewResponseController(w)
			_ = controller.SetReadDeadline(deadline)
			_ = controller.SetWriteDeadline(deadline)
			timed.ServeHTTP(w, req)
		})
	}
}

const (
	defaultRequestTimeout     = 120 * time.Second
	longRunningRequestTimeout = time.Hour
)

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
