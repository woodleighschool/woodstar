package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

const (
	reportsTag     = "Reports"
	reportResource = "report"
	reportIDPath   = "/api/osquery/reports/{id}"
)

type reportListInput struct {
	ListQueryInput
}

type reportGetInput struct {
	ID string `path:"id"`
}

type reportCreateInput struct {
	Body reports.ReportCreate
}

type reportPutInput struct {
	ID   string `path:"id"`
	Body reports.ReportUpdate
}

type reportDeleteInput struct {
	ID string `path:"id"`
}

type reportBulkDeleteInput struct {
	Body bulkIDsBody
}

type reportListOutput struct {
	Body paginatedBody[reports.Report]
}

type reportOutput struct {
	Body reports.Report
}

type reportResultsOutput struct {
	Body itemsBody[reports.ReportResult]
}

type hostReportsOutput struct {
	Body itemsBody[reports.HostReport]
}

type hostReportResultsOutput struct {
	Body hostReportResultsBody
}

type hostReportResultsBody struct {
	ReportID    int64                  `json:"report_id"`
	HostID      int64                  `json:"host_id"`
	HostName    string                 `json:"host_name"`
	LastFetched *time.Time             `json:"last_fetched,omitempty"`
	Items       []reports.ReportResult `json:"items"`
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
		return &reportListOutput{Body: paginatedBody[reports.Report]{Items: items, Count: count}}, nil
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
		params := input.Body
		params.CreatedByUserID = currentUserID(ctx)
		lscope, err := normalizeLabelScope(params.LabelScope)
		if err != nil {
			return nil, err
		}
		params.LabelScope = lscope
		report, err := reportStore.Create(ctx, params)
		if err != nil {
			return nil, resourceMutationError(reportResource, err)
		}
		return &reportOutput{Body: *report}, nil
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
		return &reportOutput{Body: *report}, nil
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
		params := input.Body
		lscope, err := normalizeLabelScope(params.LabelScope)
		if err != nil {
			return nil, err
		}
		params.LabelScope = lscope
		report, err := reportStore.Update(ctx, id, params)
		if err != nil {
			return nil, resourceMutationError(reportResource, err)
		}
		return &reportOutput{Body: *report}, nil
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
		return &reportResultsOutput{Body: itemsBody[reports.ReportResult]{Items: rows}}, nil
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
		return &hostReportsOutput{Body: itemsBody[reports.HostReport]{Items: rows}}, nil
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
			Items:       rows,
		}}, nil
	})
}

func (input reportListInput) params() reports.ReportListParams {
	return reports.ReportListParams{
		ListParams: input.ListQueryInput.params(),
	}
}
