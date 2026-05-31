package api

import (
	"reflect"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// humaConfig returns the Huma config shared by serve and openapi.
func humaConfig(version string) huma.Config {
	cfg := huma.DefaultConfig("Woodstar API", version)
	cfg.Info.Description = "Typed admin and frontend API."
	cfg.Info.License = &huma.License{Name: "Apache-2.0"}
	cfg.DocsPath = "/api/docs"
	cfg.OpenAPIPath = "/api/openapi"
	cfg.SchemasPath = "/api/schemas"

	cfg.Components = &huma.Components{
		Schemas: huma.NewMapRegistry("#/components/schemas/", woodstarSchemaNamer),
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

func woodstarSchemaNamer(t reflect.Type, hint string) string {
	t = derefType(t)
	if t.Name() == "HostState" && t.PkgPath() == "github.com/woodleighschool/woodstar/internal/munki" {
		return "MunkiHostState"
	}
	return huma.DefaultSchemaNamer(t, hint)
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// BuildSchemaAPI builds the admin API for OpenAPI schema generation only.
// Stores are nil; handlers are never invoked.
func BuildSchemaAPI(version string) huma.API {
	r := chi.NewRouter()
	humaAPI := humachi.New(r, humaConfig(version))
	registerAdminRoutes(r, humaAPI, Dependencies{})
	return humaAPI
}
