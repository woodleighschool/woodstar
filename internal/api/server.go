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
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/woodleighschool/woodstar/internal/api/handlers"
	"github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	munkiprotocol "github.com/woodleighschool/woodstar/internal/munki/protocol"
	orbitprotocol "github.com/woodleighschool/woodstar/internal/orbit/protocol"
	osqueryprotocol "github.com/woodleighschool/woodstar/internal/osquery/protocol"
	santaprotocol "github.com/woodleighschool/woodstar/internal/santa/protocol"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func init() {
	//nolint:reassign // huma exposes array nullability only as a package global
	huma.DefaultArrayNullable = false
}

// Server owns the HTTP listener and router.
type Server struct {
	httpServer *http.Server
	config     config.Config
	logger     *slog.Logger
	version    string
}

// NewServer returns an HTTP server.
func NewServer(deps handlers.Dependencies) *Server {
	server := &Server{
		config:  deps.Config,
		logger:  deps.Logger.With("component", "server"),
		version: deps.Version,
	}
	server.httpServer = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", deps.Config.Host, deps.Config.Port),
		Handler:           routes(deps),
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
	return s.httpServer.Shutdown(ctx)
}

func routes(deps handlers.Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(clientIPMiddleware(deps.Config))
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

	r.Group(func(r chi.Router) {
		protocolRoutes(r, deps)
	})
	r.Group(func(r chi.Router) {
		browserRoutes(r, deps)
	})

	return r
}

// protocolRoutes mounts every agent-facing protocol endpoint. They are wire
// protocols that live beside the app API on the same HTTP server.
func protocolRoutes(r chi.Router, deps handlers.Dependencies) {
	r.Use(middleware.RequestLogger(deps.Logger))
	storage.RegisterBlobRoutes(r, deps.StorageBackend, deps.StorageKey, deps.Logger.With("component", "storage"))
	orbitprotocol.RegisterOrbitRoutes(r, deps.OrbitAgent, deps.Logger.With("component", "orbit"))
	osqueryprotocol.RegisterOsqueryRoutes(r, deps.OsqueryAgent, deps.Logger.With("component", "osquery"))
	munkiprotocol.RegisterMunkiRoutes(
		r,
		deps.Secrets,
		deps.MunkiRepo,
		deps.MunkiDistribution,
		deps.StorageBackend,
		deps.Logger.With("component", "munki"),
	)
	mdp.RegisterProtocolRoutes(
		r,
		deps.MunkiDistributionHub,
		deps.MunkiDistribution,
		deps.StorageBackend,
		deps.Logger.With("component", "munki_distribution"),
	)
	santaprotocol.RegisterSantaRoutes(r, deps.Secrets, deps.SantaSync, deps.Logger.With("component", "santa"))
}

func browserRoutes(r chi.Router, deps handlers.Dependencies) {
	r.Use(middleware.RequestLogger(deps.Logger))
	r.Use(compressionMiddleware(deps.Logger))
	r.Use(deps.SessionManager.LoadAndSave)
	r.Use(middleware.CrossOriginProtection())
	Mount(r, deps)
	deps.WebHandler.RegisterRoutes(r)
}

// Mount attaches public and authenticated app API routes to r.
func Mount(r chi.Router, deps handlers.Dependencies) {
	humaAPI := humachi.New(r, humaConfig(deps.Version))
	registerAppRoutes(r, humaAPI, deps)
}

func registerAppRoutes(r chi.Router, humaAPI huma.API, deps handlers.Dependencies) {
	session := huma.NewGroup(humaAPI)
	session.UseMiddleware(middleware.OptionalHumaAuth(humaAPI, deps.AuthService))

	protected := huma.NewGroup(humaAPI)
	protected.UseMiddleware(middleware.RequireHumaAuth(humaAPI, deps.AuthService))

	// Authn is request-time middleware; admin posture is an operation modifier
	// so generated schemas advertise 403 on the routes that can return it.
	ordinary := huma.NewGroup(protected)
	ordinary.UseModifier(middleware.RequireAdminForMutations(humaAPI))
	sensitive := huma.NewGroup(protected)
	sensitive.UseModifier(middleware.RequireAdminForAll(humaAPI))

	groups := handlers.Groups{
		Public:    humaAPI,
		Session:   session,
		Protected: protected,
		Ordinary:  ordinary,
		Sensitive: sensitive,
		Router:    r,
	}
	handlers.Register(groups, deps)
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
func BuildSchemaAPI(version string, deps handlers.Dependencies) huma.API {
	r := chi.NewRouter()
	humaAPI := humachi.New(r, humaConfig(version))
	deps.Version = version
	registerAppRoutes(r, humaAPI, deps)
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
	return compressor
}
