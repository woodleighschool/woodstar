package api

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/api/handlers"
)

// Mount attaches public and authenticated admin API routes to r.
func Mount(r chi.Router, deps Dependencies) {
	humaAPI := humachi.New(r, humaConfig(deps.Runtime.Version))
	registerAdminRoutes(r, humaAPI, deps)
}

func registerAdminRoutes(r chi.Router, humaAPI huma.API, deps Dependencies) {
	protected := huma.NewGroup(humaAPI)
	protected.UseMiddleware(handlers.RequireAuth(humaAPI, deps.Auth.AuthService))
	admin := huma.NewGroup(protected)
	admin.UseMiddleware(handlers.RequireAdmin(humaAPI))

	handlers.RegisterPublicAuth(humaAPI, deps.Auth.AuthService)
	handlers.RegisterSSO(r, deps.Auth.AuthService)
	handlers.RegisterAccount(protected, deps.Auth.AuthService, deps.Auth.UserService)
	handlers.RegisterUsers(protected, deps.Auth.UserService)
	handlers.RegisterHosts(
		protected,
		deps.Hosts.Store,
		deps.Software.Store,
		handlers.SantaHostDetailContributor(deps.Santa.HostState),
	)
	handlers.RegisterSoftware(protected, deps.Software.Store)
	handlers.RegisterLabels(protected, deps.Labels.Store)
	handlers.RegisterAgentSecrets(admin, deps.AgentAuth.Store)
	handlers.RegisterReports(protected, deps.Osquery.Reports, deps.Hosts.Store)
	handlers.RegisterChecks(protected, deps.Osquery.Checks, deps.Hosts.Store)
	handlers.RegisterLiveQueries(protected, deps.Osquery.LiveQueries, deps.Hosts.Store)
	handlers.RegisterSantaConfigurations(admin, deps.Santa.Configurations)
	handlers.RegisterSantaRules(admin, deps.Santa.Rules)
	handlers.RegisterSantaEvents(admin, deps.Santa.Events)
	handlers.RegisterHostSantaEffectiveRules(protected, deps.Hosts.Store, deps.Santa.Rules)
}
