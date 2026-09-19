package destination

import (
	"encoding/json"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/woodleighschool/stemma/plugin"
)

func TestRetentionCoversTheWholeRemoteFamily(t *testing.T) {
	fixture, connection := serveFixture(t)
	fixture.labels = map[string]int64{"Staff": 7}
	// An operator pinned 1.1; omitted targets leave that pin unchanged.
	fixture.software = map[string]any{"id": json.Number("1"), "name": "Example", "targets": map[string]any{
		"include": []any{map[string]any{"label_id": 7, "package": map[string]any{"strategy": "specific", "package_id": 22}, "actions": []any{"managed_installs"}}},
		"exclude": []any{},
	}}
	// The plugin published none of these. Creation orders the family, then id:
	// 1.2 and 1.1 share a time, and the migrated 2.0 has the highest id and
	// version yet is among the oldest.
	for _, seeded := range []struct {
		id      int
		version string
		hour    int
	}{{20, "0.5", 6}, {21, "1.0", 9}, {22, "1.1", 10}, {23, "1.2", 10}, {24, "0.9", 11}, {25, "2.0", 8}, {26, "0.8", 7}} {
		fixture.setPackage(map[string]any{"id": json.Number(strconv.Itoa(seeded.id)), "version": seeded.version, "installer_type": "nopkg", "created_at": time.Date(2026, time.March, 1, seeded.hour, 0, 0, 0, time.UTC).Format(time.RFC3339)})
	}
	fixture.packages[23]["requires"] = []any{map[string]any{"software_id": 1, "package_id": 21}}
	fixture.packages[23]["update_for"] = []any{map[string]any{"software_id": 1, "package_id": 20}}
	// Another title requires 2.0, which only the repository's foreign key knows.
	fixture.held = map[int64]bool{25: true}
	metadata := map[string]any{"pkginfo": map[string]any{"installer_type": "nopkg", "version": "3.0"}, "retention": plugin.Retention{Keep: 3}}
	request := plugin.ReconcileRequest{Method: "plan", Prepared: true, Config: connection, Identity: plugin.Identity{Software: "Example"}, Metadata: raw(metadata)}
	run := func(method string) []string {
		t.Helper()
		request.Method = method
		response, err := Handle(t.Context(), request)
		if err != nil {
			t.Fatal(err)
		}
		return prunedVersions(t, response.Changes)
	}
	family := []string{"0.5", "0.8", "0.9", "1.0", "1.1", "1.2", "2.0"}

	// 3.0 keeps the two newest others, 0.9 and 1.2. Of the rest 1.1 is pinned and
	// 1.0 and 0.5 are referenced by 1.2.
	if planned := run("plan"); !slices.Equal(planned, []string{"2.0", "0.8"}) || fixture.writes() != 0 || !slices.Equal(fixture.versions(), family) {
		t.Fatalf("planned=%v writes=%d versions=%v", planned, fixture.writes(), fixture.versions())
	}
	if pruned := run("apply"); !slices.Equal(pruned, []string{"0.8"}) || !slices.Equal(fixture.versions(), []string{"0.5", "0.9", "1.0", "1.1", "1.2", "2.0", "3.0"}) {
		t.Fatalf("pruned=%v versions=%v", pruned, fixture.versions())
	}
	// The refused delete is neither reported nor an error, however often it recurs.
	if pruned := run("apply"); len(pruned) != 0 {
		t.Fatalf("repeated pruning=%v", pruned)
	}

	// Declared targets follow the latest package, which releases the pinned one.
	metadata["targets"] = map[string]any{"include": []any{map[string]any{"label_name": "Staff", "actions": []string{"managed_installs"}}}}
	request.Metadata = raw(metadata)
	if planned := run("plan"); !slices.Equal(planned, []string{"1.1", "2.0"}) || fixture.version("1.1") == nil {
		t.Fatalf("planned=%v versions=%v", planned, fixture.versions())
	}
	if pruned := run("apply"); !slices.Equal(pruned, []string{"1.1"}) || !slices.Equal(fixture.versions(), []string{"0.5", "0.9", "1.0", "1.2", "2.0", "3.0"}) {
		t.Fatalf("pruned=%v versions=%v", pruned, fixture.versions())
	}
}

// prunedVersions reads the versions a run reported under retention, in order.
func prunedVersions(t *testing.T, changes []plugin.Change) []string {
	t.Helper()
	var versions []string
	for _, change := range changes {
		if change.Kind != "retention" {
			continue
		}
		var version string
		if err := json.Unmarshal(change.Before, &version); err != nil || change.Field != "package" || change.Action != "delete" {
			t.Fatalf("retention change=%+v error=%v", change, err)
		}
		versions = append(versions, version)
	}
	return versions
}
