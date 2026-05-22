package reports

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/scope"
)

// Report is an admin-authored, scheduled osquery snapshot query whose results
// are persisted per host.
type Report struct {
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

// ReportCreate contains editable report fields.
type ReportCreate struct {
	Name              string
	Description       string
	Query             string
	Platform          *string
	MinOsqueryVersion *string
	ScheduleInterval  int
	LabelScope        scope.LabelScope
	CreatedByUserID   *int64
}

// ReportUpdate replaces editable report fields.
type ReportUpdate struct {
	Name              string
	Description       string
	Query             string
	Platform          *string
	MinOsqueryVersion *string
	ScheduleInterval  int
	LabelScope        scope.LabelScope
}

// ReportListParams filters saved report lists.
type ReportListParams struct {
	dbutil.ListParams

	Platform string
}

// ReportResult is one stored snapshot row from one host.
type ReportResult struct {
	ReportID    int64             `json:"report_id"`
	ReportName  string            `json:"report_name"`
	HostID      int64             `json:"host_id"`
	HostName    string            `json:"host_name"`
	Columns     map[string]string `json:"columns"`
	LastFetched time.Time         `json:"last_fetched,omitzero"`
}

// HostReport is a scheduled report as it appears on one host detail page.
type HostReport struct {
	ReportID        int64             `json:"report_id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	LastFetched     *time.Time        `json:"last_fetched,omitempty"`
	FirstResult     map[string]string `json:"first_result,omitempty"`
	HostResultCount int               `json:"n_host_results"`
}
