package software

import (
	"reflect"
	"testing"
	"time"
)

func TestClassifyDeployment(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	installed := &HostManifestSoftwareObservation{Installed: true, InstalledVersion: "2.0"}
	notInstalled := &HostManifestSoftwareObservation{Installed: false, TargetVersion: "2.0"}
	available := &HostManifestSoftwareObservation{}
	current := func(actions ...Action) deploymentFact {
		return deploymentFact{
			MunkiLastSeenAt:  &now,
			LastSuccessfulAt: &now,
			HasReport:        true,
			Actions:          actions,
		}
	}

	tests := []struct {
		name       string
		fact       deploymentFact
		wantStatus DeploymentStatus
	}{
		{
			name: "no Munki contact ignores retained observation",
			fact: deploymentFact{
				LastSuccessfulAt: &now,
				HasReport:        true,
				Actions:          []Action{ActionManagedInstalls},
				Observation:      installed,
			},
		},
		{
			name: "contact without complete report",
			fact: deploymentFact{
				MunkiLastSeenAt: &now,
				Actions:         []Action{ActionManagedInstalls},
			},
		},
		{
			name: "latest collection failed despite retained item",
			fact: deploymentFact{
				MunkiLastSeenAt:  &now,
				LastSuccessfulAt: &now,
				CollectionError:  "munki_installs: unavailable",
				HasReport:        true,
				Actions:          []Action{ActionManagedInstalls},
				Observation:      installed,
			},
		},
		{
			name: "complete collection without Munki report",
			fact: deploymentFact{
				MunkiLastSeenAt:  &now,
				LastSuccessfulAt: &now,
				Actions:          []Action{ActionManagedInstalls},
			},
		},
		{name: "required installed", fact: withObservation(current(ActionManagedInstalls), installed), wantStatus: StatusUpToDate},
		{name: "required reported not installed", fact: withObservation(current(ActionManagedInstalls), notInstalled), wantStatus: StatusPending},
		{name: "required without pending version", fact: withObservation(current(ActionManagedInstalls), available), wantStatus: StatusNotInstalled},
		{name: "required omitted", fact: current(ActionManagedInstalls), wantStatus: StatusNotInstalled},
		{name: "update installed", fact: withObservation(current(ActionManagedUpdates), installed), wantStatus: StatusUpToDate},
		{name: "optional installed", fact: withObservation(current(ActionOptionalInstalls), installed), wantStatus: StatusInstalled},
		{name: "optional pending", fact: withObservation(current(ActionOptionalInstalls), notInstalled), wantStatus: StatusPending},
		{name: "optional reported available", fact: withObservation(current(ActionOptionalInstalls), available), wantStatus: StatusAvailable},
		{name: "optional omitted", fact: current(ActionOptionalInstalls), wantStatus: StatusAvailable},
		{name: "uninstall has no inferred status", fact: withObservation(current(ActionManagedUninstalls), installed)},
		{
			name:       "optional presentation modifiers remain optional",
			fact:       current(ActionOptionalInstalls, ActionFeaturedItems, ActionDefaultInstalls),
			wantStatus: StatusAvailable,
		},
		{
			name:       "managed install has precedence",
			fact:       withObservation(current(ActionManagedUpdates, ActionManagedUninstalls, ActionManagedInstalls), installed),
			wantStatus: StatusUpToDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := deploymentStatusValue(deploymentStatus(tt.fact)); got != tt.wantStatus {
				t.Fatalf("deploymentStatus() = %q, want %q", got, tt.wantStatus)
			}
		})
	}
}

func TestDeploymentReportState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		fact deploymentFact
		want DeploymentReportState
	}{
		{name: "not contacted", want: ReportNotContacted},
		{
			name: "never collected",
			fact: deploymentFact{MunkiLastSeenAt: &now},
			want: ReportNeverCollected,
		},
		{
			name: "collection failed",
			fact: deploymentFact{
				MunkiLastSeenAt:  &now,
				LastSuccessfulAt: &now,
				CollectionError:  "munki_installs: unavailable",
				HasReport:        true,
			},
			want: ReportCollectionFailed,
		},
		{
			name: "no Munki report",
			fact: deploymentFact{MunkiLastSeenAt: &now, LastSuccessfulAt: &now},
			want: ReportNoReport,
		},
		{
			name: "current",
			fact: deploymentFact{
				MunkiLastSeenAt:  &now,
				LastSuccessfulAt: &now,
				HasReport:        true,
			},
			want: ReportCurrent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := deploymentReportState(tt.fact); got != tt.want {
				t.Fatalf("deploymentReportState() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeploymentSummaryCombinesPackageAssignmentsAndInstalls(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	packageID := int64(20)
	current := func(actions []Action, observation *HostManifestSoftwareObservation) deploymentFact {
		return deploymentFact{
			MunkiLastSeenAt:  &now,
			LastSuccessfulAt: &now,
			HasReport:        true,
			Actions:          actions,
			Package:          HostManifestPackage{Strategy: PackageLatest},
			Observation:      observation,
		}
	}
	facts := []deploymentFact{
		{
			Actions: []Action{ActionManagedInstalls},
			Package: HostManifestPackage{Strategy: PackageSpecific, ID: &packageID, Version: " 2.0 "},
		},
		{Actions: []Action{ActionManagedInstalls}, Package: HostManifestPackage{Strategy: PackageLatest}},
		current([]Action{ActionManagedInstalls}, &HostManifestSoftwareObservation{TargetVersion: "2.0"}),
		current([]Action{ActionManagedUpdates}, &HostManifestSoftwareObservation{
			Installed:        true,
			InstalledVersion: "1.0",
			TargetVersion:    "2.0",
		}),
		current([]Action{ActionOptionalInstalls}, &HostManifestSoftwareObservation{
			Installed:        true,
			InstalledVersion: "2.0",
		}),
		current([]Action{ActionManagedUninstalls}, &HostManifestSoftwareObservation{
			Installed:        true,
			InstalledVersion: "old",
		}),
	}

	got := deploymentSummary(facts)
	want := DeploymentSummary{
		AssignedCount:  5,
		ReportingCount: 3,
		InstalledCount: 2,
		Packages: []PackageDeployment{
			{Version: "1.0", InstalledCount: 1},
			{Version: "2.0", AssignedCount: 4, ReportingCount: 3, InstalledCount: 1},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("deploymentSummary() = %+v, want %+v", got, want)
	}
}

func TestCanonicalDeploymentVersion(t *testing.T) {
	t.Parallel()

	for input, want := range map[string]string{
		"":      "",
		" \t ":  "",
		" 2.0 ": "2.0",
	} {
		if got := canonicalDeploymentVersion(input); got != want {
			t.Fatalf("canonicalDeploymentVersion(%q) = %q, want %q", input, got, want)
		}
	}
}

func withObservation(fact deploymentFact, observation *HostManifestSoftwareObservation) deploymentFact {
	fact.Observation = observation
	return fact
}
