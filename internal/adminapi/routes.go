package adminapi

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/inventory"
	"github.com/woodleighschool/woodstar/internal/labels"
	munkiartifacts "github.com/woodleighschool/woodstar/internal/munki/artifacts"
	munkipackages "github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	"github.com/woodleighschool/woodstar/internal/santa/events"
	"github.com/woodleighschool/woodstar/internal/santa/references"
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
	hosts.RegisterAdminRoutes(protected, hostRoutesOptions(deps))
	inventory.RegisterAdminRoutes(protected, deps.Inventory.Software)
	inventory.RegisterHostAdminRoutes(protected, deps.Inventory.Software, deps.Inventory.Hosts)
	references.RegisterSoftwareAdminRoutes(protected, deps.Santa.References)
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
	munkisoftware.RegisterAdminRoutes(admin, deps.Munki.SoftwareTitles, deps.Munki.Packages)
	munkiartifacts.RegisterAdminRoutes(admin, deps.Munki.Artifacts, deps.Munki.ArtifactStorage)
	munkipackages.RegisterAdminRoutes(admin, deps.Munki.Packages)
}

func hostRoutesOptions(deps Dependencies) hosts.AdminRoutesOptions[HostDetail] {
	var checkFilter hosts.CheckStatusFilter
	if deps.Osquery.Checks != nil {
		checkFilter = osqueryCheckFilter{store: deps.Osquery.Checks}
	}
	return hosts.AdminRoutesOptions[HostDetail]{
		Store:          deps.Inventory.Hosts,
		UserAffinities: deps.Inventory.UserAffinities,
		RequireAdmin:   requireAdminUser,
		CheckFilter:    checkFilter,
		DetailBuilder:  func(detail hosts.HostDetail) HostDetail { return HostDetail{HostDetail: detail} },
		Contributors: []hosts.DetailContributor[HostDetail]{
			newMunkiHostDetailContributor(deps.Munki.HostState),
			newSantaHostDetailContributor(deps.Santa.HostState),
		},
	}
}
