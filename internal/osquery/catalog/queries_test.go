package catalog

import (
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/scope"
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
	got := DetailQueriesDue(nil, "", scope.PlatformDarwin)
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
	got := DetailQueriesDue(nil, "", scope.PlatformDarwin)
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
		platform scope.Platform
		include  []string
		exclude  []string
	}{
		{
			platform: scope.PlatformWindows,
			include:  []string{QueryPrimaryInterfaceWindows, QuerySoftwareWindows, QueryCertificatesWindows},
			exclude: []string{
				QueryPrimaryInterfaceUnix,
				QueryRootDiskDarwin,
				QuerySoftwareMacOS,
				QueryCertificatesDarwin,
			},
		},
		{
			platform: scope.PlatformLinux,
			include:  []string{QueryPrimaryInterfaceUnix, QuerySoftwareLinux},
			exclude: []string{
				QueryPrimaryInterfaceWindows,
				QueryRootDiskDarwin,
				QuerySoftwareMacOS,
				QueryCertificatesDarwin,
				QueryCertificatesWindows,
			},
		},
		{
			platform: scope.PlatformUnknown,
			exclude: []string{
				QueryPrimaryInterfaceUnix,
				QueryPrimaryInterfaceWindows,
				QueryRootDiskDarwin,
				QuerySoftwareMacOS,
				QuerySoftwareLinux,
				QuerySoftwareWindows,
				QueryCertificatesDarwin,
				QueryCertificatesWindows,
			},
		},
	}
	for _, tc := range cases {
		t.Run(string(tc.platform), func(t *testing.T) {
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
	if got := DetailQueriesDue(&now, DetailQueryHash(), scope.PlatformDarwin); len(got.Queries) != 0 {
		t.Fatalf("fresh matching hash returned %d queries, want 0", len(got.Queries))
	}
	if got := DetailQueriesDue(&now, "old-hash", scope.PlatformDarwin); len(got.Queries) == 0 {
		t.Fatal("fresh stale hash returned no queries")
	}
}

func TestDetailQueriesUseExplicitColumns(t *testing.T) {
	for name, query := range DetailQueries() {
		upperSQL := strings.ToUpper(query.SQL)
		if strings.Contains(upperSQL, "SELECT *") || strings.Contains(query.SQL, ".*") {
			t.Fatalf("%s uses wildcard columns: %s", name, query.SQL)
		}
	}
}

func TestHostDetailQueriesProjectIngestShape(t *testing.T) {
	osVersionSQL := DetailQueries()[QueryOSVersion].SQL
	for _, want := range []string{"name", "version", "major", "minor", "build", "platform", "platform_like"} {
		if !strings.Contains(osVersionSQL, want) {
			t.Fatalf("os_version SQL missing %q: %s", want, osVersionSQL)
		}
	}
	if strings.Contains(osVersionSQL, "arch") {
		t.Fatalf("os_version SQL includes unused arch: %s", osVersionSQL)
	}

	orbitSQL := strings.TrimSpace(DetailQueries()[QueryOrbitInfo].SQL)
	if orbitSQL != "SELECT version FROM orbit_info;" {
		t.Fatalf("orbit_info SQL = %q, want version only", orbitSQL)
	}
}

func TestSoftwareQueriesProjectIngestShape(t *testing.T) {
	for _, name := range []string{
		QuerySoftwareMacOS,
		QuerySoftwareLinux,
		QuerySoftwareWindows,
		QuerySoftwareVSCodeExtensions,
		QuerySoftwareJetBrainsPlugins,
		QuerySoftwareGoBinaries,
		QuerySoftwarePythonPackages,
		QuerySoftwarePythonPackagesLegacy,
	} {
		sql := DetailQueries()[name].SQL
		for _, want := range []string{
			"name",
			"version",
			"bundle_identifier",
			"extension_id",
			"extension_for",
			"source",
			"vendor",
			"arch",
			"release",
			"last_opened_at",
			"installed_path",
		} {
			if !strings.Contains(sql, want) {
				t.Fatalf("%s SQL missing %q: %s", name, want, sql)
			}
		}
	}
}

func TestSoftwareEnrichmentQueriesProjectIngestShape(t *testing.T) {
	codesignSQL := DetailQueries()[QuerySoftwareMacOSCodesign].SQL
	for _, want := range []string{"path", "team_identifier", "cdhash_sha256"} {
		if !strings.Contains(codesignSQL, want) {
			t.Fatalf("codesign SQL missing %q: %s", want, codesignSQL)
		}
	}

	hashSQL := DetailQueries()[QuerySoftwareMacOSExecutableHash].SQL
	for _, want := range []string{"path", "executable_path", "executable_sha256"} {
		if !strings.Contains(hashSQL, want) {
			t.Fatalf("executable hash SQL missing %q: %s", want, hashSQL)
		}
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
