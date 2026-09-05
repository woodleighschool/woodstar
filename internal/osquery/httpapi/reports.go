package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/woodleighschool/goodies/auth/authn"

	"github.com/woodleighschool/woodstar/internal/activity"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

const (
	reportResource = "report"
	reportIDPath   = "/api/osquery/reports/{id}"
)

type reportListInput struct {
	api.ListQueryInput
}

func (input reportListInput) params() reports.ReportListParams {
	return reports.ReportListParams{
		ListParams: input.Params(),
	}
}

type reportGetInput struct {
	ID int64 `path:"id"`
}

type reportSnapshotsInput struct {
	api.ListQueryInput

	ID     int64                        `path:"id"`
	Status reports.ReportSnapshotStatus `          query:"status,omitempty"`
}

func (input reportSnapshotsInput) params() reports.ReportSnapshotListParams {
	return reports.ReportSnapshotListParams{
		ListParams: input.Params(),
		Status:     input.Status,
	}
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
	Body api.Page[reports.Report]
}

type reportOutput struct {
	Body reports.Report
}

type reportSnapshotsOutput struct {
	Body api.Page[reports.ReportSnapshot]
}

func registerOsqueryReports(
	humaAPI huma.API,
	reportStore *reports.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	registerListReports(humaAPI, reportStore, logger)
	registerCreateReport(humaAPI, reportStore, activityRecorder, logger)
	registerGetReport(humaAPI, reportStore, logger)
	registerUpdateReport(humaAPI, reportStore, activityRecorder, logger)
	registerDeleteReport(humaAPI, reportStore, activityRecorder, logger)
	registerBulkDeleteReports(humaAPI, reportStore, activityRecorder, logger)
	registerReportSnapshots(humaAPI, reportStore, logger)
}

func registerListReports(humaAPI huma.API, reportStore *reports.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-osquery-reports",
		Method:      http.MethodGet,
		Path:        "/api/osquery/reports",
		Tags:        []string{api.TagOsqueryReports},
		Summary:     "List reports",
	}, func(ctx context.Context, input *reportListInput) (*reportListOutput, error) {
		items, count, err := reportStore.List(ctx, input.params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-osquery-reports", reportResource, err)
		}
		return &reportListOutput{Body: api.Page[reports.Report]{Items: items, Count: count}}, nil
	})
}

func registerCreateReport(
	humaAPI huma.API,
	reportStore *reports.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-osquery-report",
		Method:        http.MethodPost,
		Path:          "/api/osquery/reports",
		Tags:          []string{api.TagOsqueryReports},
		Summary:       "Create a report",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *reportCreateInput) (*reportOutput, error) {
		report, err := reportStore.Create(ctx, reports.ReportCreateMutation{
			ReportMutation:  input.Body,
			CreatedByUserID: authn.CurrentPrincipalID(ctx),
		})
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "create-osquery-report", reportResource, err)
		}
		activity.RecordUser(ctx, activityRecorder, logger, activity.AreaOsquery, activity.ActionReportCreated,
			activity.Resource(reportResource, report.ID, report.Name))
		return &reportOutput{Body: *report}, nil
	})
}

func registerGetReport(humaAPI huma.API, reportStore *reports.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-osquery-report",
		Method:      http.MethodGet,
		Path:        reportIDPath,
		Tags:        []string{api.TagOsqueryReports},
		Summary:     "Get a report",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *reportGetInput) (*reportOutput, error) {
		report, err := reportStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(
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

//nolint:dupl // Policy and report handlers intentionally keep their domain contracts explicit.
func registerUpdateReport(
	humaAPI huma.API,
	reportStore *reports.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-osquery-report",
		Method:      http.MethodPut,
		Path:        reportIDPath,
		Tags:        []string{api.TagOsqueryReports},
		Summary:     "Update a report",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, input *reportPutInput) (*reportOutput, error) {
		report, err := reportStore.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"update-osquery-report",
				reportResource,
				err,
				"id",
				input.ID,
			)
		}
		activity.RecordUser(ctx, activityRecorder, logger, activity.AreaOsquery, activity.ActionReportUpdated,
			activity.Resource(reportResource, report.ID, report.Name))
		return &reportOutput{Body: *report}, nil
	})
}

func registerDeleteReport(
	humaAPI huma.API,
	reportStore *reports.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "delete-osquery-report",
		Method:      http.MethodDelete,
		Path:        reportIDPath,
		Tags:        []string{api.TagOsqueryReports},
		Summary:     "Delete a report",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *reportDeleteInput) (*struct{}, error) {
		report, err := reportStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "delete-osquery-report", reportResource, err, "id", input.ID)
		}
		if err := reportStore.Delete(ctx, input.ID); err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"delete-osquery-report",
				reportResource,
				err,
				"id",
				input.ID,
			)
		}
		activity.RecordUser(ctx, activityRecorder, logger, activity.AreaOsquery, activity.ActionReportDeleted,
			activity.Resource(reportResource, report.ID, report.Name))
		return &struct{}{}, nil
	})
}

func registerBulkDeleteReports(
	humaAPI huma.API,
	reportStore *reports.Store,
	activityRecorder activity.Recorder,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "bulk-delete-osquery-reports",
		Method:        http.MethodDelete,
		Path:          "/api/osquery/reports",
		Tags:          []string{api.TagOsqueryReports},
		Summary:       "Delete reports",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *api.DeleteManyInput) (*struct{}, error) {
		deleted, err := reportStore.DeleteMany(ctx, input.IDs)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "bulk-delete-osquery-reports", err)
		}
		if deleted > 0 {
			activity.RecordUser(ctx, activityRecorder, logger, activity.AreaOsquery, activity.ActionReportsDeleted,
				activity.Collection(reportResource, fmt.Sprintf("%d reports", deleted)))
		}
		return &struct{}{}, nil
	})
}

func registerReportSnapshots(humaAPI huma.API, reportStore *reports.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-osquery-report-snapshots",
		Method:      http.MethodGet,
		Path:        "/api/osquery/reports/{id}/snapshots",
		Tags:        []string{api.TagOsqueryReports},
		Summary:     "List report snapshots",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *reportSnapshotsInput) (*reportSnapshotsOutput, error) {
		rows, count, err := reportStore.Snapshots(ctx, input.ID, input.params())
		if err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"list-osquery-report-snapshots",
				reportResource,
				err,
				"id",
				input.ID,
			)
		}
		return &reportSnapshotsOutput{
			Body: api.Page[reports.ReportSnapshot]{Items: rows, Count: count},
		}, nil
	})
}
