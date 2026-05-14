package api

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/api/handlers"
	"github.com/woodleighschool/woodstar/internal/api/middleware"
)

// Mount attaches public and authenticated admin API routes to r.
func Mount(r chi.Router, deps Dependencies) huma.API {
	humaAPI := humachi.New(r, humaConfig(deps.Version))
	protected := huma.NewGroup(humaAPI)
	protected.UseMiddleware(middleware.RequireAuth(humaAPI, deps.AuthService))

	handlers.RegisterPublicAuth(humaAPI, deps.AuthService)
	handlers.RegisterAccount(protected, deps.AuthService)
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
	r.With(middleware.RequireAuthChi(deps.AuthService)).
		Get("/api/live-queries/{id}/stream", handlers.LiveQueryStreamHandler(deps.LiveQueryManager))

	return humaAPI
}
