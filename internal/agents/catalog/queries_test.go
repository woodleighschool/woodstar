package catalog

import (
	"strings"
	"testing"
	"time"
)

func TestDetailQueryRegistryIsComplete(t *testing.T) {
	queries := DetailQueries()
	wantNames := []string{
		"os_version",
		"system_info",
		"osquery_info",
		"osquery_flags",
		"orbit_info",
		"uptime",
		"root_disk",
		"primary_interface",
		"users",
		"batteries",
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
	}
}

func TestDetailQueriesDue(t *testing.T) {
	if got := DetailQueriesDue(nil, ""); len(got.Queries) != 17 {
		t.Fatalf("nil timestamp returned %d queries, want 17", len(got.Queries))
	}
	if got := DetailQueriesDue(nil, ""); got.Discovery["orbit_info"] == "" ||
		got.Discovery["software_vscode_extensions"] == "" ||
		got.Discovery["software_jetbrains_plugins"] == "" ||
		got.Discovery["software_go_binaries"] == "" ||
		got.Discovery["software_python_packages"] == "" ||
		got.Discovery["software_macos_codesign"] == "" ||
		got.Discovery["software_macos_executable_sha256"] == "" {
		t.Fatalf("missing optional detail query discovery: %#v", got.Discovery)
	}
}

func TestDetailQueriesDueWhenHashChanges(t *testing.T) {
	now := time.Now()
	if got := DetailQueriesDue(&now, DetailQueryHash()); len(got.Queries) != 0 {
		t.Fatalf("fresh matching hash returned %d queries, want 0", len(got.Queries))
	}
	if got := DetailQueriesDue(&now, "old-hash"); len(got.Queries) == 0 {
		t.Fatal("fresh stale hash returned no queries")
	}
}

func TestRootDiskQueryUsesOrbitDiskSpace(t *testing.T) {
	sql := DetailQueries()[QueryRootDisk].SQL
	if !strings.Contains(sql, "FROM disk_space") {
		t.Fatalf("root_disk SQL = %q, want orbit disk_space table", sql)
	}
	if strings.Contains(sql, "FROM mounts") {
		t.Fatalf("root_disk SQL = %q, must not fall back to mounts", sql)
	}
}

func TestUsersQueryFiltersServiceAccounts(t *testing.T) {
	sql := DetailQueries()[QueryUsers].SQL
	for _, want := range []string{
		"type <> 'special'",
		"shell NOT LIKE '%/false'",
		"shell NOT LIKE '%/nologin'",
		"shell NOT LIKE '%/shutdown'",
		"shell NOT LIKE '%/halt'",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("users SQL missing %q: %s", want, sql)
		}
	}
}
