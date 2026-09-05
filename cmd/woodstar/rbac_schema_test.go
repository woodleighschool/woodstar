package main

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/woodleighschool/woodstar/internal/rbac"
)

func TestOpenAPIResourceCatalogue(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(buildOpenAPI("test").OpenAPI())
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Components struct {
			Schemas map[string]struct {
				Type string `json:"type"`
				Enum []any  `json:"enum"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatal(err)
	}
	schema, exists := document.Components.Schemas["AuthzResource"]
	want := rbac.ResourceSchema()
	if !exists || schema.Type != "string" || !reflect.DeepEqual(schema.Enum, want.Enum) {
		t.Fatalf("resource catalogue schema = %+v, want %+v", schema, want)
	}
}
