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
		"root_disk_darwin",
		"primary_interface_unix",
		"primary_interface_windows",
		"users",
		"batteries",
		"certificates_darwin",
		"certificates_windows",
		"software_macos",
		"software_linux",
		"software_windows",
		"software_vscode_extensions",
		"software_jetbrains_plugins",
		"software_go_binaries",
		"software_python_packages",
		"software_python_packages_legacy",
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
	got := DetailQueriesDue(nil, "", "darwin")
	for _, name := range []string{
		QueryOSVersion,
		QuerySystemInfo,
		QueryOsqueryInfo,
		QueryUptime,
		QueryRootDiskDarwin,
		QueryPrimaryInterfaceUnix,
		QueryUsers,
		QuerySoftwareMacOS,
	} {
		if got.Queries[name] == "" {
			t.Fatalf("missing darwin detail query %q", name)
		}
	}
	for _, name := range []string{
		QueryPrimaryInterfaceWindows,
		QuerySoftwareLinux,
		QuerySoftwareWindows,
	} {
		if got.Queries[name] != "" {
			t.Fatalf("unexpected darwin detail query %q", name)
		}
	}
	if got.Discovery["orbit_info"] == "" ||
		got.Discovery["certificates_darwin"] == "" ||
		got.Discovery["software_vscode_extensions"] == "" ||
		got.Discovery["software_jetbrains_plugins"] == "" ||
		got.Discovery["software_go_binaries"] == "" ||
		got.Discovery["software_python_packages"] == "" ||
		got.Discovery["software_python_packages_legacy"] == "" ||
		got.Discovery["software_macos_codesign"] == "" ||
		got.Discovery["software_macos_executable_sha256"] == "" {
		t.Fatalf("missing optional detail query discovery: %#v", got.Discovery)
	}
}

func TestDetailQueriesDueDiscoversOsqueryVirtualTables(t *testing.T) {
	got := DetailQueriesDue(nil, "", "darwin")
	for _, name := range []string{
		QueryBatteries,
		QueryCertificatesDarwin,
		QueryRootDiskDarwin,
		QuerySoftwareMacOSCodesign,
	} {
		discovery := got.Discovery[name]
		if !strings.Contains(discovery, "FROM osquery_registry") {
			t.Fatalf("%s discovery = %q, want osquery_registry", name, discovery)
		}
		if strings.Contains(discovery, "sqlite_master") {
			t.Fatalf("%s discovery = %q, must not use sqlite_master for osquery virtual tables", name, discovery)
		}
	}
}

func TestDetailQueriesDueFiltersByPlatform(t *testing.T) {
	cases := []struct {
		platform string
		include  []string
		exclude  []string
	}{
		{
			platform: "windows",
			include:  []string{QueryPrimaryInterfaceWindows, QuerySoftwareWindows, QueryCertificatesWindows},
			exclude: []string{
				QueryPrimaryInterfaceUnix,
				QueryRootDiskDarwin,
				QuerySoftwareMacOS,
				QueryCertificatesDarwin,
			},
		},
		{
			platform: "ubuntu",
			include:  []string{QueryPrimaryInterfaceUnix, QuerySoftwareLinux},
			exclude: []string{
				QueryPrimaryInterfaceWindows,
				QueryRootDiskDarwin,
				QuerySoftwareMacOS,
				QueryCertificatesDarwin,
				QueryCertificatesWindows,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.platform, func(t *testing.T) {
			got := DetailQueriesDue(nil, "", tc.platform)
			for _, name := range tc.include {
				if got.Queries[name] == "" {
					t.Fatalf("%s missing query %q", tc.platform, name)
				}
			}
			for _, name := range tc.exclude {
				if got.Queries[name] != "" {
					t.Fatalf("%s unexpectedly included query %q", tc.platform, name)
				}
			}
		})
	}
}

func TestDetailQueriesDueWhenHashChanges(t *testing.T) {
	now := time.Now()
	if got := DetailQueriesDue(&now, DetailQueryHash(), "darwin"); len(got.Queries) != 0 {
		t.Fatalf("fresh matching hash returned %d queries, want 0", len(got.Queries))
	}
	if got := DetailQueriesDue(&now, "old-hash", "darwin"); len(got.Queries) == 0 {
		t.Fatal("fresh stale hash returned no queries")
	}
}

func TestRootDiskQueryUsesOrbitDiskSpace(t *testing.T) {
	sql := DetailQueries()[QueryRootDiskDarwin].SQL
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
