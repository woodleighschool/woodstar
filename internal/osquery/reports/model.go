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
