package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/policies"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

const hostResource = "host"

type hostOsqueryPoliciesInput struct {
	api.ListQueryInput

	ID     int64                   `path:"id"`
	Status []policies.PolicyStatus `          query:"status,omitempty"`
}

func (input hostOsqueryPoliciesInput) params() policies.PolicyResultListParams {
	return policies.PolicyResultListParams{
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

type hostOsqueryPoliciesOutput struct {
	Body api.Page[policies.PolicyHostStatus]
}

type hostOsqueryReportsOutput struct {
	Body api.Page[reports.ReportSnapshot]
}

//nolint:dupl // Host policies and reports are distinct subresources; two parallel handlers do not justify generic registration machinery.
func registerHostOsqueryPolicies(
	humaAPI huma.API,
	policyStore *policies.Store,
	hostStore *hosts.Store,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-host-osquery-policies",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/osquery/policies",
		Tags:        []string{api.TagHosts},
		Summary:     "List policies for a host",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostOsqueryPoliciesInput) (*hostOsqueryPoliciesOutput, error) {
		host, err := hostStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(
				ctx,
				logger,
				"list-host-osquery-policies",
				hostResource,
				err,
				"host_id",
				input.ID,
			)
		}
		rows, count, err := policyStore.HostPolicies(ctx, host, input.params())
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"list-host-osquery-policies",
				err,
				"host_id",
				input.ID,
			)
		}
		return &hostOsqueryPoliciesOutput{
			Body: api.Page[policies.PolicyHostStatus]{Items: rows, Count: count},
		}, nil
	})
}

//nolint:dupl // Host policies and reports are distinct subresources; two parallel handlers do not justify generic registration machinery.
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
