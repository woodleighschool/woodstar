package osquery

import "testing"

func TestDetailQueryRegistryIsComplete(t *testing.T) {
	queries := DetailQueries()
	wantNames := []string{
		"os_version",
		"system_info",
		"osquery_info",
		"orbit_info",
		"software_macos",
		"software_vscode_extensions",
		"software_jetbrains_plugins",
		"software_go_binaries",
		"software_python_packages",
		"software_macos_codesign",
		"software_macos_executable_sha256",
	}
	if len(queries) != len(wantNames) {
		t.Fatalf("len(DetailQueries()) = %d, want %d", len(queries), len(wantNames))
	}

	for _, name := range wantNames {
		query, ok := queries[name]
		if !ok {
			t.Fatalf("missing query %q", name)
		}
		if query.SQL == "" {
			t.Fatalf("%s SQL is empty", name)
		}
		if query.Ingest == nil {
			t.Fatalf("%s ingest func is nil", name)
		}
	}
}

func TestDetailQueriesDue(t *testing.T) {
	if got := detailQueriesDue(nil); len(got.Queries) != 11 {
		t.Fatalf("nil timestamp returned %d queries, want 11", len(got.Queries))
	}
	if got := detailQueriesDue(nil); got.Discovery["orbit_info"] == "" ||
		got.Discovery["software_vscode_extensions"] == "" ||
		got.Discovery["software_jetbrains_plugins"] == "" ||
		got.Discovery["software_go_binaries"] == "" ||
		got.Discovery["software_python_packages"] == "" ||
		got.Discovery["software_macos_codesign"] == "" ||
		got.Discovery["software_macos_executable_sha256"] == "" {
		t.Fatalf("missing optional detail query discovery: %#v", got.Discovery)
	}
}
