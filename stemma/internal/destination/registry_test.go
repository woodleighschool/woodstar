package destination

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/woodleighschool/stemma/plugin"
)

func TestRegistryPublishesAndEnforcesTypedConnection(t *testing.T) {
	registry := plugin.New("fixture", "1")
	if err := Register(registry); err != nil {
		t.Fatal(err)
	}
	operation := registry.Descriptor().Operations[0]
	for _, fragment := range []string{`"required":["url","api_key"]`, `"additionalProperties":false`, `"writeOnly":true`, `"description":"HTTPS origin`} {
		if !strings.Contains(string(operation.ConfigSchema), fragment) {
			t.Fatalf("connection contract missing %s: %s", fragment, operation.ConfigSchema)
		}
	}
	call := func(method, config string) error {
		t.Helper()
		request := plugin.ReconcileRequest[json.RawMessage]{Identity: plugin.Identity{Project: "fixture", Destination: "woodstar", Resource: plugin.ResourceReference{APIVersion: "stemma/v1alpha1", Kind: "MacSoftware", Name: "app"}}, Config: json.RawMessage(config), Metadata: json.RawMessage(`{"pkginfo":{"name":"Fixture","version":"1","installer_type":"nopkg"}}`)}
		data, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		_, err = registry.Handle(t.Context(), plugin.Request{Protocol: plugin.ProtocolVersion, Method: method, Operation: operation.Name, Input: data})
		return err
	}
	// Validation checks desired state without connecting, so it carries no connection.
	if err := call("validate", ""); err != nil {
		t.Fatalf("validation needed a connection: %v", err)
	}
	for _, test := range []struct{ config, message string }{
		{`{"url":"https://unused.invalid"}`, "api_key"},
		{`{"url":"https://unused.invalid","api_key":"test-key","unknown":true}`, "unknown"},
		{`{"url":"https://unused.invalid/api","api_key":"test-key"}`, "HTTPS origin"},
	} {
		if err := call("plan", test.config); err == nil || !strings.Contains(err.Error(), test.message) {
			t.Fatalf("config %s: %v", test.config, err)
		}
	}
	for range 5 {
		other := plugin.New("fixture", "1")
		if err := Register(other); err != nil {
			t.Fatal(err)
		}
		a, _ := json.Marshal(registry.Descriptor())
		b, _ := json.Marshal(other.Descriptor())
		if !bytes.Equal(a, b) {
			t.Fatal("schema changed between registrations")
		}
	}
	var metadata struct {
		Properties map[string]struct {
			Description string `json:"description"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(operation.MetadataSchema, &metadata); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"targets", "pkginfo", "retention"} {
		if metadata.Properties[field].Description == "" {
			t.Fatalf("missing hover description for %s", field)
		}
	}
}
