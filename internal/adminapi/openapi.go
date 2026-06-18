package adminapi

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/humaschema"
)

func init() {
	//nolint:reassign // huma exposes array nullability only as a package global
	huma.DefaultArrayNullable = false
}

// humaConfig returns the Huma config shared by serve and openapi.
func humaConfig(version string) huma.Config {
	cfg := huma.DefaultConfig("Woodstar API", version)
	cfg.Info.Description = "Typed admin and frontend API."
	cfg.Info.License = &huma.License{Name: "Apache-2.0"}

	// Don't emit docs or schema routes, useless for us.
	cfg.OpenAPIPath = ""
	cfg.DocsPath = ""
	cfg.SchemasPath = ""
	cfg.CreateHooks = nil

	cfg.Components = &huma.Components{
		Schemas: huma.NewMapRegistry("#/components/schemas/", humaschema.WoodstarSchemaNamer),
		SecuritySchemes: map[string]*huma.SecurityScheme{
			"cookieAuth": {
				Type: "apiKey",
				In:   "cookie",
				Name: "woodstar_session",
			},
		},
	}

	return cfg
}

// BuildSchemaAPI builds the admin API for OpenAPI schema generation only. It
// runs the same admin registrars the server uses; callers pass registrars built
// with nil dependencies so no database is touched and no handler is invoked.
func BuildSchemaAPI(version string, admin []AdminRegistrar) huma.API {
	r := chi.NewRouter()
	humaAPI := humachi.New(r, humaConfig(version))
	registerAdminRoutes(r, humaAPI, Dependencies{Version: version, Admin: admin})
	return humaAPI
}
