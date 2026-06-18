package main

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/adminapi"
	"github.com/woodleighschool/woodstar/internal/buildinfo"
)

func TestBuildSchemaAPIWithZeroValueWiring(t *testing.T) {
	payload, err := adminapi.BuildSchemaAPI(buildinfo.Version, (&wiring{}).adminRegistrars()).OpenAPI().YAML()
	if err != nil {
		t.Fatalf("encode openapi: %v", err)
	}
	if len(payload) == 0 {
		t.Fatal("openapi payload is empty")
	}
}
