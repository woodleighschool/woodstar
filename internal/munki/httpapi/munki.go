package httpapi

import (
	"log/slog"

	"github.com/woodleighschool/goodies/auth/authz"
	authhttp "github.com/woodleighschool/goodies/auth/http"
	authhuma "github.com/woodleighschool/goodies/auth/huma"

	"github.com/woodleighschool/goodies/bloby"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/rbac"
)

// Dependencies are the services used by the Munki HTTP boundary.
type Dependencies struct {
	Authenticator   authhttp.Authenticator
	Authorizer      authhuma.Authorizer
	HostState       *munki.Store
	Software        *munkisoftware.Store
	DeleteSoftware  *munki.SoftwareDeletionService
	Packages        *munki.PackageService
	ClientResources *clientresources.Service
	Objects         *bloby.Service
	Distribution    *mdp.Store
	Connections     distributionPointConnections
	Logger          *slog.Logger
}

type distributionPointConnections interface {
	Disconnect(pointID int64)
}

// RegisterAPI mounts Munki host state, software, package, and distribution
// point endpoints.
func RegisterAPI(routes api.AppRoutes, deps Dependencies) {
	registerOperations(routes, deps)
	transfers := routes.Transfers.With(authhttp.RequireAuth(deps.Authenticator, deps.Logger))
	registerMunkiContentRoutes(
		transfers.With(authhttp.RequirePermission(deps.Authorizer, deps.Logger, rbac.ResourceMunkiSoftware, authz.View)),
		transfers.With(authhttp.RequirePermission(deps.Authorizer, deps.Logger, rbac.ResourceMunkiPackages, authz.View)),
		transfers.With(authhttp.RequirePermission(deps.Authorizer, deps.Logger, rbac.ResourceMunkiClientResources, authz.View)),
		deps.Objects,
		deps.Logger,
	)
}

// RegisterOpenAPI documents Munki endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerOperations(routes, Dependencies{})
}

func registerOperations(routes api.AppRoutes, deps Dependencies) {
	softwareAPI := authhuma.ResourceAPI(routes.Protected, deps.Authorizer, deps.Logger, rbac.ResourceMunkiSoftware)
	packagesAPI := authhuma.ResourceAPI(routes.Protected, deps.Authorizer, deps.Logger, rbac.ResourceMunkiPackages)
	longRunningPackagesAPI := authhuma.ResourceAPI(routes.LongRunning, deps.Authorizer, deps.Logger,
		rbac.ResourceMunkiPackages,
	)
	clientResourcesAPI := authhuma.ResourceAPI(routes.Protected, deps.Authorizer, deps.Logger,
		rbac.ResourceMunkiClientResources,
	)
	distributionPointsAPI := authhuma.ResourceAPI(routes.Protected, deps.Authorizer, deps.Logger,
		rbac.ResourceMunkiDistributionPoints,
	)

	registerHostMunkiState(softwareAPI, deps.HostState, deps.Logger)
	registerHostMunkiSoftware(softwareAPI, deps.Software, deps.Logger)
	registerMunkiSoftware(
		softwareAPI,
		deps.Software,
		deps.DeleteSoftware,
		deps.Packages,
		deps.Objects,
		deps.Logger,
	)
	registerMunkiPackages(
		packagesAPI,
		longRunningPackagesAPI,
		deps.Packages,
		deps.Objects,
		deps.Logger,
	)
	registerMunkiClientResources(
		clientResourcesAPI,
		deps.ClientResources,
		deps.Objects,
		deps.Logger,
	)
	registerMunkiDistributionPoints(
		distributionPointsAPI,
		deps.Distribution,
		deps.Connections,
		deps.Logger,
	)
}
