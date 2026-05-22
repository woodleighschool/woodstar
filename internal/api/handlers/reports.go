//nolint:dupl // Huma route registration is intentionally explicit per resource.
package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/agents/reports"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/scope"
)

const (
	reportsTag     = "Reports"
	reportResource = "report"
	reportIDPath   = "/api/reports/{id}"
)

type reportMutationBody struct {
	Name              string           `json:"name"`
	Description       string           `json:"description,omitempty"`
	Query             string           `json:"query"`
	Platform          *string          `json:"platform,omitempty"`
	MinOsqueryVersion *string          `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int              `json:"schedule_interval,omitempty"`
	LabelScope        scope.LabelScope `json:"label_scope"`
}

type reportBody struct {
	ID                int64            `json:"id"`
	Name              string           `json:"name"`
	Description       string           `json:"description"`
	Query             string           `json:"query"`
	Platform          *string          `json:"platform,omitempty"`
	MinOsqueryVersion *string          `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int              `json:"schedule_interval"`
	LabelScope        scope.LabelScope `json:"label_scope,omitzero"`
	CreatedByUserID   *int64           `json:"created_by_user_id,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
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
	Q              string `query:"q,omitempty"`
	Platform       string `query:"platform,omitempty"`
	Page           int    `query:"page,omitempty"`
	PerPage        int    `query:"per_page,omitempty"`
	OrderKey       string `query:"order_key,omitempty"`
	OrderDirection string `query:"order_direction,omitempty"`
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
	Body struct {
		Items []reportBody `json:"items"`
		Count int          `json:"count"`
	}
}

type reportOutput struct {
	Body reportBody
}

type reportResultsOutput struct {
	Body struct {
		Items []reportResultBody `json:"items"`
	}
}

type hostReportsOutput struct {
	Body struct {
		Items []reports.HostReport `json:"items"`
	}
}

type hostReportResultsOutput struct {
	Body struct {
		ReportID    int64              `json:"report_id"`
		HostID      int64              `json:"host_id"`
		HostName    string             `json:"host_name"`
		LastFetched *time.Time         `json:"last_fetched,omitempty"`
		Items       []reportResultBody `json:"items"`
	}
}

type hostReportResultsInput struct {
	ID       string `path:"id"`
	ReportID string `path:"report_id"`
}

// RegisterReports registers saved report endpoints.
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
		OperationID: "list-reports",
		Method:      http.MethodGet,
		Path:        "/api/reports",
		Tags:        []string{reportsTag},
		Summary:     "List reports",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *reportListInput) (*reportListOutput, error) {
		items, count, err := reportStore.List(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(reportResource, err)
		}
		out := &reportListOutput{}
		out.Body.Items = reportBodies(items)
		out.Body.Count = count
		return out, nil
	})
}

func registerCreateReport(api huma.API, reportStore *reports.Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-report",
		Method:        http.MethodPost,
		Path:          "/api/reports",
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
		OperationID: "get-report",
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
		OperationID: "put-report",
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
		OperationID: "delete-report",
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
		OperationID: "bulk-delete-reports",
		Method:      http.MethodPost,
		Path:        "/api/reports/bulk-delete",
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
		OperationID: "list-report-results",
		Method:      http.MethodGet,
		Path:        "/api/reports/{id}/results",
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
		out := &reportResultsOutput{}
		out.Body.Items = reportResultBodies(rows)
		return out, nil
	})
}

func registerHostReports(api huma.API, reportStore *reports.Store, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-reports",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/reports",
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
		out := &hostReportsOutput{}
		out.Body.Items = rows
		return out, nil
	})
}

func registerHostReportResults(api huma.API, reportStore *reports.Store, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-report-results",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/reports/{report_id}",
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
		out := &hostReportResultsOutput{}
		out.Body.ReportID = reportID
		out.Body.HostID = hostID
		out.Body.HostName = host.DisplayName
		out.Body.LastFetched = lastFetched
		out.Body.Items = reportResultBodies(rows)
		return out, nil
	})
}

func (input reportListInput) params() reports.ReportListParams {
	return reports.ReportListParams{
		ListParams: dbutil.CleanListParams(dbutil.ListParams{
			Q:              input.Q,
			Page:           input.Page,
			PerPage:        input.PerPage,
			OrderKey:       input.OrderKey,
			OrderDirection: input.OrderDirection,
		}),
		Platform: strings.TrimSpace(input.Platform),
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
		Platform:          body.Platform,
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
		Platform:          body.Platform,
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
		Platform:          report.Platform,
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
