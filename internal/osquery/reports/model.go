package reports

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/openapischema"
	"github.com/woodleighschool/woodstar/internal/validation"
)

var osqueryVersionRE = regexp.MustCompile(`^(0|[1-9][0-9]*)[.](0|[1-9][0-9]*)[.](0|[1-9][0-9]*)$`)

// Report is a saved osquery snapshot query.
type Report struct {
	ID                 int64                  `json:"id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	Query              string                 `json:"query"`
	MinOsqueryVersion  *string                `json:"min_osquery_version,omitempty"`
	ScheduleInterval   int32                  `json:"schedule_interval"`
	Targets            ReportTargets          `json:"targets"`
	CollectedHostCount int32                  `json:"collected_host_count"`
	ErrorHostCount     int32                  `json:"error_host_count"`
	PendingHostCount   int32                  `json:"pending_host_count"`
	CreatedBy          *directory.UserSummary `json:"created_by,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// ReportMutation is the editable report state used by create and update.
type ReportMutation struct {
	Name              string        `json:"name"                          validate:"required,notblank" minLength:"1"`
	Description       string        `json:"description,omitempty"`
	Query             string        `json:"query"                         validate:"required,notblank" minLength:"1"`
	MinOsqueryVersion *string       `json:"min_osquery_version,omitempty" pattern:"^(0|[1-9][0-9]*)[.](0|[1-9][0-9]*)[.](0|[1-9][0-9]*)$"`
	ScheduleInterval  int32         `json:"schedule_interval,omitempty"   validate:"gte=0"`
	Targets           ReportTargets `json:"targets"`
}

// ReportCreateMutation is the create input; it embeds ReportMutation and
// carries the optional creator user ID which is not caller-settable via the API.
type ReportCreateMutation struct {
	ReportMutation

	CreatedByUserID *int64
}

func (p *ReportMutation) Validate() error {
	if err := validation.Struct(p); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	if p.MinOsqueryVersion != nil && !osqueryVersionRE.MatchString(*p.MinOsqueryVersion) {
		return fmt.Errorf(
			"%w: minimum osquery version must use X.Y.Z",
			fault.ErrInvalidInput,
		)
	}
	if err := p.Targets.validate(); err != nil {
		return err
	}
	return nil
}

func (p *ReportMutation) normalize() {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.Query = strings.TrimSpace(p.Query)
	if p.MinOsqueryVersion != nil {
		version := strings.TrimSpace(*p.MinOsqueryVersion)
		if version == "" {
			p.MinOsqueryVersion = nil
		} else {
			p.MinOsqueryVersion = &version
		}
	}
	p.Targets = normalizeReportTargets(p.Targets)
}

// ReportListParams filters reports.
type ReportListParams struct {
	ListParams listing.Params
}

// ReportSnapshotListParams filters and paginates report snapshot projections.
type ReportSnapshotListParams struct {
	ListParams listing.Params

	Status ReportSnapshotStatus
}

// ReportSnapshotStatus describes a targeted host's latest report observation.
type ReportSnapshotStatus string

const (
	ReportSnapshotStatusCollected ReportSnapshotStatus = "collected"
	ReportSnapshotStatusError     ReportSnapshotStatus = "error"
	ReportSnapshotStatusPending   ReportSnapshotStatus = "pending"
)

var ReportSnapshotStatusValues = []ReportSnapshotStatus{
	ReportSnapshotStatusCollected,
	ReportSnapshotStatusError,
	ReportSnapshotStatusPending,
}

// ReportSnapshot is the latest observation for one report and host.
// Rows contains every result unless a list query returns a matching subset.
// ReportedAt is nil until the host submits its first result or error.
type ReportSnapshot struct {
	ReportID          int64                `json:"report_id"`
	ReportName        string               `json:"report_name"`
	ReportDescription string               `json:"report_description,omitempty"`
	HostID            int64                `json:"host_id"`
	HostName          string               `json:"host_name"`
	Status            ReportSnapshotStatus `json:"status"`
	ResultRowCount    int32                `json:"result_row_count"`
	ReturnedRowCount  int32                `json:"returned_row_count"`
	Rows              []map[string]string  `json:"rows"`
	Error             string               `json:"error,omitempty"`
	ReportedAt        *time.Time           `json:"reported_at,omitempty"`
}

func (ReportSnapshotStatus) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(ReportSnapshotStatusValues...)
}
