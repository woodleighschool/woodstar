package api

import (
	"github.com/danielgtaylor/huma/v2"
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
	return cfg
}

// BuildSchemaAPI builds the admin API for OpenAPI schema generation only.
// Stores are nil; handlers are never invoked.
func BuildSchemaAPI(version string) huma.API {
	r := chi.NewRouter()
	return Mount(r, Dependencies{Version: version})
}
