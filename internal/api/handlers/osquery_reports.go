package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

const (
	reportResource = "report"
	reportIDPath   = "/api/osquery/reports/{id}"
)

type reportListInput struct {
	ListQueryInput
}

func (input reportListInput) params() reports.ReportListParams {
	return reports.ReportListParams{
		ListParams: input.ListQueryInput.params(),
	}
}

type reportGetInput struct {
	ID int64 `path:"id"`
}

type reportCreateInput struct {
	Body reports.ReportMutation
}

type reportPutInput struct {
	ID   int64 `path:"id"`
	Body reports.ReportMutation
}

type reportDeleteInput struct {
	ID int64 `path:"id"`
}

type reportListOutput struct {
	Body Page[reports.Report]
}

type reportOutput struct {
	Body reports.Report
}

type reportResultsOutput struct {
	Body []reports.ReportResult
}

func registerOsqueryReports(api huma.API, reportStore *reports.Store, logger *slog.Logger) {
	registerListReports(api, reportStore, logger)
	registerCreateReport(api, reportStore, logger)
	registerGetReport(api, reportStore, logger)
	registerUpdateReport(api, reportStore, logger)
	registerDeleteReport(api, reportStore, logger)
	registerBulkDeleteReports(api, reportStore, logger)
	registerReportResults(api, reportStore, logger)
}

func registerListReports(api huma.API, reportStore *reports.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-osquery-reports",
		Method:      http.MethodGet,
		Path:        "/api/osquery/reports",
		Tags:        []string{osqueryReportsTag},
		Summary:     "List reports",
	}, func(ctx context.Context, input *reportListInput) (*reportListOutput, error) {
		items, count, err := reportStore.List(ctx, input.params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-osquery-reports", reportResource, err)
		}
		return &reportListOutput{Body: Page[reports.Report]{Items: items, Count: count}}, nil
	})
}

func registerCreateReport(api huma.API, reportStore *reports.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-osquery-report",
		Method:        http.MethodPost,
		Path:          "/api/osquery/reports",
		Tags:          []string{osqueryReportsTag},
		Summary:       "Create a report",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *reportCreateInput) (*reportOutput, error) {
		report, err := reportStore.Create(ctx, reports.ReportCreateMutation{
			ReportMutation:  input.Body,
			CreatedByUserID: ctxkeys.CurrentUserID(ctx),
		})
		if err != nil {
			return nil, resourceError(ctx, logger, "create-osquery-report", reportResource, err)
		}
		return &reportOutput{Body: *report}, nil
	})
}

func registerGetReport(api huma.API, reportStore *reports.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-osquery-report",
		Method:      http.MethodGet,
		Path:        reportIDPath,
		Tags:        []string{osqueryReportsTag},
		Summary:     "Get a report",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *reportGetInput) (*reportOutput, error) {
		report, err := reportStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"get-osquery-report",
				reportResource,
				err,
				"id",
				input.ID,
			)
		}
		return &reportOutput{Body: *report}, nil
	})
}

func registerUpdateReport(api huma.API, reportStore *reports.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "update-osquery-report",
		Method:      http.MethodPut,
		Path:        reportIDPath,
		Tags:        []string{osqueryReportsTag},
		Summary:     "Update a report",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *reportPutInput) (*reportOutput, error) {
		report, err := reportStore.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"update-osquery-report",
				reportResource,
				err,
				"id",
				input.ID,
			)
		}
		return &reportOutput{Body: *report}, nil
	})
}

func registerDeleteReport(api huma.API, reportStore *reports.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-osquery-report",
		Method:      http.MethodDelete,
		Path:        reportIDPath,
		Tags:        []string{osqueryReportsTag},
		Summary:     "Delete a report",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *reportDeleteInput) (*struct{}, error) {
		if err := reportStore.Delete(ctx, input.ID); err != nil {
			return nil, resourceError(
				ctx,
				logger,
				"delete-osquery-report",
				reportResource,
				err,
				"id",
				input.ID,
			)
		}
		return &struct{}{}, nil
	})
}

func registerBulkDeleteReports(api huma.API, reportStore *reports.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID:   "bulk-delete-osquery-reports",
		Method:        http.MethodDelete,
		Path:          "/api/osquery/reports",
		Tags:          []string{osqueryReportsTag},
		Summary:       "Delete reports",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *deleteManyInput) (*struct{}, error) {
		if _, err := reportStore.DeleteMany(ctx, input.IDs); err != nil {
			return nil, handlerError(ctx, logger, "bulk-delete-osquery-reports", err)
		}
		return &struct{}{}, nil
	})
}

func registerReportResults(api huma.API, reportStore *reports.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-osquery-report-results",
		Method:      http.MethodGet,
		Path:        "/api/osquery/reports/{id}/results",
		Tags:        []string{osqueryReportsTag},
		Summary:     "List report results",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *reportGetInput) (*reportResultsOutput, error) {
		rows, err := reportStore.Results(ctx, input.ID)
		if err != nil {
			return nil, handlerError(ctx, logger, "list-osquery-report-results", err, "id", input.ID)
		}
		return &reportResultsOutput{Body: rows}, nil
	})
}
