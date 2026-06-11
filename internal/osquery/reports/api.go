package reports

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/adminctx"
	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
)

const (
	reportsTag     = "Reports"
	reportResource = "report"
	reportIDPath   = "/api/osquery/reports/{id}"
)

type reportListInput struct {
	apitypes.ListQueryInput
}

type reportGetInput struct {
	ReportID int64 `path:"id"`
}

type reportCreateInput struct {
	Body ReportMutation
}

type reportPutInput struct {
	ReportID int64 `path:"id"`
	Body     ReportMutation
}

type reportDeleteInput struct {
	ReportID int64 `path:"id"`
}

type reportBulkDeleteInput struct {
	Body apitypes.BulkIDsBody
}

type reportListOutput struct {
	Body apitypes.Page[Report]
}

type reportOutput struct {
	Body Report
}

type reportResultsOutput struct {
	Body []ReportResult
}

func RegisterAdminRoutes(api huma.API, reportStore *Store) {
	registerListReports(api, reportStore)
	registerCreateReport(api, reportStore)
	registerGetReport(api, reportStore)
	registerUpdateReport(api, reportStore)
	registerDeleteReport(api, reportStore)
	registerBulkDeleteReports(api, reportStore)
	registerReportResults(api, reportStore)
}

func registerListReports(api huma.API, reportStore *Store) {
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
			return nil, apitypes.ResourceMutationError(reportResource, err)
		}
		return &reportListOutput{Body: apitypes.Page[Report]{Items: items, Count: count}}, nil
	})
}

func registerCreateReport(api huma.API, reportStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-osquery-report",
		Method:        http.MethodPost,
		Path:          "/api/osquery/reports",
		Tags:          []string{reportsTag},
		Summary:       "Create a report",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *reportCreateInput) (*reportOutput, error) {
		report, err := reportStore.Create(ctx, input.Body, adminctx.CurrentUserID(ctx))
		if err != nil {
			return nil, apitypes.ResourceMutationError(reportResource, err)
		}
		return &reportOutput{Body: *report}, nil
	})
}

func registerGetReport(api huma.API, reportStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-osquery-report",
		Method:      http.MethodGet,
		Path:        reportIDPath,
		Tags:        []string{reportsTag},
		Summary:     "Get a report",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *reportGetInput) (*reportOutput, error) {
		report, err := reportStore.GetByID(ctx, input.ReportID)
		if err != nil {
			return nil, apitypes.ResourceMutationError(reportResource, err)
		}
		return &reportOutput{Body: *report}, nil
	})
}

func registerUpdateReport(api huma.API, reportStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "update-osquery-report",
		Method:      http.MethodPut,
		Path:        reportIDPath,
		Tags:        []string{reportsTag},
		Summary:     "Replace a report",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *reportPutInput) (*reportOutput, error) {
		report, err := reportStore.Update(ctx, input.ReportID, input.Body)
		if err != nil {
			return nil, apitypes.ResourceMutationError(reportResource, err)
		}
		return &reportOutput{Body: *report}, nil
	})
}

func registerDeleteReport(api huma.API, reportStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-osquery-report",
		Method:      http.MethodDelete,
		Path:        reportIDPath,
		Tags:        []string{reportsTag},
		Summary:     "Delete a report",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *reportDeleteInput) (*struct{}, error) {
		if err := reportStore.Delete(ctx, input.ReportID); err != nil {
			return nil, apitypes.ResourceMutationError(reportResource, err)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteReports(api huma.API, reportStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "bulk-delete-osquery-reports",
		Method:      http.MethodPost,
		Path:        "/api/osquery/reports/bulk-delete",
		Tags:        []string{reportsTag},
		Summary:     "Delete reports",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *reportBulkDeleteInput) (*struct{}, error) {
		if _, err := reportStore.DeleteMany(ctx, input.Body.IDs); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func registerReportResults(api huma.API, reportStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-osquery-report-results",
		Method:      http.MethodGet,
		Path:        "/api/osquery/reports/{id}/results",
		Tags:        []string{reportsTag},
		Summary:     "List latest snapshots for a report",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *reportGetInput) (*reportResultsOutput, error) {
		rows, err := reportStore.Results(ctx, input.ReportID)
		if err != nil {
			return nil, err
		}
		return &reportResultsOutput{Body: rows}, nil
	})
}

func (input reportListInput) params() ReportListParams {
	return ReportListParams{
		ListParams: input.ListQueryInput.Params(),
	}
}
