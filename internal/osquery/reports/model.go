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
	Platforms         []scope.Platform `json:"platforms"                     enum:"darwin,windows,linux" minItems:"1" nullable:"false"`
	MinOsqueryVersion *string          `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int              `json:"schedule_interval"`
	LabelScope        scope.LabelScope `json:"label_scope,omitzero"`
	CreatedByUserID   *int64           `json:"created_by_user_id,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// ReportCreate is a new report.
type ReportCreate struct {
	Name              string           `json:"name"`
	Description       string           `json:"description,omitempty"`
	Query             string           `json:"query"`
	Platforms         []scope.Platform `json:"platforms"                     enum:"darwin,windows,linux" minItems:"1" nullable:"false"`
	MinOsqueryVersion *string          `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int              `json:"schedule_interval,omitempty"`
	LabelScope        scope.LabelScope `json:"label_scope"`
	CreatedByUserID   *int64           `json:"-"`
}

// ReportUpdate is the editable report state.
type ReportUpdate struct {
	Name              string           `json:"name"`
	Description       string           `json:"description,omitempty"`
	Query             string           `json:"query"`
	Platforms         []scope.Platform `json:"platforms"                     enum:"darwin,windows,linux" minItems:"1" nullable:"false"`
	MinOsqueryVersion *string          `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int              `json:"schedule_interval,omitempty"`
	LabelScope        scope.LabelScope `json:"label_scope"`
}

// ReportListParams filters reports.
type ReportListParams struct {
	dbutil.ListParams

	Platform string
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
	HostResultCount int               `json:"n_host_results"`
}
