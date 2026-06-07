package reports

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

type hostReportsOutput struct {
	Body []HostReport
}

type hostReportResultsOutput struct {
	Body hostReportResultsBody
}

type hostReportResultsBody struct {
	ReportID    int64          `json:"report_id"`
	HostID      int64          `json:"host_id"`
	HostName    string         `json:"host_name"`
	LastFetched *time.Time     `json:"last_fetched,omitempty"`
	Items       []ReportResult `json:"items"`
}

type hostReportResultsInput struct {
	HostID   int64 `path:"id"`
	ReportID int64 `path:"report_id"`
}

const hostsTag = "Hosts"

type hostGetInput struct {
	HostID int64 `path:"id"`
}

func RegisterHostAdminRoutes(api huma.API, reportStore *Store, hostStore *hosts.Store) {
	registerHostReports(api, reportStore, hostStore)
	registerHostReportResults(api, reportStore, hostStore)
}

func registerHostReports(api huma.API, reportStore *Store, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-osquery-reports",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/osquery/reports",
		Tags:        []string{reportsTag, hostsTag},
		Summary:     "List reports for a host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostGetInput) (*hostReportsOutput, error) {
		host, err := hostStore.GetByID(ctx, input.HostID)
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
		return &hostReportsOutput{Body: rows}, nil
	})
}

func registerHostReportResults(api huma.API, reportStore *Store, hostStore *hosts.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-host-osquery-report-results",
		Method:      http.MethodGet,
		Path:        "/api/hosts/{id}/osquery/reports/{report_id}",
		Tags:        []string{reportsTag, hostsTag},
		Summary:     "List report rows for one host",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostReportResultsInput) (*hostReportResultsOutput, error) {
		host, err := hostStore.GetByID(ctx, input.HostID)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		}
		if err != nil {
			return nil, err
		}
		rows, lastFetched, err := reportStore.HostResults(ctx, input.HostID, input.ReportID)
		if err != nil {
			return nil, err
		}
		return &hostReportResultsOutput{Body: hostReportResultsBody{
			ReportID:    input.ReportID,
			HostID:      input.HostID,
			HostName:    host.DisplayName,
			LastFetched: lastFetched,
			Items:       rows,
		}}, nil
	})
}
