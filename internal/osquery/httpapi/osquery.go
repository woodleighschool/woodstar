package httpapi

import (
	"log/slog"

	"github.com/woodleighschool/woodstar/internal/activity"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/history"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/policies"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

// Dependencies are the runtime services exposed by the osquery admin API.
type Dependencies struct {
	Reports     *reports.Store
	Policies    *policies.Store
	LiveQueries *livequery.Store
	Hosts       *hosts.Store
	History     *history.Store
	Activity    activity.Recorder
	Logger      *slog.Logger
}

// RegisterAPI mounts osquery report, policy, and live-query endpoints.
func RegisterAPI(routes api.AppRoutes, deps Dependencies) {
	registerAPI(routes, deps)
}

// RegisterOpenAPI documents osquery endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerAPI(routes, Dependencies{})
}

func registerAPI(routes api.AppRoutes, deps Dependencies) {
	ordinary := routes.Ordinary
	registerOsqueryReports(ordinary, deps.Reports, deps.Activity, deps.Logger)
	registerHostOsqueryReports(ordinary, deps.Reports, deps.Hosts, deps.Logger)
	registerOsqueryPolicies(ordinary, routes.Sensitive, deps.Policies, deps.Activity, deps.Logger)
	registerHostOsqueryPolicies(ordinary, deps.Policies, deps.Hosts, deps.Logger)
	registerLiveQueries(
		routes.Sensitive,
		routes.StreamingSensitive,
		deps.LiveQueries,
		deps.Hosts,
		deps.Activity,
		deps.Logger,
	)
	registerHistory(ordinary, deps.History, deps.Policies, deps.Logger)
}
