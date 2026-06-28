package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/checks"
	"github.com/woodleighschool/woodstar/internal/osquery/reports"
)

type hostOsqueryChecksInput struct {
	ID int64 `path:"id"`
}

type hostOsqueryReportsInput struct {
	ID int64 `path:"id"`
}

type hostReportsOutput struct {
	Body []reports.HostReport
}

type hostReportResultsInput struct {
	ID       int64 `path:"id"`
	ReportID int64 `path:"report_id"`
}

type hostReportResultsOutput struct {
	Body hostReportResultsBody
}

type hostReportResultsBody struct {
	ReportID    int64                  `json:"report_id"`
	ID          int64                  `json:"host_id"`
	HostName    string                 `json:"host_name"`
	LastFetched *time.Time             `json:"last_fetched,omitempty"`
	Items       []reports.ReportResult `json:"items"`
}

func registerHostOsqueryChecks(
	api huma.API,
	checkStore *checks.Store,
	hostStore *hosts.Store,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-osquery-checks",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/osquery/checks",
		Tags:        []string{checksTag, hostsTag},
		Summary:     "List checks for a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostOsqueryChecksInput) (*checkResultsOutput, error) {
		host, err := hostStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, resourceError(ctx, logger, "list-host-osquery-checks", hostResource, err, "host_id", input.ID)
		}
		rows, err := checkStore.HostChecks(ctx, host)
		if err != nil {
			return nil, handlerError(ctx, logger, "list-host-osquery-checks", err, "host_id", input.ID)
		}
		return &checkResultsOutput{Body: rows}, nil
	})
}

func registerHostOsqueryReports(
	api huma.API,
	reportStore *reports.Store,
	hostStore *hosts.Store,
	logger *slog.Logger,
) {
	registerHostReports(api, reportStore, hostStore, logger)
	registerHostReportResults(api, reportStore, hostStore, logger)
}

func registerHostReports(api huma.API, reportStore *reports.Store, hostStore *hosts.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-osquery-reports",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/osquery/reports",
		Tags:        []string{reportsTag, hostsTag},
		Summary:     "List reports for a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostOsqueryReportsInput) (*hostReportsOutput, error) {
		host, err := hostStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, resourceError(ctx, logger, "list-host-osquery-reports", hostResource, err, "host_id", input.ID)
		}
		rows, err := reportStore.HostReports(ctx, host)
		if err != nil {
			return nil, handlerError(ctx, logger, "list-host-osquery-reports", err, "host_id", input.ID)
		}
		return &hostReportsOutput{Body: rows}, nil
	})
}

func registerHostReportResults(
	api huma.API,
	reportStore *reports.Store,
	hostStore *hosts.Store,
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-osquery-report-results",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/osquery/reports/{report_id}",
		Tags:        []string{reportsTag, hostsTag},
		Summary:     "List report rows for one host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostReportResultsInput) (*hostReportResultsOutput, error) {
		host, err := hostStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, resourceError(ctx, logger, "list-host-osquery-report-results", hostResource, err, "host_id", input.ID)
		}
		rows, lastFetched, err := reportStore.HostResults(ctx, input.ID, input.ReportID)
		if err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"list-host-osquery-report-results",
				err,
				"host_id", input.ID,
				"report_id", input.ReportID,
			)
		}
		return &hostReportResultsOutput{Body: hostReportResultsBody{
			ReportID:    input.ReportID,
			ID:          input.ID,
			HostName:    host.DisplayName,
			LastFetched: lastFetched,
			Items:       rows,
		}}, nil
	})
}
