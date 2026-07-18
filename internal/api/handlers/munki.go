package handlers

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/clientresources"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// MunkiHandlerDeps are the stores, services, and route groups used by Munki
// admin handlers.
type MunkiHandlerDeps struct {
	API             huma.API
	LongRunningAPI  huma.API
	TransferRouter  chi.Router
	AuthService     *auth.Service
	HostState       *munki.Store
	Software        *munkisoftware.Store
	DeleteSoftware  *munki.SoftwareDeletionService
	Packages        *munki.PackageService
	ClientResources *clientresources.Service
	Objects         *storage.ObjectStore
	Ingestor        *storage.Ingestor
	Delivery        storage.Deliverer
	Distribution    *mdp.Store
	Connections     distributionPointConnections
	Logger          *slog.Logger
}

type distributionPointConnections interface {
	Disconnect(pointID int64)
}

// RegisterMunki mounts Munki host state, software, package, and distribution
// point endpoints.
func RegisterMunki(deps MunkiHandlerDeps) {
	registerHostMunkiState(deps.API, deps.HostState, deps.Logger)
	registerMunkiSoftware(
		deps.API,
		deps.Software,
		deps.DeleteSoftware,
		deps.Packages,
		deps.Objects,
		deps.Ingestor,
		deps.Logger,
	)
	registerMunkiContentRoutes(
		deps.TransferRouter.With(middleware.RequireHTTPAuth(deps.AuthService)),
		deps.Software,
		deps.Objects,
		deps.Delivery,
		deps.Logger,
	)
	registerMunkiPackages(
		deps.API,
		deps.LongRunningAPI,
		deps.Packages,
		deps.Ingestor,
		deps.Logger,
	)
	registerMunkiClientResources(
		deps.API,
		deps.ClientResources,
		deps.Objects,
		deps.Ingestor,
		deps.Logger,
	)
	registerMunkiDistributionPoints(deps.API, deps.Distribution, deps.Connections, deps.Logger)
}
