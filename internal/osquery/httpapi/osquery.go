package httpapi

import (
	"log/slog"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/policies"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

// RegisterAPI mounts osquery report, policy, and live-query endpoints.
func RegisterAPI(
	routes api.AppRoutes,
	reportStore *reports.Store,
	policyStore *policies.Store,
	liveQueries *livequery.Store,
	hostStore *hosts.Store,
	logger *slog.Logger,
) {
	registerAPI(routes, reportStore, policyStore, liveQueries, hostStore, logger)
}

// RegisterOpenAPI documents osquery endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerAPI(routes, nil, nil, nil, nil, nil)
}

func registerAPI(
	routes api.AppRoutes,
	reportStore *reports.Store,
	policyStore *policies.Store,
	liveQueries *livequery.Store,
	hostStore *hosts.Store,
	logger *slog.Logger,
) {
	ordinary := routes.Ordinary
	registerOsqueryReports(ordinary, reportStore, logger)
	registerHostOsqueryReports(ordinary, reportStore, hostStore, logger)
	registerOsqueryPolicies(ordinary, routes.Sensitive, policyStore, logger)
	registerHostOsqueryPolicies(ordinary, policyStore, hostStore, logger)
	registerLiveQueries(routes.Sensitive, routes.StreamingSensitive, liveQueries, hostStore, logger)
}
