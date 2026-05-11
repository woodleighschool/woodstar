package api

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	agentapi "github.com/woodleighschool/woodstar/internal/agents/api"
	"github.com/woodleighschool/woodstar/internal/api/handlers"
	"github.com/woodleighschool/woodstar/internal/api/middleware"
)

// Mount attaches public and authenticated admin API routes to r.
func Mount(r chi.Router, deps Dependencies) huma.API {
	humaAPI := humachi.New(r, humaConfig(deps.Version))
	protected := huma.NewGroup(humaAPI)
	protected.UseMiddleware(middleware.RequireAuth(humaAPI, deps.AuthService))

	handlers.RegisterPublicAuth(humaAPI, deps.AuthService)
	handlers.RegisterUsers(protected, deps.UserService)
	handlers.RegisterHosts(protected, deps.HostStore, deps.DeviceMappings, deps.SoftwareStore, deps.LabelStore)
	handlers.RegisterSoftware(protected, deps.SoftwareStore)
	handlers.RegisterLabels(protected, deps.LabelStore)
	agentapi.RegisterQueries(protected, deps.QueryStore, deps.HostStore)
	agentapi.RegisterChecks(protected, deps.CheckStore, deps.HostStore)
	agentapi.RegisterLiveQueries(protected, deps.LiveQueryManager, deps.TargetResolver)
	handlers.RegisterSecrets(protected, deps.SecretStore)

	// SSE lives outside the Huma group (Huma's typed-body model doesn't fit
	// text/event-stream). Auth uses the Chi-compatible session middleware.
	r.With(middleware.RequireAuthChi(deps.AuthService)).
		Get("/api/live-queries/{id}/stream", agentapi.LiveQueryStreamHandler(deps.LiveQueryManager))

	return humaAPI
}
