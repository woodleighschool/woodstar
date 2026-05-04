package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
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

// SessionTTL is the browser session lifetime.
const SessionTTL = 14 * 24 * time.Hour

// ServerDependencies contains runtime dependencies for Server.
type ServerDependencies struct {
	Config         config.Config
	DB             *database.DB
	Version        string
	WebHandler     *web.Handler
	AuthService    *auth.Service
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

	log.Info().Str("addr", addr).Str("base_url", s.config.BaseURL).Msg("starting woodstar")
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

	app := chi.NewRouter()

	app.Group(func(r chi.Router) {
		r.Use(apimiddleware.RequireAuth(s.basePath(), s.authService))
		registerAdminAPI(r, s.db, s.version, s.started, s.authService, adminStores{
			Hosts:          s.hostStore,
			DeviceMappings: s.deviceMappings,
			Secrets:        s.secretStore,
			Software:       s.softwareStore,
		}, handlers.CookieSettings{
			CookiePath:   s.basePath(),
			SecureCookie: secureCookie(s.config.BaseURL),
		})
	})

	// Agent endpoints authenticate with enroll secrets or node keys.
	orbit.RegisterRoutes(app, s.orbitService)
	osquery.RegisterRoutes(app, s.osqueryService)

	if s.webHandler != nil {
		s.webHandler.RegisterRoutes(app)
	}

	if s.basePath() == "/" {
		r.Mount("/", app)
	} else {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, s.basePath()+"/", http.StatusMovedPermanently)
		})
		r.Mount(s.basePath(), app)
	}

	return r
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
	cookies handlers.CookieSettings,
) huma.API {
	api := humachi.New(r, adminConfig(version))

	handlers.RegisterSystem(api, db, version, started)
	handlers.RegisterAuth(api, authService, cookies)
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
func BuildAdminAPI(version string) huma.API {
	r := chi.NewRouter()
	authService := auth.NewService(nil, nil, SessionTTL, "openapi-only-session-secret")
	return registerAdminAPI(r, nil, version, time.Now().UTC(), authService, adminStores{
		Hosts:          models.NewHostStore(nil),
		DeviceMappings: models.NewDeviceMappingStore(nil),
		Secrets:        models.NewSecretStore(nil),
		Software:       models.NewSoftwareStore(nil),
	}, handlers.CookieSettings{CookiePath: "/"})
}

func (s *Server) basePath() string {
	return s.config.BasePath()
}

func secureCookie(baseURL string) bool {
	parsed, err := url.Parse(baseURL)
	return err == nil && parsed.Scheme == "https"
}
