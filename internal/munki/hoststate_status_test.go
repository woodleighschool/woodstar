package munki

import "testing"

func TestHostStatusFromInfoRows(t *testing.T) {
	status, ok := HostStatusFromInfoRows(42, []map[string]string{{
		"version":          "7.1.2.5700",
		"manifest_name":    "site_default",
		"success":          "true",
		"errors":           "first error; second error",
		"warnings":         "first warning\nsecond warning",
		"problem_installs": "Broken App;",
		"start_time":       "2026-05-31 19:23:00 +1000",
		"end_time":         "2026-05-31 19:24:14 +1000",
	}})
	if !ok {
		t.Fatal("HostStatusFromInfoRows returned false")
	}
	if status.HostID != 42 || status.Version != "7.1.2.5700" || status.ManifestName != "site_default" {
		t.Fatalf("status identity = %+v, want host/version/manifest", status)
	}
	if status.Success == nil || !*status.Success {
		t.Fatalf("success = %v, want true", status.Success)
	}
	if !sameStrings(status.Errors, []string{"first error", "second error"}) {
		t.Fatalf("errors = %#v", status.Errors)
	}
	if !sameStrings(status.Warnings, []string{"first warning", "second warning"}) {
		t.Fatalf("warnings = %#v", status.Warnings)
	}
	if !sameStrings(status.ProblemInstalls, []string{"Broken App"}) {
		t.Fatalf("problem installs = %#v", status.ProblemInstalls)
	}
}

func TestHostStatusFromInfoRowsMissing(t *testing.T) {
	if _, ok := HostStatusFromInfoRows(42, nil); ok {
		t.Fatal("HostStatusFromInfoRows returned true for no rows")
	}
}

func TestItemsFromInstallRows(t *testing.T) {
	got := ItemsFromInstallRows(7, []map[string]string{
		{
			"name":              "GoogleChrome",
			"installed":         "true",
			"installed_version": "148.0",
			"end_time":          "2026-05-31 19:24:14 +1000",
		},
		{
			"name":      "Optional App",
			"installed": "false",
		},
		{"installed": "true"},
	})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].HostID != 7 || got[0].Name != "GoogleChrome" || !got[0].Installed ||
		got[0].InstalledVersion != "148.0" {
		t.Fatalf("first item = %+v", got[0])
	}
	if got[1].Installed {
		t.Fatalf("second installed = true, want false")
	}
}

func sameStrings(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
