// Package admin provides the Huma-backed browser/admin HTTP API.
package admin

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/db"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/models"
	queryinfra "github.com/woodleighschool/woodstar/internal/queries"
	softwarepkg "github.com/woodleighschool/woodstar/internal/software"
	"github.com/woodleighschool/woodstar/internal/transport/admin/handlers"
)

// Dependencies contains the services and stores used by the admin HTTP API.
type Dependencies struct {
	DB               *db.DB
	Version          string
	Started          time.Time
	AuthService      *auth.Service
	HostStore        *hosts.HostStore
	DeviceMappings   *hosts.DeviceMappingStore
	SecretStore      *models.SecretStore
	SoftwareStore    *softwarepkg.SoftwareStore
	LabelStore       *labels.LabelStore
	QueryStore       *queryinfra.QueryStore
	CheckStore       *queryinfra.CheckStore
	LiveQueryManager *queryinfra.LiveQueryManager
}

// Mount attaches public and authenticated admin API routes to r.
func Mount(r chi.Router, deps Dependencies) huma.API {
	api := humachi.New(r, Config(deps.Version))
	protected := huma.NewGroup(api)
	protected.UseMiddleware(RequireAuth(api, deps.AuthService))

	handlers.RegisterSystem(api, deps.DB, deps.Version, deps.Started)
	handlers.RegisterPublicAuth(api, deps.AuthService)
	handlers.RegisterUsers(protected, deps.AuthService)
	handlers.RegisterHosts(protected, deps.HostStore, deps.DeviceMappings, deps.SoftwareStore, deps.LabelStore)
	handlers.RegisterSoftware(protected, deps.SoftwareStore)
	handlers.RegisterLabels(protected, deps.LabelStore)
	handlers.RegisterQueries(protected, deps.QueryStore, deps.HostStore)
	handlers.RegisterChecks(protected, deps.CheckStore, deps.HostStore)
	handlers.RegisterLiveQueries(protected, deps.LiveQueryManager, hosts.NewTargetResolver(deps.DB))
	handlers.RegisterSecrets(protected, deps.SecretStore)

	// SSE lives outside the Huma group (Huma's typed-body model doesn't fit
	// text/event-stream). Auth uses the Chi-compatible session middleware.
	r.With(RequireAuthChi(deps.AuthService)).
		Get("/api/live-queries/{id}/stream", handlers.LiveQueryStreamHandler(deps.LiveQueryManager))

	return api
}

// BuildAPI returns the admin API without starting the server.
func BuildAPI(version string) huma.API {
	r := chi.NewRouter()
	hub := queryinfra.NewHub()
	return Mount(r, Dependencies{
		Version:          version,
		Started:          time.Now().UTC(),
		AuthService:      auth.NewService(nil, nil),
		HostStore:        hosts.NewHostStore(nil),
		DeviceMappings:   hosts.NewDeviceMappingStore(nil),
		SecretStore:      models.NewSecretStore(nil),
		SoftwareStore:    softwarepkg.NewSoftwareStore(nil),
		LabelStore:       labels.NewLabelStore(nil),
		QueryStore:       queryinfra.NewQueryStore(nil),
		CheckStore:       queryinfra.NewCheckStore(nil),
		LiveQueryManager: queryinfra.NewLiveQueryManager(hub, time.Minute),
	})
}
