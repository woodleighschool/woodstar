package handlers

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// MunkiHandlerDeps are the stores, services, and route groups used by Munki
// admin handlers.
type MunkiHandlerDeps struct {
	API          huma.API
	Router       chi.Router
	AuthService  *auth.Service
	HostState    *munki.Store
	Software     *munkisoftware.Store
	Packages     *munki.PackageService
	Objects      *storage.ObjectStore
	Storage      storage.Backend
	Distribution *mdp.Store
	Logger       *slog.Logger
}

// RegisterMunki mounts Munki host state, software, package, and distribution
// point endpoints.
func RegisterMunki(deps MunkiHandlerDeps) {
	registerHostMunkiState(deps.API, deps.HostState, deps.Logger)
	registerMunkiSoftware(deps.API, deps.Software, deps.Packages, deps.Objects, deps.Storage, deps.Logger)
	registerMunkiSoftwareIconContent(
		deps.Router.With(middleware.RequireHTTPAuth(deps.AuthService)),
		deps.Software,
		deps.Objects,
		deps.Storage,
		deps.Logger,
	)
	registerMunkiPackages(deps.API, deps.Packages, deps.Objects, deps.Storage, deps.Logger)
	registerMunkiDistributionPoints(deps.API, deps.Distribution, deps.Logger)
}
