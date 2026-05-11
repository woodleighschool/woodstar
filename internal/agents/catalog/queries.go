package catalog

import (
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
	QueryOSVersion                   = "os_version"
	QuerySystemInfo                  = "system_info"
	QueryOsqueryInfo                 = "osquery_info"
	QueryOsqueryFlags                = "osquery_flags"
	QueryOrbitInfo                   = "orbit_info"
	QueryUptime                      = "uptime"
	QueryRootDisk                    = "root_disk"
	QueryPrimaryInterface            = "primary_interface"
	QueryUsers                       = "users"
	QueryBatteries                   = "batteries"
	QuerySoftwareMacOS               = "software_macos"
	QuerySoftwareVSCodeExtensions    = "software_vscode_extensions"
	QuerySoftwareJetBrainsPlugins    = "software_jetbrains_plugins"
	QuerySoftwareGoBinaries          = "software_go_binaries"
	QuerySoftwarePythonPackages      = "software_python_packages"
	QuerySoftwareMacOSCodesign       = "software_macos_codesign"
	QuerySoftwareMacOSExecutableHash = "software_macos_executable_sha256"
)

//go:embed queries/*.sql
var queryFS embed.FS

var detailRegistry = sync.OnceValues(func() (map[string]DetailQuery, string) {
	registry := buildDetailQueries()
	return registry, hashDetailQueries(registry)
})

// DetailQuery is one built-in query descriptor.
type DetailQuery struct {
	SQL       string
	Discovery string
	Optional  bool
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
	registry := map[string]DetailQuery{
		QueryOSVersion: {
			SQL: mustQuery("queries/os_version.sql"),
		},
		QuerySystemInfo: {
			SQL: mustQuery("queries/system_info.sql"),
		},
		QueryOsqueryInfo: {
			SQL: mustQuery("queries/osquery_info.sql"),
		},
		QueryOsqueryFlags: {
			SQL:      mustQuery("queries/osquery_flags.sql"),
			Optional: true,
		},
		QueryOrbitInfo: {
			SQL:       mustQuery("queries/orbit_info.sql"),
			Discovery: tableExistsSQL("orbit_info"),
			Optional:  true,
		},
		QueryUptime: {
			SQL: mustQuery("queries/uptime.sql"),
		},
		QueryRootDisk: {
			SQL: mustQuery("queries/root_disk.sql"),
		},
		QueryPrimaryInterface: {
			SQL: mustQuery("queries/primary_interface.sql"),
		},
		QueryUsers: {
			SQL: mustQuery("queries/users.sql"),
		},
		QueryBatteries: {
			SQL:       mustQuery("queries/batteries.sql"),
			Discovery: tableExistsSQL("battery"),
			Optional:  true,
		},
	}
	maps.Copy(registry, softwareDetailQueries())
	return registry
}

func softwareDetailQueries() map[string]DetailQuery {
	return map[string]DetailQuery{
		QuerySoftwareMacOS: {
			SQL: mustQuery("queries/software_macos.sql"),
		},
		QuerySoftwareVSCodeExtensions: {
			SQL:       mustQuery("queries/software_vscode_extensions.sql"),
			Discovery: tableExistsSQL("vscode_extensions"),
			Optional:  true,
		},
		QuerySoftwareJetBrainsPlugins: {
			SQL:       mustQuery("queries/software_jetbrains_plugins.sql"),
			Discovery: tableExistsSQL("jetbrains_plugins"),
			Optional:  true,
		},
		QuerySoftwareGoBinaries: {
			SQL:       mustQuery("queries/software_go_binaries.sql"),
			Discovery: tableExistsSQL("go_binaries"),
			Optional:  true,
		},
		QuerySoftwarePythonPackages: {
			SQL:       mustQuery("queries/software_python_packages.sql"),
			Discovery: tableExistsSQL("python_packages"),
			Optional:  true,
		},
		QuerySoftwareMacOSCodesign: {
			SQL:       mustQuery("queries/software_macos_codesign.sql"),
			Discovery: tableExistsSQL("codesign"),
			Optional:  true,
		},
		QuerySoftwareMacOSExecutableHash: {
			SQL:       mustQuery("queries/software_macos_executable_sha256.sql"),
			Discovery: tableExistsSQL("executable_hashes"),
			Optional:  true,
		},
	}
}

func DetailQueriesDue(lastUpdated *time.Time, lastHash string) DueDetailQueries {
	registry, hash := detailRegistry()
	if lastUpdated != nil && lastHash == hash && time.Since(*lastUpdated) < detailQueryCadence {
		return DueDetailQueries{}
	}

	querySQL := make(map[string]string, len(registry))
	discovery := make(map[string]string)
	for name, query := range registry {
		querySQL[name] = query.SQL
		if query.Discovery != "" {
			discovery[name] = query.Discovery
		}
	}
	return DueDetailQueries{Queries: querySQL, Discovery: discovery}
}

func DetailQueryHash() string {
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

func tableExistsSQL(name string) string {
	return "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = '" + name + "'"
}

func mustQuery(path string) string {
	content, err := queryFS.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(content)
}
