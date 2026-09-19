package destination

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

func TestTargetsResolveLabelNamesOnTheInstance(t *testing.T) {
	fixture, connection := serveFixture(t)
	// The repository lists labels by name and searches by substring, so "Staff"
	// finds "All Staff" first.
	fixture.labels = map[string]int64{"Staff": 7, "All Staff": 8, "Loaners": 9}
	request := plugin.ReconcileRequest{Method: "apply", Prepared: true, Config: connection, Identity: plugin.Identity{Software: "Policy"}}
	declare := func(description, targets string) {
		request.Metadata = json.RawMessage(`{"pkginfo":{"installer_type":"nopkg","version":"1","description":"` + description + `"}` + targets + `}`)
	}
	apply := func() software.Targets {
		t.Helper()
		if _, err := Handle(t.Context(), request); err != nil {
			t.Fatal(err)
		}
		fixture.mu.Lock()
		defer fixture.mu.Unlock()
		var published software.Targets
		if err := json.Unmarshal(raw(fixture.software["targets"]), &published); err != nil {
			t.Fatal(err)
		}
		return published
	}
	targetsJSON := `,"targets":{"include":[{"label_name":"Staff","actions":["optional_installs","managed_updates"]}],"exclude":[{"label_name":"Loaners"}]}`
	want := software.Targets{
		Include: []software.Include{{LabelID: 7, Package: software.PackageSelector{Strategy: software.PackageLatest}, Actions: []software.Action{software.ActionOptionalInstalls, software.ActionManagedUpdates}}},
		Exclude: []targeting.LabelRef{{LabelID: 9}},
	}
	declare("Managed", targetsJSON)
	if got := apply(); !reflect.DeepEqual(got, want) {
		t.Fatalf("resolved targets=%+v, want %+v", got, want)
	}
	assertFixtureConverged(t, fixture, request)

	// Declared lists replace what the repository holds, including an operator's pin.
	operator := map[string]any{"include": []any{
		map[string]any{"label_id": 7, "package": map[string]any{"strategy": "specific", "package_id": 1}, "actions": []any{"optional_installs", "managed_updates"}},
		map[string]any{"label_id": 8, "package": map[string]any{"strategy": "latest"}, "actions": []any{"managed_installs"}},
	}, "exclude": []any{map[string]any{"label_id": 9}}}
	fixture.mu.Lock()
	fixture.software["targets"] = operator
	fixture.mu.Unlock()
	request.Method = "plan"
	planned, err := Handle(t.Context(), request)
	if err != nil || len(planned.Changes) != 1 || planned.Changes[0].Kind != "targets" || planned.Changes[0].Field != "software.targets" {
		t.Fatalf("operator drift plan=%+v error=%v", planned.Changes, err)
	}
	request.Method = "apply"
	if got := apply(); !reflect.DeepEqual(got, want) {
		t.Fatalf("restored targets=%+v, want %+v", got, want)
	}

	// Omitted targets are left unchanged, whatever else the run changes.
	fixture.mu.Lock()
	fixture.software["targets"] = operator
	fixture.mu.Unlock()
	declare("Managed elsewhere", "")
	if got := apply(); len(got.Include) != 2 || len(got.Exclude) != 1 || fixture.title()["description"] != "Managed elsewhere" {
		t.Fatalf("omitted targets=%+v software=%v", got, fixture.title())
	}
	// Empty objects preserve omitted lists; each declared list replaces itself.
	declare("Managed elsewhere", `,"targets":{}`)
	if got := apply(); len(got.Include) != 2 || len(got.Exclude) != 1 {
		t.Fatalf("empty targets changed lists: %+v", got)
	}
	declare("Managed elsewhere", `,"targets":{"include":[]}`)
	if got := apply(); len(got.Include) != 0 || len(got.Exclude) != 1 || got.Exclude[0].LabelID != 9 {
		t.Fatalf("clearing includes changed exclusions: %+v", got)
	}
	declare("Managed elsewhere", `,"targets":{"exclude":[]}`)
	if got := apply(); len(got.Include)+len(got.Exclude) != 0 {
		t.Fatalf("cleared targets=%+v", got)
	}
	assertFixtureConverged(t, fixture, request)
}

func TestTargetsRejectLabelsTheInstanceDoesNotName(t *testing.T) {
	fixture, connection := serveFixture(t)
	fixture.labels = map[string]int64{"Staff": 7, "All Staff": 8}
	for _, test := range []struct{ targets, want string }{
		{`{"include":[{"label_name":"Students","actions":["managed_installs"]}]}`, `targets: unknown label "Students"`},
		// The search ignores case, but only the exact name identifies a label.
		{`{"exclude":[{"label_name":"staff"}]}`, `targets: unknown label "staff"`},
	} {
		for _, method := range []string{"plan", "apply"} {
			request := plugin.ReconcileRequest{Method: method, Prepared: true, Config: connection, Identity: plugin.Identity{Software: "Policy"}, Metadata: json.RawMessage(`{"pkginfo":{"installer_type":"nopkg","version":"1"},"targets":` + test.targets + `}`)}
			if _, err := Handle(t.Context(), request); err == nil || !strings.Contains(err.Error(), test.want) || fixture.writes() != 0 {
				t.Fatalf("%s %s: error=%v writes=%d", method, test.targets, err, fixture.writes())
			}
		}
	}
}
