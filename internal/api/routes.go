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

	handlers.RegisterPublicAuth(humaAPI, deps.Auth.AuthService)
	handlers.RegisterSSO(r, deps.Auth.AuthService)
	handlers.RegisterAccount(protected, deps.Auth.AuthService, deps.Auth.UserService)
	handlers.RegisterUsers(protected, deps.Auth.UserService)
	handlers.RegisterHosts(
		protected,
		deps.Inventory.HostStore,
		deps.Inventory.SoftwareStore,
		deps.Santa.Store,
	)
	handlers.RegisterSoftware(protected, deps.Inventory.SoftwareStore)
	handlers.RegisterLabels(protected, deps.Inventory.LabelStore)
	handlers.RegisterReports(protected, deps.Inventory.ReportStore, deps.Inventory.HostStore)
	handlers.RegisterChecks(protected, deps.Inventory.CheckStore, deps.Inventory.HostStore)
	handlers.RegisterLiveQueries(protected, deps.Orbit.LiveQueryManager, deps.Inventory.HostStore)
	handlers.RegisterEnrollSecrets(protected, deps.Inventory.SecretStore)
	handlers.RegisterSantaSyncTokens(protected, deps.Santa.Store)
	handlers.RegisterSantaConfigurations(protected, deps.Santa.Store)
	handlers.RegisterSantaRules(protected, deps.Santa.Store)
}
