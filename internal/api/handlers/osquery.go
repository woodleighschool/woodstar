//nolint:dupl // Huma route registration is intentionally explicit per resource.
package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
	"github.com/woodleighschool/woodstar/internal/platforms"
	"github.com/woodleighschool/woodstar/internal/scope"
)

const (
	reportsTag     = "Reports"
	reportResource = "report"
	reportIDPath   = "/api/osquery/reports/{id}"

	checksTag     = "Checks"
	checkResource = "check"
	checkIDPath   = "/api/osquery/checks/{id}"

	liveQueriesTag = "Live Queries"
)

// Osquery reports.

type reportMutationBody struct {
	Name              string               `json:"name"`
	Description       string               `json:"description,omitempty"`
	Query             string               `json:"query"`
	Platforms         []platforms.Platform `json:"platforms"                     minItems:"1" nullable:"false"`
	MinOsqueryVersion *string              `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int                  `json:"schedule_interval,omitempty"`
	LabelScope        scope.LabelScope     `json:"label_scope"`
}

type reportBody struct {
	ID                int64                `json:"id"`
	Name              string               `json:"name"`
	Description       string               `json:"description"`
	Query             string               `json:"query"`
	Platforms         []platforms.Platform `json:"platforms"                     minItems:"1" nullable:"false"`
	MinOsqueryVersion *string              `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int                  `json:"schedule_interval"`
	LabelScope        scope.LabelScope     `json:"label_scope,omitzero"`
	CreatedByUserID   *int64               `json:"created_by_user_id,omitempty"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
}

type reportResultBody struct {
	ReportID    int64             `json:"report_id"`
	ReportName  string            `json:"report_name"`
	HostID      int64             `json:"host_id"`
	HostName    string            `json:"host_name"`
	Columns     map[string]string `json:"columns"`
	LastFetched time.Time         `json:"last_fetched,omitzero"`
}

type reportListInput struct {
	ListQueryInput
	Platform string `query:"platform,omitempty"`
}

type reportGetInput struct {
	ID string `path:"id"`
}

type reportCreateInput struct {
	Body reportMutationBody
}

type reportPutInput struct {
	ID   string `path:"id"`
	Body reportMutationBody
}

type reportDeleteInput struct {
	ID string `path:"id"`
}

type reportBulkDeleteInput struct {
	Body bulkIDsBody
}

type reportListOutput struct {
	Body paginatedBody[reportBody]
}

type reportOutput struct {
	Body reportBody
}

type reportResultsOutput struct {
	Body reportResultsBody
}

type hostReportsOutput struct {
	Body hostReportsBody
}

type hostReportResultsOutput struct {
	Body hostReportResultsBody
}

type reportResultsBody struct {
	Items []reportResultBody `json:"items"`
}

type hostReportsBody struct {
	Items []reports.HostReport `json:"items"`
}

type hostReportResultsBody struct {
	ReportID    int64              `json:"report_id"`
	HostID      int64              `json:"host_id"`
	HostName    string             `json:"host_name"`
	LastFetched *time.Time         `json:"last_fetched,omitempty"`
	Items       []reportResultBody `json:"items"`
}

type hostReportResultsInput struct {
	ID       string `path:"id"`
	ReportID string `path:"report_id"`
}

func RegisterReports(api huma.API, reportStore *reports.Store, hostStore *hosts.Store) {
	registerListReports(api, reportStore)
	registerCreateReport(api, reportStore)
	registerGetReport(api, reportStore)
	registerUpdateReport(api, reportStore)
	registerDeleteReport(api, reportStore)
	registerBulkDeleteReports(api, reportStore)
	registerReportResults(api, reportStore)
	registerHostReports(api, reportStore, hostStore)
	registerHostReportResults(api, reportStore, hostStore)
}

func registerListReports(api huma.API, reportStore *reports.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-osquery-reports",
		Method:      http.MethodGet,
		Path:        "/api/osquery/reports",
		Tags:        []string{reportsTag},
		Summary:     "List reports",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *reportListInput) (*reportListOutput, error) {
		items, count, err := reportStore.List(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(reportResource, err)
		}
		return &reportListOutput{Body: paginatedBody[reportBody]{
			Items: reportBodies(items),
			Count: count,
		}}, nil
	})
}

func registerCreateReport(api huma.API, reportStore *reports.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-osquery-report",
		Method:        http.MethodPost,
		Path:          "/api/osquery/reports",
		Tags:          []string{reportsTag},
		Summary:       "Create a report",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusConflict},
	}, func(ctx context.Context, input *reportCreateInput) (*reportOutput, error) {
		params, err := input.Body.createParams(currentUserID(ctx))
		if err != nil {
			return nil, err
		}
		report, err := reportStore.Create(ctx, params)
		if err != nil {
			return nil, resourceMutationError(reportResource, err)
		}
		return &reportOutput{Body: reportBodyFromReport(*report)}, nil
	})
}

func registerGetReport(api huma.API, reportStore *reports.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-osquery-report",
		Method:      http.MethodGet,
		Path:        reportIDPath,
		Tags:        []string{reportsTag},
		Summary:     "Get a report",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *reportGetInput) (*reportOutput, error) {
		id, err := parseResourceID(input.ID, reportResource)
		if err != nil {
			return nil, err
		}
		report, err := reportStore.GetByID(ctx, id)
		if err != nil {
			return nil, resourceMutationError(reportResource, err)
		}
		return &reportOutput{Body: reportBodyFromReport(*report)}, nil
	})
}

func registerUpdateReport(api huma.API, reportStore *reports.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-osquery-report",
		Method:      http.MethodPut,
		Path:        reportIDPath,
		Tags:        []string{reportsTag},
		Summary:     "Replace a report",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *reportPutInput) (*reportOutput, error) {
		id, err := parseResourceID(input.ID, reportResource)
		if err != nil {
			return nil, err
		}
		params, err := input.Body.updateParams()
		if err != nil {
			return nil, err
		}
		report, err := reportStore.Update(ctx, id, params)
		if err != nil {
			return nil, resourceMutationError(reportResource, err)
		}
		return &reportOutput{Body: reportBodyFromReport(*report)}, nil
	})
}

func registerDeleteReport(api huma.API, reportStore *reports.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-osquery-report",
		Method:      http.MethodDelete,
		Path:        reportIDPath,
		Tags:        []string{reportsTag},
		Summary:     "Delete a report",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *reportDeleteInput) (*struct{}, error) {
		id, err := parseResourceID(input.ID, reportResource)
		if err != nil {
			return nil, err
		}
		if err := reportStore.Delete(ctx, id); err != nil {
			return nil, resourceMutationError(reportResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteReports(api huma.API, reportStore *reports.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-osquery-reports",
		Method:      http.MethodPost,
		Path:        "/api/osquery/reports/bulk-delete",
		Tags:        []string{reportsTag},
		Summary:     "Delete reports",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *reportBulkDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		ids, err := input.Body.ids("report IDs")
		if err != nil {
			return nil, err
		}
		if _, err := reportStore.DeleteMany(ctx, ids); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func registerReportResults(api huma.API, reportStore *reports.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-osquery-report-results",
		Method:      http.MethodGet,
		Path:        "/api/osquery/reports/{id}/results",
		Tags:        []string{reportsTag},
		Summary:     "List latest snapshots for a report",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *reportGetInput) (*reportResultsOutput, error) {
		id, err := parseResourceID(input.ID, reportResource)
		if err != nil {
			return nil, err
		}
		rows, err := reportStore.Results(ctx, id)
		if err != nil {
			return nil, err
		}
		return &reportResultsOutput{Body: reportResultsBody{Items: reportResultBodies(rows)}}, nil
	})
}

func registerHostReports(api huma.API, reportStore *reports.Store, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-osquery-reports",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/osquery/reports",
		Tags:        []string{reportsTag, hostsTag},
		Summary:     "List reports for a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostReportsOutput, error) {
		id, err := parseResourceID(input.ID, hostResource)
		if err != nil {
			return nil, err
		}
		host, err := hostStore.GetByID(ctx, id)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		}
		if err != nil {
			return nil, err
		}
		rows, err := reportStore.HostReports(ctx, host)
		if err != nil {
			return nil, err
		}
		return &hostReportsOutput{Body: hostReportsBody{Items: rows}}, nil
	})
}

func registerHostReportResults(api huma.API, reportStore *reports.Store, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-osquery-report-results",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/osquery/reports/{report_id}",
		Tags:        []string{reportsTag, hostsTag},
		Summary:     "List report rows for one host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostReportResultsInput) (*hostReportResultsOutput, error) {
		hostID, err := parseResourceID(input.ID, hostResource)
		if err != nil {
			return nil, err
		}
		reportID, err := parseResourceID(input.ReportID, reportResource)
		if err != nil {
			return nil, err
		}
		host, err := hostStore.GetByID(ctx, hostID)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		}
		if err != nil {
			return nil, err
		}
		rows, lastFetched, err := reportStore.HostResults(ctx, hostID, reportID)
		if err != nil {
			return nil, err
		}
		return &hostReportResultsOutput{Body: hostReportResultsBody{
			ReportID:    reportID,
			HostID:      hostID,
			HostName:    host.DisplayName,
			LastFetched: lastFetched,
			Items:       reportResultBodies(rows),
		}}, nil
	})
}

func (input reportListInput) params() reports.ReportListParams {
	return reports.ReportListParams{
		ListParams: input.ListQueryInput.params(),
		Platform:   input.Platform,
	}
}

func (body reportMutationBody) createParams(userID *int64) (reports.ReportCreate, error) {
	s, err := normalizeLabelScope(body.LabelScope)
	if err != nil {
		return reports.ReportCreate{}, err
	}
	return reports.ReportCreate{
		Name:              body.Name,
		Description:       body.Description,
		Query:             body.Query,
		Platforms:         body.Platforms,
		MinOsqueryVersion: body.MinOsqueryVersion,
		ScheduleInterval:  body.ScheduleInterval,
		LabelScope:        s,
		CreatedByUserID:   userID,
	}, nil
}

func (body reportMutationBody) updateParams() (reports.ReportUpdate, error) {
	s, err := normalizeLabelScope(body.LabelScope)
	if err != nil {
		return reports.ReportUpdate{}, err
	}
	return reports.ReportUpdate{
		Name:              body.Name,
		Description:       body.Description,
		Query:             body.Query,
		Platforms:         body.Platforms,
		MinOsqueryVersion: body.MinOsqueryVersion,
		ScheduleInterval:  body.ScheduleInterval,
		LabelScope:        s,
	}, nil
}

func reportBodies(items []reports.Report) []reportBody {
	out := make([]reportBody, 0, len(items))
	for _, item := range items {
		out = append(out, reportBodyFromReport(item))
	}
	return out
}

func reportBodyFromReport(report reports.Report) reportBody {
	return reportBody{
		ID:                report.ID,
		Name:              report.Name,
		Description:       report.Description,
		Query:             report.Query,
		Platforms:         report.Platforms,
		MinOsqueryVersion: report.MinOsqueryVersion,
		ScheduleInterval:  report.ScheduleInterval,
		LabelScope:        report.LabelScope,
		CreatedByUserID:   report.CreatedByUserID,
		CreatedAt:         report.CreatedAt,
		UpdatedAt:         report.UpdatedAt,
	}
}

func reportResultBodies(items []reports.ReportResult) []reportResultBody {
	out := make([]reportResultBody, 0, len(items))
	for _, item := range items {
		out = append(out, reportResultBody{
			ReportID:    item.ReportID,
			ReportName:  item.ReportName,
			HostID:      item.HostID,
			HostName:    item.HostName,
			Columns:     item.Columns,
			LastFetched: item.LastFetched,
		})
	}
	return out
}

// Osquery checks.

type checkMutationBody struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Query       string               `json:"query"`
	Platforms   []platforms.Platform `json:"platforms"             minItems:"1" nullable:"false"`
	LabelScope  scope.LabelScope     `json:"label_scope"`
}

type checkListInput struct {
	ListQueryInput
	Platform string `query:"platform,omitempty"`
}

type checkGetInput struct {
	ID string `path:"id"`
}

type checkCreateInput struct {
	Body checkMutationBody
}

type checkPutInput struct {
	ID   string `path:"id"`
	Body checkMutationBody
}

type checkDeleteInput struct {
	ID string `path:"id"`
}

type checkBulkDeleteInput struct {
	Body bulkIDsBody
}

type checkListOutput struct {
	Body paginatedBody[checks.Check]
}

type checkOutput struct {
	Body checks.Check
}

type checkHostsOutput struct {
	Body struct {
		Items []checks.CheckHostStatus `json:"items"`
	}
}

func RegisterChecks(api huma.API, checkStore *checks.Store, hostStore *hosts.Store) {
	registerListChecks(api, checkStore)
	registerCreateCheck(api, checkStore)
	registerGetCheck(api, checkStore)
	registerUpdateCheck(api, checkStore)
	registerDeleteCheck(api, checkStore)
	registerBulkDeleteChecks(api, checkStore)
	registerCheckHosts(api, checkStore)
	registerHostChecks(api, checkStore, hostStore)
}

func registerListChecks(api huma.API, checkStore *checks.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-osquery-checks",
		Method:      http.MethodGet,
		Path:        "/api/osquery/checks",
		Tags:        []string{checksTag},
		Summary:     "List checks",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *checkListInput) (*checkListOutput, error) {
		items, count, err := checkStore.List(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(checkResource, err)
		}
		return &checkListOutput{Body: paginatedBody[checks.Check]{Items: items, Count: count}}, nil
	})
}

func registerCreateCheck(api huma.API, checkStore *checks.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-osquery-check",
		Method:        http.MethodPost,
		Path:          "/api/osquery/checks",
		Tags:          []string{checksTag},
		Summary:       "Create a check",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusConflict},
	}, func(ctx context.Context, input *checkCreateInput) (*checkOutput, error) {
		params, err := input.Body.createParams(currentUserID(ctx))
		if err != nil {
			return nil, err
		}
		check, err := checkStore.Create(ctx, params)
		if err != nil {
			return nil, resourceMutationError(checkResource, err)
		}
		return &checkOutput{Body: *check}, nil
	})
}

func registerGetCheck(api huma.API, checkStore *checks.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-osquery-check",
		Method:      http.MethodGet,
		Path:        checkIDPath,
		Tags:        []string{checksTag},
		Summary:     "Get a check",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *checkGetInput) (*checkOutput, error) {
		id, err := parseResourceID(input.ID, checkResource)
		if err != nil {
			return nil, err
		}
		check, err := checkStore.GetByID(ctx, id)
		if err != nil {
			return nil, resourceMutationError(checkResource, err)
		}
		return &checkOutput{Body: *check}, nil
	})
}

func registerUpdateCheck(api huma.API, checkStore *checks.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-osquery-check",
		Method:      http.MethodPut,
		Path:        checkIDPath,
		Tags:        []string{checksTag},
		Summary:     "Replace a check",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *checkPutInput) (*checkOutput, error) {
		id, err := parseResourceID(input.ID, checkResource)
		if err != nil {
			return nil, err
		}
		params, err := input.Body.updateParams()
		if err != nil {
			return nil, err
		}
		check, err := checkStore.Update(ctx, id, params)
		if err != nil {
			return nil, resourceMutationError(checkResource, err)
		}
		return &checkOutput{Body: *check}, nil
	})
}

func registerDeleteCheck(api huma.API, checkStore *checks.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-osquery-check",
		Method:      http.MethodDelete,
		Path:        checkIDPath,
		Tags:        []string{checksTag},
		Summary:     "Delete a check",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *checkDeleteInput) (*struct{}, error) {
		id, err := parseResourceID(input.ID, checkResource)
		if err != nil {
			return nil, err
		}
		if err := checkStore.Delete(ctx, id); err != nil {
			return nil, resourceMutationError(checkResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteChecks(api huma.API, checkStore *checks.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-osquery-checks",
		Method:      http.MethodPost,
		Path:        "/api/osquery/checks/bulk-delete",
		Tags:        []string{checksTag},
		Summary:     "Delete checks",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *checkBulkDeleteInput) (*struct{}, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		ids, err := input.Body.ids("check IDs")
		if err != nil {
			return nil, err
		}
		if _, err := checkStore.DeleteMany(ctx, ids); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func registerCheckHosts(api huma.API, checkStore *checks.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-osquery-check-hosts",
		Method:      http.MethodGet,
		Path:        "/api/osquery/checks/{id}/hosts",
		Tags:        []string{checksTag},
		Summary:     "List check host status",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *checkGetInput) (*checkHostsOutput, error) {
		id, err := parseResourceID(input.ID, checkResource)
		if err != nil {
			return nil, err
		}
		rows, err := checkStore.HostStatuses(ctx, id)
		if err != nil {
			return nil, err
		}
		out := &checkHostsOutput{}
		out.Body.Items = rows
		return out, nil
	})
}

func registerHostChecks(api huma.API, checkStore *checks.Store, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-osquery-checks",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/osquery/checks",
		Tags:        []string{checksTag, hostsTag},
		Summary:     "List checks for a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*checkHostsOutput, error) {
		id, err := parseResourceID(input.ID, hostResource)
		if err != nil {
			return nil, err
		}
		host, err := hostStore.GetByID(ctx, id)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		}
		if err != nil {
			return nil, err
		}
		rows, err := checkStore.HostChecks(ctx, host)
		if err != nil {
			return nil, err
		}
		out := &checkHostsOutput{}
		out.Body.Items = rows
		return out, nil
	})
}

func (input checkListInput) params() checks.CheckListParams {
	return checks.CheckListParams{
		ListParams: input.ListQueryInput.params(),
		Platform:   input.Platform,
	}
}

func (body checkMutationBody) createParams(userID *int64) (checks.CheckCreate, error) {
	s, err := normalizeLabelScope(body.LabelScope)
	if err != nil {
		return checks.CheckCreate{}, err
	}
	return checks.CheckCreate{
		Name:            body.Name,
		Description:     body.Description,
		Query:           body.Query,
		Platforms:       body.Platforms,
		LabelScope:      s,
		CreatedByUserID: userID,
	}, nil
}

func (body checkMutationBody) updateParams() (checks.CheckUpdate, error) {
	s, err := normalizeLabelScope(body.LabelScope)
	if err != nil {
		return checks.CheckUpdate{}, err
	}
	return checks.CheckUpdate{
		Name:        body.Name,
		Description: body.Description,
		Query:       body.Query,
		Platforms:   body.Platforms,
		LabelScope:  s,
	}, nil
}

// Osquery live queries (one-shot, no DB row, results stream then disappear).

type liveQueryCreateBody struct {
	ReportID *int64                `json:"report_id,omitempty"`
	SQL      string                `json:"sql"`
	Selected liveQuerySelectedBody `json:"selected,omitzero"`
}

type liveQuerySelectedBody struct {
	Hosts  []int64 `json:"hosts,omitempty"`
	Labels []int64 `json:"labels,omitempty"`
}

type liveQueryTargetCountBody struct {
	ReportID *int64                `json:"report_id,omitempty"`
	Selected liveQuerySelectedBody `json:"selected,omitzero"`
}

type liveQueryTargetCountOutputBody struct {
	TargetsCount   int `json:"targets_count"`
	TargetsOnline  int `json:"targets_online"`
	TargetsOffline int `json:"targets_offline"`
}

type liveQueryHandleBody struct {
	ID                int64     `json:"id"`
	SQL               string    `json:"sql"`
	StartedAt         time.Time `json:"started_at"`
	ResolvedHostCount int       `json:"resolved_host_count"`
}

type liveQueryCreateInput struct {
	Body liveQueryCreateBody
}

type liveQueryCreateOutput struct {
	Body liveQueryHandleBody
}

type liveQueryTargetCountInput struct {
	Body liveQueryTargetCountBody
}

type liveQueryTargetCountOutput struct {
	Body liveQueryTargetCountOutputBody
}

type liveQueryStreamInput struct {
	ID int64 `path:"id" minimum:"1"`
}

type liveQueryStopInput struct {
	ID int64 `path:"id" minimum:"1"`
}

type liveQueryStopOutput struct{}

type liveQueryPingEvent struct {
	Status string `json:"status"`
}

type liveQueryCompletedEvent struct {
	Status string `json:"status"`
}

type liveQueryResultEvent livequery.Event

// RegisterLiveQueries registers the one-shot live query create endpoint. The
// matching SSE stream endpoint is registered through Huma's SSE support.
func RegisterLiveQueries(
	api huma.API,
	manager *livequery.Manager,
	hostStore *hosts.Store,
) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-live-query",
		Method:        http.MethodPost,
		Path:          "/api/live-queries",
		Tags:          []string{liveQueriesTag},
		Summary:       "Start a live run against online hosts",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized},
	}, func(ctx context.Context, input *liveQueryCreateInput) (*liveQueryCreateOutput, error) {
		hostIDs, err := input.Body.resolveTargets(ctx, hostStore)
		if err != nil {
			return nil, err
		}
		handle := manager.Start(input.Body.SQL, hostIDs)
		return &liveQueryCreateOutput{Body: liveQueryHandleResponse(handle)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "count-live-query-targets",
		Method:      http.MethodPost,
		Path:        "/api/live-queries/targets/count",
		Tags:        []string{liveQueriesTag},
		Summary:     "Count live query targets",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized},
	}, func(ctx context.Context, input *liveQueryTargetCountInput) (*liveQueryTargetCountOutput, error) {
		selection, err := input.Body.Selected.targetSelection()
		if err != nil {
			return nil, err
		}
		metrics, err := hostStore.CountSelectedTargets(ctx, selection, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		return &liveQueryTargetCountOutput{Body: liveQueryTargetCountResponse(metrics)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "stop-live-query",
		Method:        http.MethodPost,
		Path:          "/api/live-queries/{id}/stop",
		Tags:          []string{liveQueriesTag},
		Summary:       "Stop a running live query",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *liveQueryStopInput) (*liveQueryStopOutput, error) {
		if err := manager.Stop(input.ID); err != nil {
			return nil, huma.Error404NotFound("live query not found")
		}
		return &liveQueryStopOutput{}, nil
	})

	sse.Register(api, huma.Operation{
		OperationID: "stream-live-query",
		Method:      http.MethodGet,
		Path:        "/api/live-queries/{id}/stream",
		Tags:        []string{liveQueriesTag},
		Summary:     "Stream live query results",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, map[string]any{
		"ping":      liveQueryPingEvent{},
		"result":    liveQueryResultEvent{},
		"completed": liveQueryCompletedEvent{},
	}, func(ctx context.Context, input *liveQueryStreamInput, send sse.Sender) {
		streamLiveQuery(ctx, manager, input.ID, send)
	})
}

func liveQueryHandleResponse(h livequery.Handle) liveQueryHandleBody {
	return liveQueryHandleBody{
		ID:                h.ID,
		SQL:               h.SQL,
		StartedAt:         h.StartedAt,
		ResolvedHostCount: h.ResolvedHostCount,
	}
}

func liveQueryTargetCountResponse(metrics hosts.TargetMetrics) liveQueryTargetCountOutputBody {
	return liveQueryTargetCountOutputBody{
		TargetsCount:   metrics.Total,
		TargetsOnline:  metrics.Online,
		TargetsOffline: metrics.Offline,
	}
}

func (body liveQueryCreateBody) resolveTargets(ctx context.Context, hostStore *hosts.Store) ([]int64, error) {
	if body.SQL == "" {
		return nil, huma.Error400BadRequest("sql is required")
	}
	selection, err := body.Selected.targetSelection()
	if err != nil {
		return nil, err
	}
	resolved, err := hostStore.ResolveOnlineSelectedTargets(ctx, selection, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, huma.Error400BadRequest("no online hosts targeted")
	}
	return resolved, nil
}

func (body liveQuerySelectedBody) targetSelection() (hosts.TargetSelection, error) {
	hostIDs, err := parseIDList(body.Hosts, "selected.hosts")
	if err != nil {
		return hosts.TargetSelection{}, err
	}
	labelIDs, err := parseIDList(body.Labels, "selected.labels")
	if err != nil {
		return hosts.TargetSelection{}, err
	}
	return hosts.TargetSelection{HostIDs: hostIDs, LabelIDs: labelIDs}, nil
}

func streamLiveQuery(
	ctx context.Context,
	manager *livequery.Manager,
	id int64,
	send sse.Sender,
) {
	events, release, err := manager.Subscribe(id)
	if err != nil {
		_ = send.Data(liveQueryCompletedEvent{Status: "completed"})
		return
	}
	defer release()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := send.Data(liveQueryPingEvent{Status: "ok"}); err != nil {
				return
			}
		case event, ok := <-events:
			if !ok {
				_ = send.Data(liveQueryCompletedEvent{Status: "completed"})
				return
			}
			if event.Status == "completed" {
				_ = send.Data(liveQueryCompletedEvent{Status: "completed"})
				return
			}
			if err := send.Data(liveQueryResultEvent(event)); err != nil {
				return
			}
		}
	}
}
