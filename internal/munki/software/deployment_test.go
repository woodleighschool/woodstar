package software

import (
	"reflect"
	"testing"
	"time"
)

func TestInstallationStatusUsesOnlyApplicationInventory(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		fact deploymentFact
		want InstallationStatus
	}{
		{name: "no detector", fact: deploymentFact{ObservedInstallationEligible: true, SoftwareInventoryUpdatedAt: &now}, want: StatusUnknown},
		{name: "ineligible package", fact: deploymentFact{HasInstallationDetector: true, SoftwareInventoryUpdatedAt: &now}, want: StatusUnknown},
		{
			name: "no successful app snapshot",
			fact: deploymentFact{HasInstallationDetector: true, ObservedInstallationEligible: true},
			want: StatusUnknown,
		},
		{
			name: "successful snapshot without match",
			fact: deploymentFact{
				HasInstallationDetector: true, ObservedInstallationEligible: true,
				SoftwareInventoryUpdatedAt: &now,
			},
			want: StatusNotInstalled,
		},
		{
			name: "successful snapshot with match",
			fact: deploymentFact{
				HasInstallationDetector: true, ObservedInstallationEligible: true,
				SoftwareInventoryUpdatedAt: &now, ObservedApplication: true,
			},
			want: StatusInstalled,
		},
		{
			name: "Munki installed does not override missing app",
			fact: deploymentFact{
				HasInstallationDetector:      true,
				ObservedInstallationEligible: true,
				SoftwareInventoryUpdatedAt:   &now,
				MunkiLastSeenAt:              &now,
				MunkiLastSuccessfulAt:        &now,
				MunkiHasReport:               true,
				Observation:                  &munkiObservation{Installed: true},
			},
			want: StatusNotInstalled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := installationStatus(tt.fact); got != tt.want {
				t.Fatalf("installationStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInstalledVersionRequiresOneObservedValue(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	fact := deploymentFact{
		HasInstallationDetector:      true,
		ObservedInstallationEligible: true,
		SoftwareInventoryUpdatedAt:   &now,
		ObservedApplication:          true,
	}

	for name, tt := range map[string]struct {
		versions []string
		want     string
	}{
		"one version":          {versions: []string{" 1.2 "}, want: "1.2"},
		"identical versions":   {versions: []string{"1.2", "1.2"}, want: "1.2"},
		"conflicting versions": {versions: []string{"1.2", "1.3"}},
		"only empty versions":  {versions: []string{"", " "}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			testFact := fact
			testFact.ObservedVersions = tt.versions
			if got := installedVersion(testFact); got != tt.want {
				t.Fatalf("installedVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMunkiResultIsIndependentFromObservedInstallation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	current := deploymentFact{
		MunkiLastSeenAt:       &now,
		MunkiLastSuccessfulAt: &now,
		MunkiHasReport:        true,
	}
	tests := []struct {
		name string
		fact deploymentFact
		want MunkiResult
	}{
		{name: "not contacted", want: MunkiResultNotReported},
		{
			name: "failed latest collection",
			fact: deploymentFact{
				MunkiLastSeenAt: &now, MunkiLastSuccessfulAt: &now,
				MunkiCollectionError: "munki_installs: unavailable", MunkiHasReport: true,
				Observation: &munkiObservation{Installed: true},
			},
			want: MunkiResultNotReported,
		},
		{name: "current omitted item", fact: current, want: MunkiResultUnresolved},
		{
			name: "install indicated",
			fact: withMunkiObservation(current, &munkiObservation{Installed: true, TargetVersion: "1.3"}),
			want: MunkiResultInstallIndicated,
		},
		{
			name: "no install needed",
			fact: withMunkiObservation(current, &munkiObservation{Installed: true}),
			want: MunkiResultNoInstallNeeded,
		},
		{
			name: "current unresolved item",
			fact: withMunkiObservation(current, &munkiObservation{}),
			want: MunkiResultUnresolved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := munkiResult(tt.fact); got != tt.want {
				t.Fatalf("munkiResult() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTargetVersionPrefersCurrentMunkiResultThenPin(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	packageID := int64(20)
	pinned := deploymentFact{
		Package: HostManifestPackage{Strategy: PackageSpecific, ID: &packageID, Version: " 1.3 "},
	}
	if got := targetVersion(pinned); got != "1.3" {
		t.Fatalf("pinned targetVersion() = %q, want 1.3", got)
	}
	pinned.MunkiLastSeenAt = &now
	pinned.MunkiLastSuccessfulAt = &now
	pinned.MunkiHasReport = true
	pinned.Observation = &munkiObservation{TargetVersion: " 1.4 "}
	if got := targetVersion(pinned); got != "1.4" {
		t.Fatalf("client targetVersion() = %q, want 1.4", got)
	}
	latest := deploymentFact{Package: HostManifestPackage{Strategy: PackageLatest}}
	if got := targetVersion(latest); got != "" {
		t.Fatalf("latest targetVersion() = %q, want empty", got)
	}
}

func TestDeploymentSummaryUsesObservedApplicationFacts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	packageID := int64(20)
	facts := []deploymentFact{
		{
			HasInstallationDetector: true, ObservedInstallationEligible: true,
			SoftwareInventoryUpdatedAt: &now,
			ObservedApplication:        true, ObservedVersions: []string{"1.2"},
			Package: HostManifestPackage{Strategy: PackageSpecific, ID: &packageID, Version: "1.3"},
			Actions: []Action{ActionManagedInstalls},
		},
		{
			HasInstallationDetector: true, ObservedInstallationEligible: true,
			SoftwareInventoryUpdatedAt: &now,
			Package:                    HostManifestPackage{Strategy: PackageSpecific, ID: &packageID, Version: "1.3"},
			Actions:                    []Action{ActionManagedUninstalls},
		},
		{
			Package: HostManifestPackage{Strategy: PackageLatest},
			Actions: []Action{ActionOptionalInstalls},
		},
	}

	got := deploymentSummary(facts)
	want := DeploymentSummary{
		AssignedCount:  3,
		ObservedCount:  2,
		InstalledCount: 1,
		Packages: []PackageDeployment{
			{Version: "1.2", InstalledCount: 1},
			{Version: "1.3", AssignedCount: 2, ObservedCount: 2},
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

func withMunkiObservation(fact deploymentFact, observation *munkiObservation) deploymentFact {
	fact.Observation = observation
	return fact
}
