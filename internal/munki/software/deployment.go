package software

import (
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/openapischema"
)

// DeploymentStatus describes the current Munki state of one desired software assignment.
type DeploymentStatus string

const (
	StatusUpToDate     DeploymentStatus = "up_to_date"
	StatusPending      DeploymentStatus = "pending"
	StatusNotInstalled DeploymentStatus = "not_installed"
	StatusInstalled    DeploymentStatus = "installed"
	StatusAvailable    DeploymentStatus = "available"
)

var deploymentStatusValues = []DeploymentStatus{
	StatusUpToDate,
	StatusPending,
	StatusNotInstalled,
	StatusInstalled,
	StatusAvailable,
}

// DeploymentReportState describes whether current Munki evidence is available for a host.
type DeploymentReportState string

const (
	ReportCurrent          DeploymentReportState = "current"
	ReportNotContacted     DeploymentReportState = "not_contacted"
	ReportNeverCollected   DeploymentReportState = "never_collected"
	ReportNoReport         DeploymentReportState = "no_report"
	ReportCollectionFailed DeploymentReportState = "collection_failed"
)

var deploymentReportStateValues = []DeploymentReportState{
	ReportCurrent,
	ReportNotContacted,
	ReportNeverCollected,
	ReportNoReport,
	ReportCollectionFailed,
}

// PackageDeployment is the desired and observed host count for one package version.
type PackageDeployment struct {
	Version        string `json:"version"`
	AssignedCount  int    `json:"assigned_count"`
	ReportingCount int    `json:"reporting_count"`
	InstalledCount int    `json:"installed_count"`
}

// DeploymentSummary is the current desired and observed state for one software title.
type DeploymentSummary struct {
	AssignedCount  int                 `json:"assigned_count"`
	ReportingCount int                 `json:"reporting_count"`
	InstalledCount int                 `json:"installed_count"`
	Packages       []PackageDeployment `json:"packages" nullable:"false"`
}

// SoftwareWithDeployment adds current installation counts to software metadata.
type SoftwareWithDeployment struct {
	Software

	Deployment DeploymentSummary `json:"deployment"`
}

// DeploymentHost is one host's desired package and current Munki observation.
type DeploymentHost struct {
	HostID           int64                 `json:"host_id"`
	DisplayName      string                `json:"display_name"`
	HardwareSerial   string                `json:"hardware_serial"`
	Actions          []Action              `json:"actions" nullable:"false"`
	Package          HostManifestPackage   `json:"package"`
	ReportState      DeploymentReportState `json:"report_state"`
	Status           *DeploymentStatus     `json:"status,omitempty"`
	Installed        bool                  `json:"installed"`
	InstalledVersion string                `json:"installed_version,omitempty"`
	TargetVersion    string                `json:"target_version,omitempty"`
	LastAttemptAt    *time.Time            `json:"last_attempt_at,omitempty"`
	LastSuccessfulAt *time.Time            `json:"last_successful_at,omitempty"`
	CollectionError  string                `json:"collection_error,omitempty"`
}

// DeploymentHostListParams filters the effective host state for one software title.
type DeploymentHostListParams struct {
	dbutil.ListParams

	Status *DeploymentStatus
	Action *Action
}

type deploymentFact struct {
	SoftwareID       int64
	HostID           int64
	DisplayName      string
	HardwareSerial   string
	MunkiLastSeenAt  *time.Time
	LastAttemptAt    *time.Time
	LastSuccessfulAt *time.Time
	CollectionError  string
	HasReport        bool
	Actions          []Action
	Package          HostManifestPackage
	Observation      *HostManifestSoftwareObservation
}

// Schema returns the OpenAPI schema for DeploymentStatus.
func (DeploymentStatus) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(deploymentStatusValues...)
}

// Schema returns the OpenAPI schema for DeploymentReportState.
func (DeploymentReportState) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(deploymentReportStateValues...)
}

func deploymentStatus(fact deploymentFact) *DeploymentStatus {
	if !hasCurrentDeploymentReport(fact) {
		return nil
	}

	var status DeploymentStatus
	switch primaryDeploymentAction(fact.Actions) {
	case ActionManagedInstalls, ActionManagedUpdates:
		switch {
		case fact.Observation == nil:
			status = StatusNotInstalled
		case fact.Observation.TargetVersion != "":
			status = StatusPending
		case fact.Observation.Installed:
			status = StatusUpToDate
		default:
			status = StatusNotInstalled
		}
	case ActionManagedUninstalls:
		return nil
	case ActionOptionalInstalls, ActionFeaturedItems, ActionDefaultInstalls:
		switch {
		case fact.Observation == nil:
			status = StatusAvailable
		case fact.Observation.TargetVersion != "":
			status = StatusPending
		case fact.Observation.Installed:
			status = StatusInstalled
		default:
			status = StatusAvailable
		}
	}
	return &status
}

func hasCurrentDeploymentReport(fact deploymentFact) bool {
	return deploymentReportState(fact) == ReportCurrent
}

func deploymentReportState(fact deploymentFact) DeploymentReportState {
	if fact.MunkiLastSeenAt == nil {
		return ReportNotContacted
	}
	if fact.LastSuccessfulAt == nil {
		return ReportNeverCollected
	}
	if fact.CollectionError != "" {
		return ReportCollectionFailed
	}
	if !fact.HasReport {
		return ReportNoReport
	}
	return ReportCurrent
}

func primaryDeploymentAction(actions []Action) Action {
	if slices.Contains(actions, ActionManagedInstalls) {
		return ActionManagedInstalls
	}
	if slices.Contains(actions, ActionManagedUninstalls) {
		return ActionManagedUninstalls
	}
	if slices.Contains(actions, ActionManagedUpdates) {
		return ActionManagedUpdates
	}
	return ActionOptionalInstalls
}

func deploymentSummary(facts []deploymentFact) DeploymentSummary {
	packages := make(map[string]*PackageDeployment)
	summary := DeploymentSummary{Packages: []PackageDeployment{}}

	for _, fact := range facts {
		if primaryDeploymentAction(fact.Actions) == ActionManagedUninstalls {
			continue
		}

		summary.AssignedCount++
		current := hasCurrentDeploymentReport(fact)
		if current {
			summary.ReportingCount++
		}

		if version := assignedPackageVersion(fact); version != "" {
			deployment := packageDeployment(packages, version)
			deployment.AssignedCount++
			if current {
				deployment.ReportingCount++
			}
		}

		if current && fact.Observation != nil && fact.Observation.Installed {
			summary.InstalledCount++
			if version := canonicalDeploymentVersion(fact.Observation.InstalledVersion); version != "" {
				packageDeployment(packages, version).InstalledCount++
			}
		}
	}

	for _, deployment := range packages {
		summary.Packages = append(summary.Packages, *deployment)
	}
	sort.Slice(summary.Packages, func(i, j int) bool {
		return summary.Packages[i].Version < summary.Packages[j].Version
	})
	return summary
}

func assignedPackageVersion(fact deploymentFact) string {
	if fact.Package.Strategy == PackageSpecific {
		return canonicalDeploymentVersion(fact.Package.Version)
	}
	if fact.Observation == nil {
		return ""
	}
	if version := canonicalDeploymentVersion(fact.Observation.TargetVersion); version != "" {
		return version
	}
	if fact.Observation.Installed {
		return canonicalDeploymentVersion(fact.Observation.InstalledVersion)
	}
	return ""
}

func packageDeployment(
	packages map[string]*PackageDeployment,
	version string,
) *PackageDeployment {
	deployment, ok := packages[version]
	if !ok {
		deployment = &PackageDeployment{Version: version}
		packages[version] = deployment
	}
	return deployment
}

func canonicalDeploymentVersion(version string) string {
	return strings.TrimSpace(version)
}
