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
		"users",
		"batteries",
		"certificates_darwin",
		"munki_info",
		"munki_installs",
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
	got := DetailQueriesDue(nil, "")
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
			t.Fatalf("missing detail query %q", name)
		}
	}
	if got.Discovery["orbit_info"] == "" ||
		got.Discovery["certificates_darwin"] == "" ||
		got.Discovery["software_vscode_extensions"] == "" ||
		got.Discovery["software_jetbrains_plugins"] == "" ||
		got.Discovery["software_go_binaries"] == "" ||
		got.Discovery["software_python_packages"] == "" ||
		got.Discovery["software_macos_codesign"] == "" ||
		got.Discovery["software_macos_executable_sha256"] == "" ||
		got.Discovery["munki_info"] == "" ||
		got.Discovery["munki_installs"] == "" {
		t.Fatalf("missing optional detail query discovery: %#v", got.Discovery)
	}
}

func TestDetailQueriesDueDiscoversOsqueryVirtualTables(t *testing.T) {
	got := DetailQueriesDue(nil, "")
	for _, name := range []string{
		QueryBatteries,
		QueryCertificatesDarwin,
		QueryRootDiskDarwin,
		QueryMunkiInfo,
		QueryMunkiInstalls,
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

func TestDetailQueriesDueWhenHashChanges(t *testing.T) {
	now := time.Now()
	if got := DetailQueriesDue(&now, DetailQueryHash()); len(got.Queries) != 0 {
		t.Fatalf("fresh matching hash returned %d queries, want 0", len(got.Queries))
	}
	if got := DetailQueriesDue(&now, "old-hash"); len(got.Queries) == 0 {
		t.Fatal("fresh stale hash returned no queries")
	}
}

func TestDetailQueryHashIncludesEveryQueryContract(t *testing.T) {
	base := DetailQuery{
		SQL:       "SELECT value FROM example",
		Discovery: "SELECT 1 FROM osquery_registry",
		Optional:  true,
		Ingest:    IngestMunkiInstalls,
	}
	baseHash := hashDetailQueries(map[string]DetailQuery{"example": base})

	tests := map[string]DetailQuery{
		"SQL": {
			SQL:       "SELECT other FROM example",
			Discovery: base.Discovery,
			Optional:  base.Optional,
			Ingest:    base.Ingest,
		},
		"discovery": {
			SQL:       base.SQL,
			Discovery: "SELECT 1",
			Optional:  base.Optional,
			Ingest:    base.Ingest,
		},
		"optional": {
			SQL:       base.SQL,
			Discovery: base.Discovery,
			Optional:  false,
			Ingest:    base.Ingest,
		},
		"ingest": {
			SQL:       base.SQL,
			Discovery: base.Discovery,
			Optional:  base.Optional,
			Ingest:    IngestMunkiInfo,
		},
	}
	for name, changed := range tests {
		t.Run(name, func(t *testing.T) {
			if got := hashDetailQueries(map[string]DetailQuery{"example": changed}); got == baseHash {
				t.Fatalf("hash unchanged after %s contract changed", name)
			}
		})
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
	for _, want := range []string{"name", "version", "major", "minor", "build", "platform"} {
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
		QuerySoftwareVSCodeExtensions,
		QuerySoftwareJetBrainsPlugins,
		QuerySoftwareGoBinaries,
		QuerySoftwarePythonPackages,
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

func TestMunkiQueriesProjectStatusShape(t *testing.T) {
	infoSQL := DetailQueries()[QueryMunkiInfo].SQL
	for _, want := range []string{
		"version",
		"errors",
		"warnings",
		"problem_installs",
		"success",
		"start_time",
		"end_time",
		"manifest_name",
	} {
		if !strings.Contains(infoSQL, want) {
			t.Fatalf("munki_info SQL missing %q: %s", want, infoSQL)
		}
	}

	installsSQL := DetailQueries()[QueryMunkiInstalls].SQL
	for _, want := range []string{"name", "installed", "installed_version"} {
		if !strings.Contains(installsSQL, want) {
			t.Fatalf("munki_installs SQL missing %q: %s", want, installsSQL)
		}
	}
	if strings.Contains(installsSQL, "end_time") {
		t.Fatalf("munki_installs SQL includes report-level end_time: %s", installsSQL)
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
