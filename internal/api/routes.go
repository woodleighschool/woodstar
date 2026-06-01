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
		deps.Inventory.Hosts,
		deps.Inventory.UserAffinities,
		deps.Inventory.Software,
		handlers.MunkiHostDetailContributor(deps.Munki.Store),
		handlers.SantaHostDetailContributor(deps.Santa.HostState),
	)
	handlers.RegisterSoftware(protected, deps.Inventory.Software, deps.Santa.References)
	handlers.RegisterLabels(protected, deps.Inventory.Labels)
	handlers.RegisterDirectory(protected, deps.Directory.Store)
	handlers.RegisterAgentSecrets(admin, deps.AgentAuth.Store)
	handlers.RegisterReports(protected, deps.Osquery.Reports)
	handlers.RegisterHostReports(protected, deps.Osquery.Reports, deps.Inventory.Hosts)
	handlers.RegisterChecks(protected, deps.Osquery.Checks)
	handlers.RegisterHostChecks(protected, deps.Osquery.Checks, deps.Inventory.Hosts)
	handlers.RegisterLiveQueries(protected, deps.Osquery.LiveQueries, deps.Inventory.Hosts)
	handlers.RegisterSantaConfigurations(admin, deps.Santa.Configurations)
	handlers.RegisterSantaRules(admin, deps.Santa.Rules)
	handlers.RegisterSantaEvents(admin, deps.Santa.Events)
	handlers.RegisterHostSantaRules(protected, deps.Inventory.Hosts, deps.Santa.Rules)
	handlers.RegisterMunki(admin, deps.Munki.Store, deps.Munki.ArtifactStorage)
}
