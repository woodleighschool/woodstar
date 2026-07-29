// Package reports manages scheduled osquery reports and their result snapshots.
package reports

import (
	"fmt"
	"strings"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/validation"
)

// Report is a saved osquery snapshot query.
type Report struct {
	ID                int64         `json:"id"`
	Name              string        `json:"name"`
	Description       string        `json:"description"`
	Query             string        `json:"query"`
	MinOsqueryVersion *string       `json:"min_osquery_version,omitempty"`
	ScheduleInterval  int32         `json:"schedule_interval"`
	Targets           ReportTargets `json:"targets"`
	CreatedByUserID   *int64        `json:"created_by_user_id,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// ReportMutation is the editable report state used by create and update.
type ReportMutation struct {
	Name              string        `json:"name"                          validate:"required,notblank" minLength:"1"`
	Description       string        `json:"description,omitempty"`
	Query             string        `json:"query"                         validate:"required,notblank" minLength:"1"`
	MinOsqueryVersion *string       `json:"min_osquery_version,omitempty"`
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
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
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
	dbutil.ListParams
}

// ReportSnapshot is the latest complete observation for one report and host.
// CollectedAt is nil until the host submits its first snapshot.
type ReportSnapshot struct {
	ReportID          int64               `json:"report_id"`
	ReportName        string              `json:"report_name"`
	ReportDescription string              `json:"report_description,omitempty"`
	HostID            int64               `json:"host_id"`
	HostName          string              `json:"host_name"`
	Rows              []map[string]string `json:"rows"`
	CollectedAt       *time.Time          `json:"collected_at,omitempty"`
}
