package osquery

import (
	"context"
	"embed"
	"time"
)

const detailQueryCadence = time.Hour

const (
	queryOSVersion     = "os_version"
	querySystemInfo    = "system_info"
	queryOsqueryInfo   = "osquery_info"
	querySoftwareMacOS = "software_macos"
)

//go:embed queries/*.sql
var queryFS embed.FS

// DetailQuery is one built-in query and its ingest function.
type DetailQuery struct {
	SQL    string
	Ingest func(context.Context, *Service, int64, []map[string]string) error
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
		querySoftwareMacOS: {
			SQL:    mustQuery("queries/software_macos.sql"),
			Ingest: ingestSoftwareMacOS,
		},
	}
}

func detailQueriesDue(lastUpdated *time.Time) map[string]string {
	if lastUpdated != nil && time.Since(*lastUpdated) < detailQueryCadence {
		return map[string]string{}
	}

	registry := DetailQueries()
	queries := make(map[string]string, len(registry))
	for name, query := range registry {
		queries[name] = query.SQL
	}
	return queries
}

func mustQuery(path string) string {
	content, err := queryFS.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(content)
}
