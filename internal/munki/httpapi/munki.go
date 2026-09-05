package httpapi

import (
	"log/slog"

	"github.com/woodleighschool/goodies/bloby"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
)

// Dependencies are the services used by the Munki HTTP boundary.
type Dependencies struct {
	AuthService     *auth.Service
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
	registerMunkiContentRoutes(
		routes.Transfers.With(middleware.RequireHTTPAuth(deps.AuthService)),
		deps.Objects,
		deps.Logger,
	)
}

// RegisterOpenAPI documents Munki endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerOperations(routes, Dependencies{})
}

func registerOperations(routes api.AppRoutes, deps Dependencies) {
	registerHostMunkiState(routes.Ordinary, deps.HostState, deps.Logger)
	registerHostMunkiSoftware(routes.Ordinary, deps.Software, deps.Logger)
	registerMunkiSoftware(
		routes.Ordinary,
		deps.Software,
		deps.DeleteSoftware,
		deps.Packages,
		deps.Objects,
		deps.Logger,
	)
	registerMunkiPackages(
		routes.Ordinary,
		routes.LongRunningOrdinary,
		deps.Packages,
		deps.Objects,
		deps.Logger,
	)
	registerMunkiClientResources(
		routes.Ordinary,
		deps.ClientResources,
		deps.Objects,
		deps.Logger,
	)
	registerMunkiDistributionPoints(routes.Ordinary, deps.Distribution, deps.Connections, deps.Logger)
}
