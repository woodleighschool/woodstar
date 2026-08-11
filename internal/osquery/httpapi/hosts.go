package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

const hostResource = "host"

type hostOsqueryChecksInput struct {
	api.ListQueryInput

	ID     int64                `path:"id"`
	Status []checks.CheckStatus `          query:"status,omitempty"`
}

func (input hostOsqueryChecksInput) params() checks.CheckResultListParams {
	return checks.CheckResultListParams{
		ListParams: input.Params(),
		Statuses:   input.Status,
	}
}

type hostOsqueryReportsInput struct {
	api.ListQueryInput

	ID     int64                        `path:"id"`
	Status reports.ReportSnapshotStatus `          query:"status,omitempty"`
}

func (input hostOsqueryReportsInput) params() reports.ReportSnapshotListParams {
	return reports.ReportSnapshotListParams{
		ListParams: input.Params(),
		Status:     input.Status,
	}
}

type hostOsqueryChecksOutput struct {
	Body api.Page[checks.CheckHostStatus]
}

type hostOsqueryReportsOutput struct {
	Body api.Page[reports.ReportSnapshot]
}

//nolint:dupl // Host checks and reports are distinct subresources; two parallel handlers do not justify generic registration machinery.
func registerHostOsqueryChecks(
	humaAPI huma.API,
	checkStore *checks.Store,
	hostStore *hosts.Store,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-host-osquery-checks",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/osquery/checks",
		Tags:        []string{api.TagHosts},
		Summary:     "List checks for a host",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostOsqueryChecksInput) (*hostOsqueryChecksOutput, error) {
		host, err := hostStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"list-host-osquery-checks",
				hostResource,
				err,
				"host_id",
				input.ID,
			)
		}
		rows, count, err := checkStore.HostChecks(ctx, host, input.params())
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"list-host-osquery-checks",
				err,
				"host_id",
				input.ID,
			)
		}
		return &hostOsqueryChecksOutput{
			Body: api.Page[checks.CheckHostStatus]{Items: rows, Count: count},
		}, nil
	})
}

//nolint:dupl // Host checks and reports are distinct subresources; two parallel handlers do not justify generic registration machinery.
func registerHostOsqueryReports(
	humaAPI huma.API,
	reportStore *reports.Store,
	hostStore *hosts.Store,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-host-osquery-reports",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/osquery/reports",
		Tags:        []string{api.TagHosts},
		Summary:     "List reports for a host",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostOsqueryReportsInput) (*hostOsqueryReportsOutput, error) {
		host, err := hostStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"list-host-osquery-reports",
				hostResource,
				err,
				"host_id",
				input.ID,
			)
		}
		rows, count, err := reportStore.HostSnapshots(ctx, host, input.params())
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"list-host-osquery-reports",
				err,
				"host_id",
				input.ID,
			)
		}
		return &hostOsqueryReportsOutput{
			Body: api.Page[reports.ReportSnapshot]{Items: rows, Count: count},
		}, nil
	})
}
