package adminapi

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// Mount attaches public and authenticated admin API routes to r.
func Mount(r chi.Router, deps Dependencies) {
	humaAPI := humachi.New(r, humaConfig(deps.Version))
	registerAdminRoutes(r, humaAPI, deps)
}

func registerAdminRoutes(r chi.Router, humaAPI huma.API, deps Dependencies) {
	protected := huma.NewGroup(humaAPI)
	protected.UseMiddleware(RequireAuth(humaAPI, deps.AuthService))
	ordinary := huma.NewGroup(protected)
	ordinary.UseModifier(RequireAdminForMutations(humaAPI))
	sensitive := huma.NewGroup(protected)
	sensitive.UseModifier(requireAdminForAll(humaAPI))

	groups := AdminGroups{
		Public:    humaAPI,
		Protected: protected,
		Ordinary:  ordinary,
		Sensitive: sensitive,
		Router:    r,
	}
	for _, register := range deps.Admin {
		register(groups)
	}
}
