package osquery

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"maps"
	"sort"
	"strings"
	"sync"
	"time"
)

const detailQueryCadence = time.Hour

const (
	queryOSVersion                   = "os_version"
	querySystemInfo                  = "system_info"
	queryOsqueryInfo                 = "osquery_info"
	queryOsqueryFlags                = "osquery_flags"
	queryOrbitInfo                   = "orbit_info"
	queryUptime                      = "uptime"
	queryRootDisk                    = "root_disk"
	queryPrimaryInterface            = "primary_interface"
	queryUsers                       = "users"
	queryBatteries                   = "batteries"
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

var detailRegistry = sync.OnceValues(func() (map[string]DetailQuery, string) {
	registry := buildDetailQueries()
	return registry, hashDetailQueries(registry)
})

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
	registry, _ := detailRegistry()
	return maps.Clone(registry)
}

func buildDetailQueries() map[string]DetailQuery {
	queries := map[string]DetailQuery{
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
		queryOsqueryFlags: {
			SQL:      mustQuery("queries/osquery_flags.sql"),
			Optional: true,
			Ingest:   ingestOsqueryFlags,
		},
		queryOrbitInfo: {
			SQL:       mustQuery("queries/orbit_info.sql"),
			Discovery: discoveryTable("orbit_info"),
			Optional:  true,
			Ingest:    ingestOrbitInfo,
		},
		queryUptime: {
			SQL:    mustQuery("queries/uptime.sql"),
			Ingest: ingestUptime,
		},
		queryRootDisk: {
			SQL:    mustQuery("queries/root_disk.sql"),
			Ingest: ingestRootDisk,
		},
		queryPrimaryInterface: {
			SQL:    mustQuery("queries/primary_interface.sql"),
			Ingest: ingestPrimaryInterface,
		},
		queryUsers: {
			SQL:    mustQuery("queries/users.sql"),
			Ingest: ingestUsers,
		},
		queryBatteries: {
			SQL:       mustQuery("queries/batteries.sql"),
			Discovery: discoveryTable("battery"),
			Optional:  true,
			Ingest:    ingestBatteries,
		},
	}
	maps.Copy(queries, softwareDetailQueries())
	return queries
}

func softwareDetailQueries() map[string]DetailQuery {
	return map[string]DetailQuery{
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

func detailQueriesDue(lastUpdated *time.Time, lastHash string) DueDetailQueries {
	registry, hash := detailRegistry()
	if lastUpdated != nil && lastHash == hash && time.Since(*lastUpdated) < detailQueryCadence {
		return DueDetailQueries{Queries: map[string]string{}, Discovery: map[string]string{}}
	}

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

func detailQueryHash() string {
	_, hash := detailRegistry()
	return hash
}

func hashDetailQueries(registry map[string]DetailQuery) string {
	names := make([]string, 0, len(registry))
	for name, query := range registry {
		if !query.Optional {
			names = append(names, name+"\x00"+query.SQL)
		}
	}
	sort.Strings(names)
	sum := sha256.Sum256([]byte(strings.Join(names, "\x00")))
	return hex.EncodeToString(sum[:])
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
