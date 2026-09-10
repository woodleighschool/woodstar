package destination

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/woodleighschool/stemma/plugin"
)

func TestNativeDerivationAndOwnedRetention(t *testing.T) {
	fixture := &apiFixture{objects: map[int64][]byte{}, names: map[int64]string{}}
	remote := testClient(t, fixture)
	fixture.origin = remote.config.URL
	request := plugin.ReconcileRequest{
		Method: "apply", Prepared: true, Config: raw(remote.config),
		Identity: plugin.Identity{Software: "Example"},
		Subjects: map[string]plugin.SubjectSelector{"app": {Kind: "app", Path: "Example.app"}},
		Facts:    plugin.Facts{Subjects: []plugin.Subject{{Kind: "app", Path: "Example.app", App: &plugin.AppFacts{BundleID: "test.example", Version: "1.0", Build: "100", MinimumOS: "14.0"}}}},
		Metadata: raw(map[string]any{"derive": map[string]any{"app": map[string]any{"subject": "app"}}, "retention": plugin.Retention{Keep: 1}}),
	}
	setNativeInstaller(t, &request, "first installer", "1.0")
	call := func() binding {
		t.Helper()
		response, err := Handle(t.Context(), request)
		if err != nil {
			t.Fatal(err)
		}
		request.Binding = response.Binding
		var saved binding
		if err := json.Unmarshal(response.Binding, &saved); err != nil {
			t.Fatal(err)
		}
		return saved
	}
	first := call()
	installs, _ := fixture.pkg["installs"].([]any)
	if len(installs) != 1 {
		t.Fatalf("installs=%v", installs)
	}
	app, _ := installs[0].(map[string]any)
	if app["path"] != "/Applications/Example.app" || app["bundle_identifier"] != "test.example" || app["bundle_version"] != "100" {
		t.Fatalf("derived application=%v", app)
	}

	request.Metadata = raw(map[string]any{"pkginfo": map[string]any{"version": "1.1"}, "retention": plugin.Retention{Keep: 1}})
	request.Facts.Subjects[0].App.MinimumOS = ""
	second := call()
	if second.Version != "1.1" || second.PackageID != first.PackageID || second.Publications.Sequence != 1 || fixture.uploadCount() != 1 || fixture.pkg["minimum_os_version"] != nil {
		t.Fatalf("metadata-only change: binding=%+v package=%v uploads=%d", second, fixture.pkg, fixture.uploadCount())
	}
	fixture.pkg["minimum_os_version"] = "15.0"
	request.Metadata = raw(map[string]any{"pkginfo": map[string]any{"version": "1.1"}, "unmanaged": []string{"pkginfo.minimum_os_version"}, "retention": plugin.Retention{Keep: 1}})
	call()
	if fixture.pkg["minimum_os_version"] != "15.0" {
		t.Fatal("unmanaged value was changed")
	}

	setNativeInstaller(t, &request, "second installer", "2.0")
	request.Metadata = raw(map[string]any{"retention": plugin.Retention{Keep: 1}, "targets": map[string]any{"include": []any{map[string]any{"label_id": 1, "package": map[string]any{"strategy": "specific", "package_id": first.PackageID}, "actions": []string{"optional_installs"}}}}})
	third := call()
	if third.Publications.Sequence != 2 || len(fixture.packages) != 2 {
		t.Fatalf("pinned history: %+v packages=%d", third, len(fixture.packages))
	}
	request.Metadata = raw(map[string]any{"retention": plugin.Retention{Keep: 1}, "targets": map[string]any{"include": []any{}}})
	pruned := call()
	if len(pruned.Packages) != 1 || len(fixture.packages) != 1 || fixture.packages[pruned.PackageID] == nil {
		t.Fatalf("retention=%+v packages=%v", pruned, fixture.packages)
	}
	if fixture.uploadCount() != 2 {
		t.Fatal("metadata change replayed an upload")
	}

	request.Binding = nil
	before := fixture.writes()
	if _, err := Handle(t.Context(), request); err == nil || fixture.writes() != before {
		t.Fatalf("lost binding: %v", err)
	}
}

func setNativeInstaller(t *testing.T, request *plugin.ReconcileRequest, body, version string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "Example.dmg")
	if err := os.WriteFile(path, []byte(body), 0o400); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(body))
	request.Facts.Subjects[0].App.Version = version
	request.Artifact = plugin.Artifact{Path: path, Filename: "Example.dmg", Format: "dmg", Version: version, Size: int64(len(body)), SHA256: hex.EncodeToString(hash[:])}
}
