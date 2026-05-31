package handlers

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/humaschema"
)

func testHumaConfig() huma.Config {
	cfg := huma.DefaultConfig("test", "test")
	cfg.Components = &huma.Components{
		Schemas: huma.NewMapRegistry("#/components/schemas/", humaschema.WoodstarSchemaNamer),
	}
	return cfg
}
