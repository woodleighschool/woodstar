package main

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/api"
	apihandlers "github.com/woodleighschool/woodstar/internal/api/handlers"
	"github.com/woodleighschool/woodstar/internal/buildinfo"
)

func TestBuildSchemaAPIWithZeroValueWiring(t *testing.T) {
	payload, err := api.BuildSchemaAPI(buildinfo.Version, apihandlers.Dependencies{}).OpenAPI().YAML()
	if err != nil {
		t.Fatalf("encode openapi: %v", err)
	}
	if len(payload) == 0 {
		t.Fatal("openapi payload is empty")
	}
}
