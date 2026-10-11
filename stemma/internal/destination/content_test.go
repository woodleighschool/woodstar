package destination

import (
	"os"
	"testing"

	"github.com/woodleighschool/stemma/plugin"
	"github.com/woodleighschool/woodstar/stemma/internal/destination/api"
)

func TestReconciliationRequestsContentOnlyWhenPublicationNeedsIt(t *testing.T) {
	fixture, connection := serveFixture(t)
	installer := installerFixture(t, "Example.pkg", "installer bytes", "1.0")
	icon := iconFixture(t, 10)
	request := plugin.ReconcileRequest[api.Config]{Method: "apply", Prepared: true, Config: connection, Identity: plugin.Identity{Resource: plugin.ResourceReference{Kind: "MacSoftware", Name: "Example"}}, Artifact: installer, Inputs: map[string]plugin.Artifact{"icon": icon}}
	planning := request
	planning.Method = "plan"
	planning.Artifact.Path = ""
	planning.Inputs = map[string]plugin.Artifact{"icon": icon}
	unleasedIcon := icon
	unleasedIcon.Path = ""
	planning.Inputs["icon"] = unleasedIcon
	response, err := Handle(t.Context(), planning)
	if err != nil || response.Content != nil || len(response.Changes) == 0 || fixture.writes() != 0 {
		t.Fatalf("initial plan=%+v err=%v writes=%d", response, err, fixture.writes())
	}
	if _, err := Handle(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	// Even an unavailable local copy must not affect an existing publication.
	if err := os.Rename(installer.Path, installer.Path+".saved"); err != nil {
		t.Fatal(err)
	}
	request.Artifact.Path = ""
	icon.Path = ""
	request.Inputs["icon"] = icon
	uploads := fixture.uploadCount()
	before := fixture.writes()
	for _, method := range []string{"validate", "plan", "apply"} {
		request.Method = method
		response, err := Handle(t.Context(), request)
		if err != nil || response.Content != nil || len(response.Changes) != 0 || fixture.writes() != before {
			t.Fatalf("%s response=%+v err=%v writes=%d", method, response, err, fixture.writes())
		}
	}
	setPkginfo(t, &request, `{"description":"Changed"}`)
	response, err = Handle(t.Context(), request)
	if err != nil || response.Content != nil || fixture.title()["description"] != "Changed" || fixture.uploadCount() != uploads {
		t.Fatalf("metadata response=%+v err=%v", response, err)
	}
	request.Method = "plan"
	if response, err = Handle(t.Context(), request); err != nil || len(response.Changes) != 0 {
		t.Fatalf("plan=%+v err=%v", response, err)
	}
	// The package vanishes after plan. Apply must ask before making any writes.
	fixture.mu.Lock()
	fixture.packages = nil
	fixture.pkg = nil
	fixture.mu.Unlock()
	before = fixture.writes()
	request.Method = "apply"
	response, err = Handle(t.Context(), request)
	if err != nil || response.Content == nil || !response.Content.Artifact || len(response.Content.Inputs) != 0 || fixture.writes() != before {
		t.Fatalf("race response=%+v err=%v writes=%d", response, err, fixture.writes())
	}
	if err := os.Rename(installer.Path+".saved", installer.Path); err != nil {
		t.Fatal(err)
	}
	request.Artifact.Path = installer.Path
	response, err = Handle(t.Context(), request)
	if err != nil || response.Content != nil || fixture.uploadCount() != uploads+1 {
		t.Fatalf("retry response=%+v err=%v", response, err)
	}
}
