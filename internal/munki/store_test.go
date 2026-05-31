package munki_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/munki"
)

func TestDesiredStateCreateListAndResolveForHost(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := munki.NewStore(db)

	includedHost, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-desired-included-uuid", Serial: "C02MUNKIIN"},
		OrbitNodeKey: "munki-desired-included-orbit",
	})
	if err != nil {
		t.Fatalf("enroll included host: %v", err)
	}
	excludedHost, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-desired-excluded-uuid", Serial: "C02MUNKIOUT"},
		OrbitNodeKey: "munki-desired-excluded-orbit",
	})
	if err != nil {
		t.Fatalf("enroll excluded host: %v", err)
	}
	label, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Munki Desired Test",
		LabelMembershipType: labels.LabelMembershipTypeManual,
		HostIDs:             []int64{includedHost.ID, excludedHost.ID},
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	title, err := store.CreateSoftwareTitle(ctx, munki.SoftwareTitleMutation{
		Name:        "GoogleChrome",
		DisplayName: "Google Chrome",
		Category:    "Browsers",
		Developer:   "Google",
	})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	release, err := store.CreateRelease(ctx, munki.ReleaseMutation{
		SoftwareID: title.ID,
		Name:       "GoogleChrome",
		Version:    "148.0.0.1",
		Pkginfo:    json.RawMessage(`{"name":"GoogleChrome","version":"148.0.0.1","installer_type":"nopkg"}`),
		Eligible:   true,
	})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	assignment, err := store.CreateAssignment(ctx, munki.AssignmentMutation{
		ReleaseID:       release.ID,
		Intent:          munki.IntentEnsureInstalled,
		IncludeLabelIDs: []int64{label.ID},
		ExcludeHostIDs:  []int64{excludedHost.ID},
	})
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	titles, titleCount, err := store.ListSoftwareTitles(ctx, dbutil.ListParams{})
	if err != nil {
		t.Fatalf("list software titles: %v", err)
	}
	if titleCount != 1 || len(titles) != 1 || titles[0].Name != "GoogleChrome" {
		t.Fatalf("titles = %+v count = %d, want GoogleChrome", titles, titleCount)
	}
	releases, releaseCount, err := store.ListReleases(ctx, dbutil.ListParams{})
	if err != nil {
		t.Fatalf("list releases: %v", err)
	}
	if releaseCount != 1 || len(releases) != 1 || releases[0].Version != "148.0.0.1" {
		t.Fatalf("releases = %+v count = %d, want version 148.0.0.1", releases, releaseCount)
	}
	assignments, assignmentCount, err := store.ListAssignments(ctx, dbutil.ListParams{})
	if err != nil {
		t.Fatalf("list assignments: %v", err)
	}
	if assignmentCount != 1 || len(assignments) != 1 || assignments[0].ID != assignment.ID {
		t.Fatalf("assignments = %+v count = %d, want created assignment", assignments, assignmentCount)
	}
	if !sameInt64s(assignments[0].IncludeLabelIDs, []int64{label.ID}) {
		t.Fatalf("include label ids = %v, want %v", assignments[0].IncludeLabelIDs, []int64{label.ID})
	}
	if !sameInt64s(assignments[0].ExcludeHostIDs, []int64{excludedHost.ID}) {
		t.Fatalf("exclude host ids = %v, want %v", assignments[0].ExcludeHostIDs, []int64{excludedHost.ID})
	}

	included, err := store.EffectiveReleasesForHost(ctx, includedHost.ID)
	if err != nil {
		t.Fatalf("resolve included host: %v", err)
	}
	if len(included) != 1 || included[0].Release.Name != "GoogleChrome" ||
		included[0].Intent != munki.IntentEnsureInstalled {
		t.Fatalf("included effective releases = %+v, want GoogleChrome install", included)
	}
	excluded, err := store.EffectiveReleasesForHost(ctx, excludedHost.ID)
	if err != nil {
		t.Fatalf("resolve excluded host: %v", err)
	}
	if len(excluded) != 0 {
		t.Fatalf("excluded effective releases = %+v, want none", excluded)
	}
}

func TestEffectiveReleasesForHostResolvesOverlappingAssignments(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	labelStore := labels.NewStore(db)
	store := munki.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-overlap-uuid", Serial: "C02MUNKIOL"},
		OrbitNodeKey: "munki-overlap-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}
	label, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Munki Overlap Test",
		LabelMembershipType: labels.LabelMembershipTypeManual,
		HostIDs:             []int64{host.ID},
	})
	if err != nil {
		t.Fatalf("create label: %v", err)
	}
	title, err := store.CreateSoftwareTitle(ctx, munki.SoftwareTitleMutation{Name: "OverlapApp"})
	if err != nil {
		t.Fatalf("create software title: %v", err)
	}
	optionalRelease := createMunkiRelease(t, ctx, store, title.ID, "OverlapApp", "1.0")
	installRelease := createMunkiRelease(t, ctx, store, title.ID, "OverlapApp", "2.0")
	absentRelease := createMunkiRelease(t, ctx, store, title.ID, "OverlapApp", "3.0")

	if _, err := store.CreateAssignment(ctx, munki.AssignmentMutation{
		ReleaseID: optionalRelease.ID,
		Intent:    munki.IntentOptional,
		AllHosts:  true,
	}); err != nil {
		t.Fatalf("create all-host optional assignment: %v", err)
	}
	if _, err := store.CreateAssignment(ctx, munki.AssignmentMutation{
		ReleaseID:       installRelease.ID,
		Intent:          munki.IntentEnsureInstalled,
		IncludeLabelIDs: []int64{label.ID},
	}); err != nil {
		t.Fatalf("create label install assignment: %v", err)
	}
	if _, err := store.CreateAssignment(ctx, munki.AssignmentMutation{
		ReleaseID:      absentRelease.ID,
		Intent:         munki.IntentEnsureAbsent,
		IncludeHostIDs: []int64{host.ID},
	}); err != nil {
		t.Fatalf("create host removal assignment: %v", err)
	}

	effective, err := store.EffectiveReleasesForHost(ctx, host.ID)
	if err != nil {
		t.Fatalf("resolve effective releases: %v", err)
	}
	if len(effective) != 1 {
		t.Fatalf("effective releases = %+v, want one resolved item", effective)
	}
	if effective[0].Intent != munki.IntentEnsureAbsent || effective[0].Release.Version != "3.0" {
		t.Fatalf("effective release = %+v, want removal of OverlapApp 3.0", effective[0])
	}
}

func TestCreateReleaseRejectsInvalidPkginfo(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := munki.NewStore(db)

	cases := []struct {
		name    string
		pkginfo json.RawMessage
	}{
		{name: "array", pkginfo: json.RawMessage(`[]`)},
		{name: "missing version", pkginfo: json.RawMessage(`{"name":"Broken"}`)},
		{name: "version mismatch", pkginfo: json.RawMessage(`{"name":"Broken","version":"2.0"}`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := store.CreateRelease(ctx, munki.ReleaseMutation{
				SoftwareID: 1,
				Name:       "Broken",
				Version:    "1.0",
				Pkginfo:    tc.pkginfo,
				Eligible:   true,
			})
			if !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("CreateRelease error = %v, want invalid input", err)
			}
		})
	}
}

func TestCreateAssignmentRejectsEmptyScope(t *testing.T) {
	db, ctx := dbtest.Open(t)
	store := munki.NewStore(db)

	_, err := store.CreateAssignment(ctx, munki.AssignmentMutation{
		ReleaseID: 1,
		Intent:    munki.IntentEnsureInstalled,
	})
	if !errors.Is(err, dbutil.ErrInvalidInput) {
		t.Fatalf("CreateAssignment error = %v, want invalid input", err)
	}
}

func TestHostStatusUpsertAndDetail(t *testing.T) {
	db, ctx := dbtest.Open(t)
	hostStore := hosts.NewStore(db)
	store := munki.NewStore(db)

	host, err := hostStore.UpsertOnOrbitEnroll(ctx, hosts.InventoryUpdate{
		Hardware:     hosts.HostHardware{UUID: "munki-host-observation-uuid", Serial: "C02MUNKI"},
		OrbitNodeKey: "munki-host-observation-orbit",
	})
	if err != nil {
		t.Fatalf("enroll host: %v", err)
	}

	if detail, err := store.LoadHostState(ctx, host.ID); err != nil {
		t.Fatalf("load absent munki detail: %v", err)
	} else if detail != nil {
		t.Fatalf("absent munki detail = %+v, want nil", detail)
	}

	success := true
	if err := store.UpsertHostStatus(ctx, munki.HostStatusObservation{
		HostID:          host.ID,
		Version:         "7.1.2.5700",
		ManifestName:    "site_default",
		Success:         &success,
		Errors:          []string{"first error"},
		Warnings:        []string{"first warning"},
		ProblemInstalls: []string{"Broken App"},
		RunStartedAt:    "2026-05-31 19:23:00 +1000",
		RunEndedAt:      "2026-05-31 19:24:14 +1000",
	}); err != nil {
		t.Fatalf("upsert munki host status: %v", err)
	}
	if err := store.ReplaceHostItems(ctx, host.ID, []munki.HostItem{
		{Name: "GoogleChrome", Installed: true, InstalledVersion: "148.0", RunEndedAt: "2026-05-31 19:24:14 +1000"},
		{Name: "Optional App", Installed: false},
	}); err != nil {
		t.Fatalf("replace munki host items: %v", err)
	}

	detail, err := store.LoadHostState(ctx, host.ID)
	if err != nil {
		t.Fatalf("load munki detail: %v", err)
	}
	if detail == nil {
		t.Fatal("munki detail is nil")
	}
	if detail.Version != "7.1.2.5700" || detail.ManifestName != "site_default" {
		t.Fatalf("detail = %+v, want version and manifest", detail)
	}
	if detail.Success == nil || !*detail.Success {
		t.Fatalf("success = %v, want true", detail.Success)
	}
	if len(detail.Items) != 2 || detail.Items[0].Name != "GoogleChrome" || !detail.Items[0].Installed {
		t.Fatalf("items = %+v", detail.Items)
	}

	if err := store.ReplaceHostItems(
		ctx,
		host.ID,
		[]munki.HostItem{{Name: "Replacement", Installed: true}},
	); err != nil {
		t.Fatalf("replace munki host items again: %v", err)
	}
	detail, err = store.LoadHostState(ctx, host.ID)
	if err != nil {
		t.Fatalf("load munki detail after replace: %v", err)
	}
	if len(detail.Items) != 1 || detail.Items[0].Name != "Replacement" {
		t.Fatalf("items after replace = %+v", detail.Items)
	}

	if err := store.ClearHostStatus(ctx, host.ID); err != nil {
		t.Fatalf("clear munki host status: %v", err)
	}
	if detail, err := store.LoadHostState(ctx, host.ID); err != nil {
		t.Fatalf("load cleared munki detail: %v", err)
	} else if detail != nil {
		t.Fatalf("cleared munki detail = %+v, want nil", detail)
	}
}

func sameInt64s(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func createMunkiRelease(
	t *testing.T,
	ctx context.Context,
	store *munki.Store,
	softwareID int64,
	name string,
	version string,
) munki.Release {
	t.Helper()
	pkginfo := json.RawMessage(fmt.Sprintf(
		`{"name":%q,"version":%q,"installer_type":"nopkg"}`,
		name,
		version,
	))
	release, err := store.CreateRelease(ctx, munki.ReleaseMutation{
		SoftwareID: softwareID,
		Name:       name,
		Version:    version,
		Pkginfo:    pkginfo,
		Eligible:   true,
	})
	if err != nil {
		t.Fatalf("create release %s %s: %v", name, version, err)
	}
	return *release
}
