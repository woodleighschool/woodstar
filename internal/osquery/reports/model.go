package reports

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/scope"
)

// Report is a saved osquery snapshot query.
type Report struct {
	ID                int64            `json:"id"`
	Name              string           `json:"name"`
	Description       string           `json:"description"`
	Query             string           `json:"query"`
	MinOsqueryVersion *string          `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int32            `json:"schedule_interval"`
	LabelScope        scope.LabelScope `json:"label_scope,omitzero"`
	CreatedByUserID   *int64           `json:"created_by_user_id,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// ReportMutation is the editable report state used by create and update.
type ReportMutation struct {
	Name              string           `json:"name"`
	Description       string           `json:"description,omitempty"`
	Query             string           `json:"query"`
	MinOsqueryVersion *string          `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int32            `json:"schedule_interval,omitempty"`
	LabelScope        scope.LabelScope `json:"label_scope"`
	CreatedByUserID   *int64           `json:"-"`
}

// ReportListParams filters reports.
type ReportListParams struct {
	dbutil.ListParams
}

// ReportResult is one saved result row.
type ReportResult struct {
	ReportID    int64             `json:"report_id"`
	ReportName  string            `json:"report_name"`
	HostID      int64             `json:"host_id"`
	HostName    string            `json:"host_name"`
	Columns     map[string]string `json:"columns"`
	LastFetched time.Time         `json:"last_fetched,omitzero"`
}

// HostReport is one report on a host detail page.
type HostReport struct {
	ReportID        int64             `json:"report_id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	LastFetched     *time.Time        `json:"last_fetched,omitempty"`
	FirstResult     map[string]string `json:"first_result,omitempty"`
	HostResultCount int32             `json:"n_host_results"`
}
