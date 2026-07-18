package handlers

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

// RegisterOsquery mounts osquery report, check, and live-query endpoints.
func RegisterOsquery(
	ordinary huma.API,
	sensitive huma.API,
	streamingSensitive huma.API,
	reportStore *reports.Store,
	checkStore *checks.Store,
	liveQueries *livequery.Manager,
	hostStore *hosts.Store,
	logger *slog.Logger,
) {
	registerOsqueryReports(ordinary, reportStore, logger)
	registerHostOsqueryReports(ordinary, reportStore, hostStore, logger)
	registerOsqueryChecks(ordinary, checkStore, logger)
	registerHostOsqueryChecks(ordinary, checkStore, hostStore, logger)
	registerLiveQueries(sensitive, streamingSensitive, liveQueries, hostStore, logger)
}
