package admin

import "github.com/danielgtaylor/huma/v2"

// Config returns the Huma config shared by serve and openapi.
func Config(version string) huma.Config {
	cfg := huma.DefaultConfig("Woodstar API", version)
	cfg.Info.Description = "Typed admin and frontend API."
	cfg.Info.License = &huma.License{Name: "Apache-2.0"}
	cfg.DocsPath = "/api/docs"
	cfg.OpenAPIPath = "/api/openapi"
	cfg.SchemasPath = "/api/schemas"
	cfg.Servers = []*huma.Server{{URL: "/"}}
	return cfg
}
