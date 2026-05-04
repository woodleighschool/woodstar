package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"
	"github.com/rs/zerolog/log"

	"github.com/woodleighschool/woodstar/internal/api/handlers"
	apimiddleware "github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/config"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/models"
	"github.com/woodleighschool/woodstar/internal/orbit"
	"github.com/woodleighschool/woodstar/internal/osquery"
	"github.com/woodleighschool/woodstar/internal/web"
)

// SessionLifetime is the browser session lifetime.
const SessionLifetime = 14 * 24 * time.Hour

// ServerDependencies contains runtime dependencies for Server.
type ServerDependencies struct {
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
func NewServer(deps ServerDependencies) *Server {
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

	// Agent endpoints authenticate with enroll secrets or node keys; they must
	// not pass through CSRF because Orbit/osquery can't carry a browser CSRF token.
	orbit.RegisterRoutes(r, s.orbitService)
	osquery.RegisterRoutes(r, s.osqueryService)

	// Browser-facing routes share the CSRF middleware so the SPA's index.html
	// can read the token via csrf.Token(r) and admin mutations are verified.
	r.Group(func(g chi.Router) {
		if !s.config.IsHTTPS() {
			// gorilla/csrf assumes https when constructing the request URL it
			// compares against the Origin header; mark plaintext so the check works on http.
			g.Use(plaintextHTTPMiddleware)
		}
		g.Use(s.csrfMiddleware())
		g.Group(func(adminG chi.Router) {
			adminG.Use(apimiddleware.RequireAuth(s.authService))
			registerAdminAPI(adminG, s.db, s.version, s.started, s.authService, adminStores{
				Hosts:          s.hostStore,
				DeviceMappings: s.deviceMappings,
				Secrets:        s.secretStore,
				Software:       s.softwareStore,
			})
		})
		if s.webHandler != nil {
			s.webHandler.RegisterRoutes(g)
		}
	})

	return r
}

func plaintextHTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, csrf.PlaintextHTTPRequest(r))
	})
}

func (s *Server) csrfMiddleware() func(http.Handler) http.Handler {
	return csrf.Protect(
		[]byte(s.config.SessionSecret),
		csrf.Secure(s.config.IsHTTPS()),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.Path("/"),
		csrf.CookieName("woodstar_csrf"),
		// Match the session cookie lifetime so a cached masked token in the SPA
		// stays valid for as long as the session it accompanies.
		csrf.MaxAge(int(SessionLifetime.Seconds())),
	)
}

// adminConfig returns the Huma config shared by serve and openapi.
func adminConfig(version string) huma.Config {
	cfg := huma.DefaultConfig("Woodstar API", version)
	cfg.Info.Description = "Typed admin and frontend API."
	cfg.Info.License = &huma.License{Name: "Apache-2.0"}
	cfg.DocsPath = "/api/docs"
	cfg.OpenAPIPath = "/api/openapi"
	cfg.SchemasPath = "/api/schemas"
	cfg.Servers = []*huma.Server{{URL: "/"}}
	return cfg
}

// registerAdminAPI attaches admin routes to r.
func registerAdminAPI(
	r chi.Router,
	db *database.DB,
	version string,
	started time.Time,
	authService *auth.Service,
	stores adminStores,
) huma.API {
	api := humachi.New(r, adminConfig(version))

	handlers.RegisterSystem(api, db, version, started)
	handlers.RegisterAuth(api, authService)
	handlers.RegisterUsers(api, authService)
	handlers.RegisterHosts(api, stores.Hosts, stores.DeviceMappings, stores.Software)
	handlers.RegisterSoftware(api, stores.Software)
	handlers.RegisterSecrets(api, stores.Secrets)

	return api
}

type adminStores struct {
	Hosts          *models.HostStore
	DeviceMappings *models.DeviceMappingStore
	Secrets        *models.SecretStore
	Software       *models.SoftwareStore
}

// BuildAdminAPI returns the admin API without starting the server.
// Used by the openapi subcommand to render the spec without a database.
func BuildAdminAPI(version string) huma.API {
	r := chi.NewRouter()
	authService := auth.NewService(nil, nil)
	return registerAdminAPI(r, nil, version, time.Now().UTC(), authService, adminStores{
		Hosts:          models.NewHostStore(nil),
		DeviceMappings: models.NewDeviceMappingStore(nil),
		Secrets:        models.NewSecretStore(nil),
		Software:       models.NewSoftwareStore(nil),
	})
}
