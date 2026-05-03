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
	"github.com/woodleighschool/woodstar/internal/web"
)

const sessionTTL = 14 * 24 * time.Hour

// ServerDependencies are the external pieces needed to build a Server.
type ServerDependencies struct {
	Config     config.Config
	DB         *database.DB
	Version    string
	WebHandler *web.Handler
}

// Server owns the Woodstar HTTP listener and router.
type Server struct {
	httpServer *http.Server
	config     config.Config
	db         *database.DB
	version    string
	webHandler *web.Handler
	started    time.Time
}

// NewServer wires the HTTP server with the provided runtime dependencies.
func NewServer(deps ServerDependencies) *Server {
	return &Server{
		httpServer: &http.Server{
			ReadHeaderTimeout: 15 * time.Second,
			ReadTimeout:       60 * time.Second,
			WriteTimeout:      120 * time.Second,
			IdleTimeout:       180 * time.Second,
		},
		config:     deps.Config,
		db:         deps.DB,
		version:    deps.Version,
		webHandler: deps.WebHandler,
		started:    time.Now().UTC(),
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

	authService := auth.NewService(
		models.NewUserStore(s.db),
		models.NewSessionStore(s.db),
		sessionTTL,
		s.config.SessionSecret,
	)

	app.Group(func(r chi.Router) {
		r.Use(apimiddleware.RequireAuth(s.basePath(), authService))
		registerAdminAPI(r, s.db, s.version, s.started, authService, handlers.CookieSettings{
			CookiePath:   s.basePath(),
			SecureCookie: secureCookie(s.config.BaseURL),
		})
	})

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

// adminConfig is the canonical Huma config used for the Woodstar admin API.
// Kept in one place so the running server and the openapi command stay in sync.
func adminConfig(version string) huma.Config {
	cfg := huma.DefaultConfig("Woodstar API", version)
	cfg.Info.Description = "Typed admin and frontend API for Woodstar." +
		" Agent compatibility endpoints (Orbit/osquery, Santa sync, Munki repo) are deliberately not part of this document."
	cfg.Info.License = &huma.License{Name: "Apache-2.0"}
	cfg.DocsPath = "/api/docs"
	cfg.OpenAPIPath = "/api/openapi"
	cfg.SchemasPath = "/api/schemas"
	cfg.Servers = []*huma.Server{{URL: "/"}}
	return cfg
}

// registerAdminAPI builds and attaches the admin Huma API to the given router.
func registerAdminAPI(
	r chi.Router,
	db *database.DB,
	version string,
	started time.Time,
	authService *auth.Service,
	cookies handlers.CookieSettings,
) huma.API {
	api := humachi.New(r, adminConfig(version))

	handlers.RegisterSystem(api, db, version, started)
	handlers.RegisterAuth(api, authService, cookies)
	handlers.RegisterUsers(api, authService)
	handlers.RegisterSecrets(api, models.NewSecretStore(db))

	return api
}

// BuildAdminAPI returns a populated Huma API without starting the HTTP server.
// The openapi command uses this to dump the spec without standing up the rest
// of the runtime. Handlers are registered but never invoked, so a nil DB is
// safe.
func BuildAdminAPI(version string) huma.API {
	r := chi.NewRouter()
	authService := auth.NewService(nil, nil, sessionTTL, "openapi-only-session-secret")
	return registerAdminAPI(r, nil, version, time.Now().UTC(), authService, handlers.CookieSettings{CookiePath: "/"})
}

func (s *Server) basePath() string {
	return s.config.BasePath()
}

func secureCookie(baseURL string) bool {
	parsed, err := url.Parse(baseURL)
	return err == nil && parsed.Scheme == "https"
}
