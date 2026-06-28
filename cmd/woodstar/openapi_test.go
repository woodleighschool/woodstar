package main

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/buildinfo"
)

func TestBuildSchemaAPIForOpenAPICommand(t *testing.T) {
	payload, err := api.BuildSchemaAPI(buildinfo.Version, schemaDependencies()).OpenAPI().YAML()
	if err != nil {
		t.Fatalf("encode openapi: %v", err)
	}
	if len(payload) == 0 {
		t.Fatal("openapi payload is empty")
	}
}
