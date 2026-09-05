package software

import (
	"fmt"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/openapischema"
)

// SoftwareReportStatus is Munki's last evaluation of expected software.
type SoftwareReportStatus string

const (
	SoftwareReportStatusInstalled SoftwareReportStatus = "installed"
	SoftwareReportStatusPending   SoftwareReportStatus = "pending"
)

var SoftwareReportStatusValues = []SoftwareReportStatus{
	SoftwareReportStatusInstalled,
	SoftwareReportStatusPending,
}

// SoftwareReportHost is one host expected to install a software title.
type SoftwareReportHost struct {
	HostID         int64                 `json:"host_id"`
	HostName       string                `json:"host_name"`
	HardwareSerial string                `json:"hardware_serial"`
	Status         *SoftwareReportStatus `json:"status,omitempty"`
	TargetVersion  string                `json:"target_version,omitempty"`
	EvaluatedAt    *time.Time            `json:"evaluated_at,omitempty"`
}

// SoftwareReportHostListParams filters expected hosts for one software title.
type SoftwareReportHostListParams struct {
	ListParams listing.Params
	Statuses   []SoftwareReportStatus
}

func (SoftwareReportStatus) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(SoftwareReportStatusValues...)
}

func (params *SoftwareReportHostListParams) normalize() {
	params.ListParams = listing.Normalize(params.ListParams)
	params.Statuses = listing.NormalizeValues(params.Statuses)
}

func (params *SoftwareReportHostListParams) validate() error {
	if err := listing.Validate(params.ListParams); err != nil {
		return err
	}
	for _, status := range params.Statuses {
		if !slices.Contains(SoftwareReportStatusValues, status) {
			return fmt.Errorf("%w: invalid software report status %q", fault.ErrInvalidInput, status)
		}
	}
	return nil
}
