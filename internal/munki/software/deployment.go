package software

import (
	"sort"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/openapischema"
)

// InstallationStatus describes whether the configured application was present
// in the host's last successful software inventory snapshot.
type InstallationStatus string

const (
	StatusInstalled    InstallationStatus = "installed"
	StatusNotInstalled InstallationStatus = "not_installed"
	StatusUnknown      InstallationStatus = "unknown"
)

var installationStatusValues = []InstallationStatus{
	StatusInstalled,
	StatusNotInstalled,
	StatusUnknown,
}

// MunkiResult describes Munki's evaluation of its selected pkginfo.
type MunkiResult string

const (
	MunkiResultNoInstallNeeded  MunkiResult = "no_install_needed"
	MunkiResultInstallIndicated MunkiResult = "install_indicated"
	MunkiResultUnresolved       MunkiResult = "unresolved"
	MunkiResultNotReported      MunkiResult = "not_reported"
)

var munkiResultValues = []MunkiResult{
	MunkiResultNoInstallNeeded,
	MunkiResultInstallIndicated,
	MunkiResultUnresolved,
	MunkiResultNotReported,
}

// PackageDeployment is the desired and observed host count for one package version.
type PackageDeployment struct {
	Version        string `json:"version"`
	AssignedCount  int    `json:"assigned_count"`
	ObservedCount  int    `json:"observed_count"`
	InstalledCount int    `json:"installed_count"`
}

// DeploymentSummary is the desired and observed state for one software title.
type DeploymentSummary struct {
	AssignedCount  int                 `json:"assigned_count"`
	ObservedCount  int                 `json:"observed_count"`
	InstalledCount int                 `json:"installed_count"`
	Packages       []PackageDeployment `json:"packages" nullable:"false"`
}

// SoftwareWithDeployment adds current installation counts to software metadata.
type SoftwareWithDeployment struct {
	Software

	Deployment DeploymentSummary `json:"deployment"`
}

// DeploymentHost is one host's independently observed installation and Munki result.
type DeploymentHost struct {
	HostID           int64               `json:"host_id"`
	DisplayName      string              `json:"display_name"`
	HardwareSerial   string              `json:"hardware_serial"`
	Actions          []Action            `json:"actions" nullable:"false"`
	Package          HostManifestPackage `json:"package"`
	Status           InstallationStatus  `json:"status"`
	InstalledVersion string              `json:"installed_version,omitempty"`
	MunkiResult      MunkiResult         `json:"munki_result"`
	TargetVersion    string              `json:"target_version,omitempty"`
	LastCollectedAt  *time.Time          `json:"last_collected_at,omitempty"`
}

// DeploymentHostListParams filters the effective host state for one software title.
type DeploymentHostListParams struct {
	dbutil.ListParams

	Status      *InstallationStatus
	MunkiResult *MunkiResult
	Action      *Action
}

type munkiObservation struct {
	Installed     bool
	TargetVersion string
}

type deploymentFact struct {
	SoftwareID                   int64
	HostID                       int64
	DisplayName                  string
	HardwareSerial               string
	HasInstallationDetector      bool
	ObservedInstallationEligible bool
	SoftwareInventoryUpdatedAt   *time.Time
	ObservedApplication          bool
	ObservedVersions             []string
	MunkiLastSeenAt              *time.Time
	MunkiLastSuccessfulAt        *time.Time
	MunkiCollectionError         string
	MunkiHasReport               bool
	Actions                      []Action
	Package                      HostManifestPackage
	Observation                  *munkiObservation
}

// Schema returns the OpenAPI schema for InstallationStatus.
func (InstallationStatus) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(installationStatusValues...)
}

// Schema returns the OpenAPI schema for MunkiResult.
func (MunkiResult) Schema(_ huma.Registry) *huma.Schema {
	return openapischema.StringEnum(munkiResultValues...)
}

func installationStatus(fact deploymentFact) InstallationStatus {
	if !fact.HasInstallationDetector || !fact.ObservedInstallationEligible ||
		fact.SoftwareInventoryUpdatedAt == nil {
		return StatusUnknown
	}
	if fact.ObservedApplication {
		return StatusInstalled
	}
	return StatusNotInstalled
}

func installedVersion(fact deploymentFact) string {
	if installationStatus(fact) != StatusInstalled {
		return ""
	}
	versions := make(map[string]struct{}, len(fact.ObservedVersions))
	for _, version := range fact.ObservedVersions {
		if version = canonicalDeploymentVersion(version); version != "" {
			versions[version] = struct{}{}
		}
	}
	if len(versions) != 1 {
		return ""
	}
	for version := range versions {
		return version
	}
	return ""
}

func munkiResult(fact deploymentFact) MunkiResult {
	if !hasCurrentMunkiReport(fact) {
		return MunkiResultNotReported
	}
	if fact.Observation == nil {
		return MunkiResultUnresolved
	}
	if canonicalDeploymentVersion(fact.Observation.TargetVersion) != "" {
		return MunkiResultInstallIndicated
	}
	if fact.Observation.Installed {
		return MunkiResultNoInstallNeeded
	}
	return MunkiResultUnresolved
}

func hasCurrentMunkiReport(fact deploymentFact) bool {
	return fact.MunkiLastSeenAt != nil &&
		fact.MunkiLastSuccessfulAt != nil &&
		fact.MunkiCollectionError == "" &&
		fact.MunkiHasReport
}

func targetVersion(fact deploymentFact) string {
	if hasCurrentMunkiReport(fact) && fact.Observation != nil {
		if version := canonicalDeploymentVersion(fact.Observation.TargetVersion); version != "" {
			return version
		}
	}
	if fact.Package.Strategy == PackageSpecific {
		return canonicalDeploymentVersion(fact.Package.Version)
	}
	return ""
}

func deploymentSummary(facts []deploymentFact) DeploymentSummary {
	packages := make(map[string]*PackageDeployment)
	summary := DeploymentSummary{Packages: []PackageDeployment{}}

	for _, fact := range facts {
		summary.AssignedCount++
		status := installationStatus(fact)
		if status != StatusUnknown {
			summary.ObservedCount++
		}

		if version := targetVersion(fact); version != "" {
			deployment := packageDeployment(packages, version)
			deployment.AssignedCount++
			if status != StatusUnknown {
				deployment.ObservedCount++
			}
		}

		if status == StatusInstalled {
			summary.InstalledCount++
			if version := installedVersion(fact); version != "" {
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

func packageDeployment(packages map[string]*PackageDeployment, version string) *PackageDeployment {
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
