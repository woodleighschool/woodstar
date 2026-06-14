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
	ordinary := huma.NewGroup(protected)
	ordinary.UseModifier(RequireAdminForMutations(humaAPI))
	sensitive := huma.NewGroup(protected)
	sensitive.UseModifier(requireAdminForAll(humaAPI))

	auth.RegisterPublicAdminRoutes(humaAPI, deps.Auth.AuthService)
	RegisterSSO(r, deps.Auth.AuthService)
	auth.RegisterAccountAdminRoutes(protected, deps.Auth.AuthService, deps.Auth.UserService)
	directory.RegisterUserAdminRoutes(ordinary, deps.Auth.UserService)
	directory.RegisterGroupAdminRoutes(ordinary, deps.Directory.Store)
	hosts.RegisterAdminRoutes(ordinary, hostRoutesOptions(deps))
	inventory.RegisterAdminRoutes(ordinary, deps.Inventory.Software)
	inventory.RegisterHostAdminRoutes(ordinary, deps.Inventory.Software, deps.Inventory.Hosts)
	references.RegisterSoftwareAdminRoutes(ordinary, deps.Santa.References)
	labels.RegisterAdminRoutes(ordinary, deps.Inventory.Labels)
	agentauth.RegisterAdminRoutes(sensitive, deps.AgentAuth.Store)
	reports.RegisterAdminRoutes(ordinary, deps.Osquery.Reports)
	reports.RegisterHostAdminRoutes(ordinary, deps.Osquery.Reports, deps.Inventory.Hosts)
	checks.RegisterAdminRoutes(ordinary, deps.Osquery.Checks)
	checks.RegisterHostAdminRoutes(ordinary, deps.Osquery.Checks, deps.Inventory.Hosts)
	livequery.RegisterAdminRoutes(sensitive, deps.Osquery.LiveQueries, deps.Inventory.Hosts)
	configurations.RegisterAdminRoutes(ordinary, deps.Santa.Configurations)
	rules.RegisterAdminRoutes(ordinary, deps.Santa.Rules)
	events.RegisterAdminRoutes(ordinary, deps.Santa.Events)
	rules.RegisterHostAdminRoutes(ordinary, deps.Santa.Rules, deps.Inventory.Hosts)
	munkisoftware.RegisterAdminRoutes(ordinary, deps.Munki.Software, deps.Munki.Packages)
	munkipackages.RegisterAdminRoutes(ordinary, deps.Munki.Packages)
}

func hostRoutesOptions(deps Dependencies) hosts.AdminRoutesOptions[HostDetail] {
	var checkFilter hosts.CheckStatusFilter
	if deps.Osquery.Checks != nil {
		checkFilter = osqueryCheckFilter{store: deps.Osquery.Checks}
	}
	return hosts.AdminRoutesOptions[HostDetail]{
		Store:          deps.Inventory.Hosts,
		UserAffinities: deps.Inventory.UserAffinities,
		CheckFilter:    checkFilter,
		DetailBuilder:  func(detail hosts.HostDetail) HostDetail { return HostDetail{HostDetail: detail} },
		Contributors: []hosts.DetailContributor[HostDetail]{
			newMunkiHostDetailContributor(deps.Munki.HostState),
			newSantaHostDetailContributor(deps.Santa.HostState),
		},
	}
}
