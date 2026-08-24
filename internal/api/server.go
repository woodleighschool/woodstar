package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"

	"github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
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
	config     config.Config
	logger     *slog.Logger
	version    string
}

// ServerOptions configures shared HTTP infrastructure. Capability services
// stay in the composition root and are captured only by RegisterRoutes.
type ServerOptions struct {
	Config         config.Config
	Ready          func(context.Context) error
	Version        string
	Logger         *slog.Logger
	WebHandler     *webui.Handler
	SessionManager *scs.SessionManager
	AuthService    *auth.Service
	TransferOrigin string
	RegisterRoutes func(Routes)
}

// Routes are the server-owned HTTP surfaces available to capability
// registrars during application composition.
type Routes struct {
	App              AppRoutes
	Protocols        ProtocolRoutes
	StorageTransfers chi.Router
}

// AppRoutes groups browser/admin APIs by their shared authentication and
// timeout policy.
type AppRoutes struct {
	PasswordLogin       huma.API
	Session             huma.API
	Protected           huma.API
	Ordinary            huma.API
	Sensitive           huma.API
	StreamingSensitive  huma.API
	LongRunningOrdinary huma.API
	Router              chi.Router
	Transfers           chi.Router
}

// ProtocolRoutes groups agent-facing routers by their transport policy.
type ProtocolRoutes struct {
	Ordinary   chi.Router
	Transfers  chi.Router
	WebSockets chi.Router
}

// NewServer returns an HTTP server.
func NewServer(options ServerOptions) *Server {
	handler := routes(options)
	return &Server{
		config:  options.Config,
		logger:  options.Logger,
		version: options.Version,
		httpServer: &http.Server{
			Addr:              options.Config.Listen,
			Handler:           handler,
			ReadHeaderTimeout: 15 * time.Second,
			IdleTimeout:       180 * time.Second,
		},
	}
}

// Addr returns the configured HTTP listen address.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// Serve starts the server on listener and blocks until shutdown or failure.
func (s *Server) Serve(listener net.Listener) error {
	cfg := s.config
	transport := "http"
	serve := func() error { return s.httpServer.Serve(listener) }
	if cfg.TLSConfigured() {
		transport = "https"
		serve = func() error {
			return s.httpServer.ServeTLS(listener, cfg.TLSCertFile, cfg.TLSKeyFile)
		}
	}
	s.logger.With("component", "server").Info(
		"starting woodstar",
		"operation", "start",
		"addr", s.httpServer.Addr,
		"server_url", cfg.ServerURL,
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
	s.logger.With("component", "server").InfoContext(
		ctx,
		"stopping woodstar",
		"operation", "shutdown",
	)
	return s.httpServer.Shutdown(ctx)
}

func routes(options ServerOptions) http.Handler {
	compression := compressionMiddleware()

	r := chi.NewRouter()
	r.Use(clientIPMiddleware(options.Config))
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.SecurityHeaders(options.TransferOrigin))
	r.Use(corsMiddleware(options.Config))

	ordinary := r.With(requestTimeoutMiddleware(defaultRequestTimeout), compression)
	ordinary.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("alive\n"))
	})
	ordinary.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		if err := options.Ready(req.Context()); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ready\n"))
	})

	var protocols ProtocolRoutes
	r.Group(func(group chi.Router) {
		protocols = newProtocolRoutes(group, compression, options.Logger)
	})

	var app AppRoutes
	var storageTransfers chi.Router
	r.Group(func(group chi.Router) {
		app, storageTransfers = newBrowserRoutes(group, compression, options)
	})

	if options.RegisterRoutes != nil {
		options.RegisterRoutes(Routes{
			App:              app,
			Protocols:        protocols,
			StorageTransfers: storageTransfers,
		})
	}
	options.WebHandler.RegisterRoutes(app.Router)

	return r
}

func newProtocolRoutes(
	r chi.Router,
	compression func(http.Handler) http.Handler,
	logger *slog.Logger,
) ProtocolRoutes {
	requestLogger := middleware.RequestLogger(logger)
	return ProtocolRoutes{
		Ordinary:   r.With(requestTimeoutMiddleware(defaultRequestTimeout), compression, requestLogger),
		Transfers:  r.With(requestLogger),
		WebSockets: r.With(requestLogger),
	}
}

func newBrowserRoutes(
	r chi.Router,
	compression func(http.Handler) http.Handler,
	options ServerOptions,
) (AppRoutes, chi.Router) {
	requestLogger := middleware.RequestLogger(options.Logger)
	storageTransfers := r.With(requestLogger)

	sessionMiddleware := options.SessionManager.LoadAndSave
	crossOriginProtection := middleware.CrossOriginProtection(options.Config.CORSAllowedOrigins)
	passwordLoginLimiter := middleware.NewPasswordLoginLimiter()
	// Reject excess attempts before session loading so the denied path cannot
	// reach the session store or any password-authentication work.
	passwordLogin := r.With(
		requestTimeoutMiddleware(defaultRequestTimeout),
		compression,
		requestLogger,
		middleware.LimitPasswordLogin(passwordLoginLimiter),
		sessionMiddleware,
		crossOriginProtection,
	)
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
	apis := newAppAPIs(passwordLogin, ordinary, streaming, longRunning, options.Version)
	return newAppRoutes(ordinary, transfers, apis, options.AuthService), storageTransfers
}

type appAPIs struct {
	passwordLogin huma.API
	ordinary      huma.API
	streaming     huma.API
	longRunning   huma.API
}

func newAppAPIs(
	passwordLogin chi.Router,
	ordinary chi.Router,
	streaming chi.Router,
	longRunning chi.Router,
	version string,
) appAPIs {
	cfg := humaConfig(version)
	return appAPIs{
		passwordLogin: humachi.New(passwordLogin, cfg),
		ordinary:      humachi.New(ordinary, cfg),
		streaming:     humachi.New(streaming, cfg),
		longRunning:   humachi.New(longRunning, cfg),
	}
}

func newAppRoutes(
	ordinaryRouter chi.Router,
	transferRouter chi.Router,
	apis appAPIs,
	authService *auth.Service,
) AppRoutes {
	session := huma.NewGroup(apis.ordinary)
	session.UseMiddleware(middleware.OptionalHumaAuth(apis.ordinary, authService))

	protected := newProtectedGroup(apis.ordinary, authService)
	ordinary := newOrdinaryGroup(protected)
	sensitive := newSensitiveGroup(protected)

	streamingSensitive := newSensitiveGroup(
		newProtectedGroup(apis.streaming, authService),
	)
	longRunningOrdinary := newOrdinaryGroup(
		newProtectedGroup(apis.longRunning, authService),
	)

	return AppRoutes{
		PasswordLogin:       apis.passwordLogin,
		Session:             session,
		Protected:           protected,
		Ordinary:            ordinary,
		Sensitive:           sensitive,
		StreamingSensitive:  streamingSensitive,
		LongRunningOrdinary: longRunningOrdinary,
		Router:              ordinaryRouter,
		Transfers:           transferRouter,
	}
}

// NewSchema returns an app API and route surfaces configured for OpenAPI
// registration without runtime authentication middleware.
func NewSchema(version string) (huma.API, AppRoutes) {
	r := chi.NewRouter()
	apis := newAppAPIs(r, r, r, r, version)

	session := huma.NewGroup(apis.ordinary)
	protected := newDocumentedProtectedGroup(apis.ordinary)
	ordinary := newOrdinaryGroup(protected)
	sensitive := newSensitiveGroup(protected)
	streamingSensitive := newSensitiveGroup(newDocumentedProtectedGroup(apis.streaming))
	longRunningOrdinary := newOrdinaryGroup(newDocumentedProtectedGroup(apis.longRunning))

	return apis.ordinary, AppRoutes{
		PasswordLogin:       apis.passwordLogin,
		Session:             session,
		Protected:           protected,
		Ordinary:            ordinary,
		Sensitive:           sensitive,
		StreamingSensitive:  streamingSensitive,
		LongRunningOrdinary: longRunningOrdinary,
	}
}

func newProtectedGroup(api huma.API, authService *auth.Service) *huma.Group {
	protected := huma.NewGroup(api)
	protected.UseMiddleware(middleware.RequireHumaAuth(api, authService))
	protected.UseModifier(middleware.ProtectedOperation(api))
	return protected
}

func newDocumentedProtectedGroup(api huma.API) *huma.Group {
	protected := huma.NewGroup(api)
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
	cfg := huma.DefaultConfig("API", version)
	cfg.Info.License = &huma.License{Name: "Apache-2.0"}
	configureOpenAPI(cfg.OpenAPI)

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

func compressionMiddleware() func(http.Handler) http.Handler {
	return chimiddleware.Compress(2)
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
