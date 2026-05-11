// Package admin provides the Huma-backed browser/admin HTTP API.
package admin

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/queries"
	"github.com/woodleighschool/woodstar/internal/secrets"
	"github.com/woodleighschool/woodstar/internal/software"
	"github.com/woodleighschool/woodstar/internal/transport/admin/handlers"
	"github.com/woodleighschool/woodstar/internal/users"
)

// Dependencies contains the services and stores used by the admin HTTP API.
type Dependencies struct {
	DB               *database.DB
	Version          string
	AuthService      *auth.Service
	UserService      *users.Service
	HostStore        *hosts.HostStore
	DeviceMappings   *hosts.DeviceMappingStore
	SecretStore      *secrets.Store
	SoftwareStore    *software.SoftwareStore
	LabelStore       *labels.LabelStore
	QueryStore       *queries.QueryStore
	CheckStore       *queries.CheckStore
	LiveQueryManager *queries.LiveQueryManager
	TargetResolver   *hosts.TargetResolver
}

// Mount attaches public and authenticated admin API routes to r.
func Mount(r chi.Router, deps Dependencies) huma.API {
	api := humachi.New(r, Config(deps.Version))
	protected := huma.NewGroup(api)
	protected.UseMiddleware(RequireAuth(api, deps.AuthService))

	handlers.RegisterPublicAuth(api, deps.AuthService)
	handlers.RegisterUsers(protected, deps.UserService)
	handlers.RegisterHosts(protected, deps.HostStore, deps.DeviceMappings, deps.SoftwareStore, deps.LabelStore)
	handlers.RegisterSoftware(protected, deps.SoftwareStore)
	handlers.RegisterLabels(protected, deps.LabelStore)
	handlers.RegisterQueries(protected, deps.QueryStore, deps.HostStore)
	handlers.RegisterChecks(protected, deps.CheckStore, deps.HostStore)
	handlers.RegisterLiveQueries(protected, deps.LiveQueryManager, deps.TargetResolver)
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
	return Mount(r, Dependencies{
		Version: version,
	})
}
