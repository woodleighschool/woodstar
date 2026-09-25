package destination

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/stemma/internal/destination/api"
)

func TestPlanRejectsPaddedArtifactFilename(t *testing.T) {
	declared := metadata{name: "Example", version: "1.0", installer: plugin.Artifact{Filename: " Example.pkg", Version: "1.0", Size: 1, SHA256: strings.Repeat("a", 64)}}
	if err := json.Unmarshal(json.RawMessage(`{"version":"1.0","installer_type":"pkg"}`), &declared.pkg); err != nil {
		t.Fatal(err)
	}
	if _, err := plan(declared, api.Observation{}); err == nil {
		t.Fatal("accepted a filename that changes during upload")
	}
}

func TestCreationReportsInitialMetadataAndAppliesIt(t *testing.T) {
	fixture, connection := serveFixture(t)
	fixture.labels = map[string]int64{"Staff": 7}
	request := plugin.ReconcileRequest[api.Config]{
		Method: "plan", Prepared: true, Config: connection,
		Identity: plugin.Identity{Resource: plugin.ResourceReference{Kind: "MacSoftware", Name: "Example"}},
		Metadata: json.RawMessage(`{
			"pkginfo":{"name":"Example","version":"1.0","installer_type":"nopkg","display_name":"Example","description":" Managed description ","unattended_install":false,"supported_architectures":[]},
			"targets":{"include":[{"label_name":"Staff","actions":["managed_installs"]}]}
		}`),
	}
	planned, err := Handle(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(planned.Changes) != 2 || fixture.writes() != 0 {
		t.Fatalf("initial plan=%+v writes=%d", planned.Changes, fixture.writes())
	}
	initial := make(map[string]map[string]json.RawMessage)
	for _, change := range planned.Changes {
		if change.Action != "create" || len(change.Before) != 0 || change.Kind != "metadata" {
			t.Fatalf("initial change=%+v", change)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(change.After, &fields); err != nil {
			t.Fatal(err)
		}
		initial[change.Field] = fields
	}
	for field, want := range map[string]string{"name": `"Example"`, "display_name": `""`, "description": `"Managed description"`} {
		if got := string(initial["software"][field]); got != want {
			t.Errorf("software.%s=%s, want %s", field, got, want)
		}
	}
	if !strings.Contains(string(initial["software"]["targets"]), `"label_id":7`) {
		t.Fatalf("initial targets=%s", initial["software"]["targets"])
	}
	for field, want := range map[string]string{"version": `"1.0"`, "installer_type": `"nopkg"`, "unattended_install": `false`, "supported_architectures": `[]`} {
		if got := string(initial["package"][field]); got != want {
			t.Errorf("package.%s=%s, want %s", field, got, want)
		}
	}
	for _, field := range []string{"on_demand", "installer_object_id"} {
		if _, found := initial["package"][field]; found {
			t.Errorf("initial package includes undeclared field %s", field)
		}
	}

	request.Method = "apply"
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if title := fixture.title(); title["description"] != " Managed description " || !strings.Contains(string(raw(title["targets"])), `"label_id":7`) {
		t.Fatalf("created software lost metadata or targets: %v", title)
	}
	if pkg := fixture.version("1.0"); pkg["installer_type"] != "nopkg" {
		t.Fatalf("created package=%v", pkg)
	}
	assertFixtureConverged(t, fixture, request)

	request.Method = "plan"
	request.Metadata = json.RawMessage(`{"pkginfo":{"name":"Example","version":"1.0","installer_type":"nopkg","description":"Updated","unattended_install":true}}`)
	updated, err := Handle(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if got := changedFields(updated.Changes); !slices.Equal(got, []string{"package.unattended_install", "software.description"}) {
		t.Fatalf("existing publication lost scalar changes: %v", got)
	}
}
