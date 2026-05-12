package queries

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/scope"
)

// QueryLoggingType is the storage mode for scheduled query results.
type QueryLoggingType string

const (
	QueryLoggingSnapshot QueryLoggingType = "snapshot"
)

// Query is admin-authored osquery SQL. Scheduled snapshot queries are reports.
type Query struct {
	ID                int64            `json:"id"`
	Name              string           `json:"name"`
	Description       string           `json:"description"`
	Query             string           `json:"query"`
	Platform          *string          `json:"platform,omitempty"`
	MinOsqueryVersion *string          `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int              `json:"schedule_interval"`
	LoggingType       QueryLoggingType `json:"-"`
	LabelScope        scope.LabelScope `json:"label_scope,omitzero"`
	CreatedByUserID   *int64           `json:"created_by_user_id,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// QueryCreate contains editable query fields.
type QueryCreate struct {
	Name              string
	Description       string
	Query             string
	Platform          *string
	MinOsqueryVersion *string
	ScheduleInterval  int
	LoggingType       QueryLoggingType
	LabelScope        scope.LabelScope
	CreatedByUserID   *int64
}

// QueryUpdate replaces editable query fields.
type QueryUpdate QueryCreate

// QueryListParams filters saved query lists.
type QueryListParams struct {
	dbutil.ListParams

	Platform string
}

// QueryResult is one stored report row from one host.
type QueryResult struct {
	QueryID     int64             `json:"query_id"`
	QueryName   string            `json:"query_name"`
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
