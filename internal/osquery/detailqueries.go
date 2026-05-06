package osquery

import (
	"context"
	"embed"
	"time"
)

const detailQueryCadence = time.Hour

const (
	queryOSVersion                   = "os_version"
	querySystemInfo                  = "system_info"
	queryOsqueryInfo                 = "osquery_info"
	queryOrbitInfo                   = "orbit_info"
	querySoftwareMacOS               = "software_macos"
	querySoftwareVSCodeExtensions    = "software_vscode_extensions"
	querySoftwareJetBrainsPlugins    = "software_jetbrains_plugins"
	querySoftwareGoBinaries          = "software_go_binaries"
	querySoftwarePythonPackages      = "software_python_packages"
	querySoftwareMacOSCodesign       = "software_macos_codesign"
	querySoftwareMacOSExecutableHash = "software_macos_executable_sha256"
)

//go:embed queries/*.sql
var queryFS embed.FS

// DetailQuery is one built-in query and its ingest function.
type DetailQuery struct {
	SQL       string
	Discovery string
	Optional  bool
	Ingest    func(context.Context, *Service, int64, []map[string]string) error
}

// DueDetailQueries is the osquery distributed work due for a host.
type DueDetailQueries struct {
	Queries   map[string]string
	Discovery map[string]string
}

// DetailQueries returns the built-in detail query registry.
func DetailQueries() map[string]DetailQuery {
	return map[string]DetailQuery{
		queryOSVersion: {
			SQL:    mustQuery("queries/os_version.sql"),
			Ingest: ingestOSVersion,
		},
		querySystemInfo: {
			SQL:    mustQuery("queries/system_info.sql"),
			Ingest: ingestSystemInfo,
		},
		queryOsqueryInfo: {
			SQL:    mustQuery("queries/osquery_info.sql"),
			Ingest: ingestOsqueryInfo,
		},
		queryOrbitInfo: {
			SQL:       mustQuery("queries/orbit_info.sql"),
			Discovery: discoveryTable("orbit_info"),
			Optional:  true,
			Ingest:    ingestOrbitInfo,
		},
		querySoftwareMacOS: {
			SQL:    mustQuery("queries/software_macos.sql"),
			Ingest: ingestSoftwareMacOS,
		},
		querySoftwareVSCodeExtensions: {
			SQL:       mustQuery("queries/software_vscode_extensions.sql"),
			Discovery: discoveryTable("vscode_extensions"),
			Optional:  true,
			Ingest:    ingestNoop,
		},
		querySoftwareJetBrainsPlugins: {
			SQL:       mustQuery("queries/software_jetbrains_plugins.sql"),
			Discovery: discoveryTable("jetbrains_plugins"),
			Optional:  true,
			Ingest:    ingestNoop,
		},
		querySoftwareGoBinaries: {
			SQL:       mustQuery("queries/software_go_binaries.sql"),
			Discovery: discoveryTable("go_binaries"),
			Optional:  true,
			Ingest:    ingestNoop,
		},
		querySoftwarePythonPackages: {
			SQL:       mustQuery("queries/software_python_packages.sql"),
			Discovery: discoveryTable("python_packages"),
			Optional:  true,
			Ingest:    ingestNoop,
		},
		querySoftwareMacOSCodesign: {
			SQL:       mustQuery("queries/software_macos_codesign.sql"),
			Discovery: discoveryTable("codesign"),
			Optional:  true,
			Ingest:    ingestNoop,
		},
		querySoftwareMacOSExecutableHash: {
			SQL:       mustQuery("queries/software_macos_executable_sha256.sql"),
			Discovery: discoveryTable("executable_hashes"),
			Optional:  true,
			Ingest:    ingestNoop,
		},
	}
}

func detailQueriesDue(lastUpdated *time.Time) DueDetailQueries {
	if lastUpdated != nil && time.Since(*lastUpdated) < detailQueryCadence {
		return DueDetailQueries{Queries: map[string]string{}, Discovery: map[string]string{}}
	}

	registry := DetailQueries()
	queries := make(map[string]string, len(registry))
	discovery := make(map[string]string)
	for name, query := range registry {
		queries[name] = query.SQL
		if query.Discovery != "" {
			discovery[name] = query.Discovery
		}
	}
	return DueDetailQueries{Queries: queries, Discovery: discovery}
}

func discoveryTable(name string) string {
	return "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = '" + name + "'"
}

func mustQuery(path string) string {
	content, err := queryFS.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(content)
}
