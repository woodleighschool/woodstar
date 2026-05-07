// Package admin provides the Huma-backed browser/admin HTTP API.
package admin

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/danielgtaylor/huma/v2/autopatch"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/models"
	"github.com/woodleighschool/woodstar/internal/transport/admin/handlers"
)

// Dependencies contains the services and stores used by the admin HTTP API.
type Dependencies struct {
	DB             *database.DB
	Version        string
	Started        time.Time
	AuthService    *auth.Service
	HostStore      *models.HostStore
	DeviceMappings *models.DeviceMappingStore
	SecretStore    *models.SecretStore
	SoftwareStore  *models.SoftwareStore
	LabelStore     *models.LabelStore
}

// Mount attaches public and authenticated admin API routes to r.
func Mount(r chi.Router, deps Dependencies) huma.API {
	api := humachi.New(r, Config(deps.Version))
	protected := huma.NewGroup(api)
	protected.UseMiddleware(RequireAuth(api, deps.AuthService))

	handlers.RegisterSystem(api, deps.DB, deps.Version, deps.Started)
	handlers.RegisterPublicAuth(api, deps.AuthService)
	handlers.RegisterProtectedAuth(protected, deps.AuthService)
	handlers.RegisterUsers(protected, deps.AuthService)
	handlers.RegisterHosts(protected, deps.HostStore, deps.DeviceMappings, deps.SoftwareStore, deps.LabelStore)
	handlers.RegisterSoftware(protected, deps.SoftwareStore)
	handlers.RegisterLabels(protected, deps.LabelStore)
	handlers.RegisterSecrets(protected, deps.SecretStore)

	// Synthesise PATCH for resources that expose GET + PUT.
	// https://huma.rocks/features/auto-patch/.
	autopatch.AutoPatch(api)

	return api
}

// BuildAPI returns the admin API without starting the server.
func BuildAPI(version string) huma.API {
	r := chi.NewRouter()
	return Mount(r, Dependencies{
		Version:        version,
		Started:        time.Now().UTC(),
		AuthService:    auth.NewService(nil, nil),
		HostStore:      models.NewHostStore(nil),
		DeviceMappings: models.NewDeviceMappingStore(nil),
		SecretStore:    models.NewSecretStore(nil),
		SoftwareStore:  models.NewSoftwareStore(nil),
		LabelStore:     models.NewLabelStore(nil),
	})
}
