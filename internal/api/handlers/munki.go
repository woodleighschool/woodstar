package handlers

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/api/middleware"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/munki/mdp"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// RegisterMunki mounts Munki host state, software, package, and distribution
// point endpoints.
func RegisterMunki(
	api huma.API,
	router chi.Router,
	authService *auth.Service,
	hostState *munki.Store,
	hostStore *hosts.Store,
	softwareStore *munkisoftware.Store,
	packageService *munki.PackageService,
	objects *storage.ObjectStore,
	storageStore storage.Backend,
	distributionStore *mdp.Store,
	logger *slog.Logger,
) {
	registerHostMunkiState(api, hostState, hostStore, logger)
	registerMunkiSoftware(api, softwareStore, packageService, objects, storageStore, logger)
	registerMunkiSoftwareIconContent(
		router.With(middleware.RequireHTTPAuth(authService)),
		softwareStore,
		objects,
		storageStore,
		logger,
	)
	registerMunkiPackages(api, packageService, objects, storageStore, logger)
	registerMunkiDistributionPoints(api, distributionStore, logger)
}
