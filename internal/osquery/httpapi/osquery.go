package httpapi

import (
	"log/slog"

	authhuma "github.com/woodleighschool/goodies/auth/huma"

	"github.com/woodleighschool/woodstar/internal/activity"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/history"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/policies"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/rbac"
)

// Dependencies are the runtime services exposed by the osquery admin API.
type Dependencies struct {
	Reports     *reports.Store
	Policies    *policies.Store
	LiveQueries *livequery.Store
	Hosts       *hosts.Store
	History     *history.Store
	Activity    activity.Recorder
	Authorizer  authhuma.Authorizer
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
	reportsAPI := authhuma.ResourceAPI(routes.Protected, deps.Authorizer, deps.Logger, rbac.ResourceOsqueryReports)
	policiesAPI := authhuma.ResourceAPI(routes.Protected, deps.Authorizer, deps.Logger, rbac.ResourceOsqueryPolicies)
	remediationsAPI := authhuma.ResourceAPI(routes.Protected, deps.Authorizer, deps.Logger, rbac.ResourceOsqueryRemediations)
	liveQueriesAPI := authhuma.ResourceAPI(routes.Protected, deps.Authorizer, deps.Logger, rbac.ResourceOsqueryLiveQueries)
	streamingLiveQueriesAPI := authhuma.ResourceAPI(routes.Streaming, deps.Authorizer, deps.Logger,
		rbac.ResourceOsqueryLiveQueries,
	)
	overviewAPI := authhuma.ResourceAPI(routes.Protected, deps.Authorizer, deps.Logger, rbac.ResourceOsqueryOverview)

	registerOsqueryReports(reportsAPI, deps.Reports, deps.Activity, deps.Logger)
	registerHostOsqueryReports(reportsAPI, deps.Reports, deps.Hosts, deps.Logger)
	registerOsqueryPolicies(
		policiesAPI,
		remediationsAPI,
		deps.Policies,
		deps.Activity,
		deps.Logger,
	)
	registerHostOsqueryPolicies(policiesAPI, deps.Policies, deps.Hosts, deps.Logger)
	registerLiveQueries(
		liveQueriesAPI,
		streamingLiveQueriesAPI,
		deps.LiveQueries,
		deps.Hosts,
		deps.Activity,
		deps.Logger,
	)
	registerHistory(overviewAPI, deps.History, deps.Policies, deps.Logger)
}
