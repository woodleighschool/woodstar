package adminapi

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/api/handlers"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
	"github.com/woodleighschool/woodstar/internal/santa/rules"
)

// Mount attaches public and authenticated admin API routes to r.
func Mount(r chi.Router, deps Dependencies) {
	humaAPI := humachi.New(r, humaConfig(deps.Runtime.Version))
	registerAdminRoutes(r, humaAPI, deps)
}

func registerAdminRoutes(r chi.Router, humaAPI huma.API, deps Dependencies) {
	protected := huma.NewGroup(humaAPI)
	protected.UseMiddleware(RequireAuth(humaAPI, deps.Auth.AuthService))
	admin := huma.NewGroup(protected)
	admin.UseMiddleware(RequireAdmin(humaAPI))

	auth.RegisterPublicAdminRoutes(humaAPI, deps.Auth.AuthService)
	RegisterSSO(r, deps.Auth.AuthService)
	auth.RegisterAccountAdminRoutes(protected, deps.Auth.AuthService, deps.Auth.UserService)
	directory.RegisterUserAdminRoutes(admin, deps.Auth.UserService)
	directory.RegisterGroupAdminRoutes(admin, deps.Directory.Store)
	handlers.RegisterHosts(
		protected,
		deps.Inventory.Hosts,
		deps.Inventory.UserAffinities,
		deps.Inventory.Software,
		deps.Osquery.Checks,
		handlers.MunkiHostDetailContributor(deps.Munki.HostState),
		handlers.SantaHostDetailContributor(deps.Santa.HostState),
	)
	handlers.RegisterSoftware(protected, deps.Inventory.Software, deps.Santa.References)
	labels.RegisterAdminRoutes(protected, deps.Inventory.Labels)
	agentauth.RegisterAdminRoutes(admin, deps.AgentAuth.Store)
	reports.RegisterAdminRoutes(protected, deps.Osquery.Reports)
	reports.RegisterHostAdminRoutes(protected, deps.Osquery.Reports, deps.Inventory.Hosts)
	checks.RegisterAdminRoutes(protected, deps.Osquery.Checks)
	checks.RegisterHostAdminRoutes(protected, deps.Osquery.Checks, deps.Inventory.Hosts)
	livequery.RegisterAdminRoutes(protected, deps.Osquery.LiveQueries, deps.Inventory.Hosts)
	configurations.RegisterAdminRoutes(admin, deps.Santa.Configurations)
	rules.RegisterAdminRoutes(admin, deps.Santa.Rules)
	events.RegisterAdminRoutes(admin, deps.Santa.Events)
	rules.RegisterHostAdminRoutes(protected, deps.Santa.Rules, deps.Inventory.Hosts)
	handlers.RegisterMunki(admin, handlers.MunkiStores{
		Artifacts:      deps.Munki.Artifacts,
		Packages:       deps.Munki.Packages,
		SoftwareTitles: deps.Munki.SoftwareTitles,
	}, deps.Munki.ArtifactStorage)
}
