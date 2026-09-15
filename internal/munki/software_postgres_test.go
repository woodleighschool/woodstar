//go:build postgres

package munki_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/targeting"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestMunkiSoftwareIdentityIsUniqueAndSeparateFromDisplayName(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)

	software, err := stores.software.Create(ctx, munkisoftware.CreateMutation{
		Name:        "com.vendor.app",
		DisplayName: "Vendor App",
	})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	if software.Name != "com.vendor.app" || software.DisplayName == nil || *software.DisplayName != "Vendor App" {
		t.Fatalf(
			"software identity = %q/%v, want canonical and presentation names",
			software.Name,
			software.DisplayName,
		)
	}

	pkg, err := stores.packages.Create(ctx, packages.PackageCreateMutation{
		SoftwareID:    software.ID,
		Version:       "1.0",
		InstallerType: packages.InstallerTypeNoPkg,
		OnDemand:      true,
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	if pkg.Software.Name != "com.vendor.app" || pkg.Software.DisplayName == nil ||
		*pkg.Software.DisplayName != "Vendor App" {
		t.Fatalf(
			"package software identity = %q/%v, want canonical and presentation names",
			pkg.Software.Name,
			pkg.Software.DisplayName,
		)
	}
	packageRows, count, err := stores.packages.List(ctx, packages.PackageListParams{
		ListParams: listing.Params{Q: "Vendor App"},
	})
	if err != nil {
		t.Fatalf("search packages by visible software name: %v", err)
	}
	if count != 1 || len(packageRows) != 1 || packageRows[0].ID != pkg.ID {
		t.Fatalf("visible-name package search = %+v count %d, want package %d", packageRows, count, pkg.ID)
	}

	_, err = stores.software.Create(ctx, munkisoftware.CreateMutation{
		Name:        "com.vendor.app",
		DisplayName: "Duplicate Vendor App",
	})
	if !errors.Is(err, fault.ErrAlreadyExists) {
		t.Fatalf("duplicate canonical name error = %v, want already exists", err)
	}
}

func TestSoftwareTargetsRejectPinnedPackageFromAnotherSoftware(t *testing.T) {
	db, ctx := testdb.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(t, db)

	first, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "FirstAssignedApp"})
	if err != nil {
		t.Fatalf("create first software: %v", err)
	}
	second, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "SecondAssignedApp"})
	if err != nil {
		t.Fatalf("create second software: %v", err)
	}
	pkg := createMunkiPackage(t, ctx, stores, first.ID, "FirstAssignedApp", "1.0")
	_, err = stores.software.Update(ctx, second.ID, softwareTargetMutation(
		second,
		[]munkisoftware.Include{
			includeSpecificTarget(
				allHostsLabelID(t, ctx, labelStore),
				munkisoftware.ActionManagedInstalls,
				pkg.ID,
			),
		},
		nil,
	))
	if !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("Update software target error = %v, want invalid input", err)
	}
}

func TestSoftwareTargetsRejectBuiltinExclude(t *testing.T) {
	db, ctx := testdb.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(t, db)

	title, err := stores.software.Create(
		ctx,
		munkisoftware.CreateMutation{Name: "ExcludeOverlapApp"},
	)
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	excludeLabel, err := labelStore.Create(ctx, labels.LabelMutation{
		Name:                "Exclude Overlap Exclude",
		LabelMembershipType: labels.LabelMembershipTypeManual,
	})
	if err != nil {
		t.Fatalf("create exclude label: %v", err)
	}
	if _, err := stores.software.Update(ctx, title.ID, munkisoftware.UpdateMutation{
		Targets: munkisoftware.Targets{
			Exclude: labelRefs([]int64{allHostsLabelID(t, ctx, labelStore)}),
		},
	}); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("Update software builtin exclude error = %v, want ErrInvalidInput", err)
	}
	if _, err := stores.software.Update(ctx, title.ID, munkisoftware.UpdateMutation{
		Targets: munkisoftware.Targets{
			Exclude: labelRefs([]int64{excludeLabel.ID}),
		},
	}); err != nil {
		t.Fatalf("set exclude labels: %v", err)
	}
}

func TestDeleteMunkiSoftwareCleansPackagesTargetsAndIgnoresMissingBulkIDs(t *testing.T) {
	db, ctx := testdb.Open(t)
	labelStore := labels.NewStore(db)
	stores := newMunkiStores(t, db)
	labelID := allHostsLabelID(t, ctx, labelStore)

	firstIcon := createMunkiIconObject(t, ctx, stores, "DeletePinnedApp.png", "f")
	first, err := stores.software.Create(ctx, munkisoftware.CreateMutation{
		Name:         "DeletePinnedApp",
		IconObjectID: &firstIcon.ID,
	})
	if err != nil {
		t.Fatalf("create first software: %v", err)
	}
	firstPkg := createMunkiPackage(t, ctx, stores, first.ID, "DeletePinnedApp", "1.0")
	replaceTargets(t, ctx, stores, first, []munkisoftware.Include{
		includeSpecificTarget(labelID, munkisoftware.ActionManagedInstalls, firstPkg.ID),
	})

	if err := stores.software.Delete(ctx, first.ID); err != nil {
		t.Fatalf("delete first software: %v", err)
	}
	if _, err := stores.software.GetByID(ctx, first.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("GetByID after delete error = %v, want ErrNotFound", err)
	}
	assertObjectDeleted(t, ctx, stores.objects, firstIcon.ID)
	assertNoMunkiChildren(t, ctx, stores, first.ID)
	if err := stores.software.Delete(ctx, first.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("repeat delete error = %v, want ErrNotFound", err)
	}

	second, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "BulkPinnedApp"})
	if err != nil {
		t.Fatalf("create second software: %v", err)
	}
	secondPkg := createMunkiPackage(t, ctx, stores, second.ID, "BulkPinnedApp", "1.0")
	replaceTargets(t, ctx, stores, second, []munkisoftware.Include{
		includeSpecificTarget(labelID, munkisoftware.ActionManagedInstalls, secondPkg.ID),
	})
	third, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "BulkPlainApp"})
	if err != nil {
		t.Fatalf("create third software: %v", err)
	}

	deleted, err := stores.software.DeleteMany(ctx, []int64{second.ID, third.ID, third.ID + 999})
	if err != nil {
		t.Fatalf("bulk delete software: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("bulk deleted = %d, want 2", deleted)
	}
	assertNoMunkiChildren(t, ctx, stores, second.ID)
	if _, err := stores.software.GetByID(ctx, third.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("GetByID third after bulk delete error = %v, want ErrNotFound", err)
	}
}

func TestTargetMissingLabelFallsThroughToNotFound(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)
	title, err := stores.software.Create(
		ctx,
		munkisoftware.CreateMutation{Name: "MissingLabelTarget"},
	)
	if err != nil {
		t.Fatalf("create software: %v", err)
	}

	_, err = stores.software.Update(ctx, title.ID, softwareTargetMutation(
		title,
		[]munkisoftware.Include{
			includeTarget(999_999, munkisoftware.ActionManagedInstalls),
		},
		nil,
	))
	if !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("Update software target error = %v, want ErrNotFound", err)
	}
}

func TestSoftwarePatchPreservesTargetsAndReconcilesSharedIcon(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)
	labelStore := labels.NewStore(db)
	exclude, err := labelStore.Create(ctx, labels.LabelMutation{Name: "Patch exclude", LabelMembershipType: labels.LabelMembershipTypeManual})
	if err != nil {
		t.Fatal(err)
	}
	icon := createMunkiIconObject(t, ctx, stores, "shared.png", "a")
	targets := munkisoftware.Targets{
		Include: []munkisoftware.Include{includeTarget(allHostsLabelID(t, ctx, labelStore), munkisoftware.ActionManagedInstalls)},
		Exclude: []targeting.LabelRef{{LabelID: exclude.ID}},
	}
	current, err := stores.software.Create(ctx, munkisoftware.CreateMutation{
		Name: "PatchSoftware", DisplayName: "Visible title", Description: "Keep description", Category: "Keep category",
		IconObjectID: &icon.ID, Targets: targets,
	})
	if err != nil {
		t.Fatal(err)
	}
	other, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "SharesIcon", IconObjectID: &icon.ID})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := stores.software.Patch(ctx, current.ID, readPatch[munkisoftware.Patch](t, `{"developer":"  Publisher  "}`))
	if err != nil {
		t.Fatal(err)
	}
	actualTargets, err := stores.software.TargetsForSoftware(ctx, current.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := current.Mutation(targets)
	want.Developer = "Publisher"
	if !reflect.DeepEqual(updated.Mutation(actualTargets), want) {
		t.Fatalf("patch lost omitted metadata or targets: %+v", updated.Mutation(actualTargets))
	}
	updated, err = stores.software.Patch(ctx, current.ID, readPatch[munkisoftware.Patch](t,
		`{"display_name":null,"category":null,"icon_object_id":null,"targets":{"include":[]}}`))
	if err != nil {
		t.Fatal(err)
	}
	actualTargets, err = stores.software.TargetsForSoftware(ctx, current.ID)
	if err != nil || len(actualTargets.Include) != 0 || !reflect.DeepEqual(actualTargets.Exclude, targets.Exclude) ||
		updated.DisplayName != nil || updated.Category != "" || updated.IconObjectID != nil || updated.Description != "Keep description" {
		t.Fatalf("clear lost omitted metadata or targets: %+v, %+v, %v", updated, actualTargets, err)
	}
	if _, err := stores.objects.GetByID(ctx, icon.ID); err != nil {
		t.Fatalf("shared icon was removed: %v", err)
	}
	if _, err := stores.software.Patch(ctx, other.ID, readPatch[munkisoftware.Patch](t, `{"icon_object_id":null}`)); err != nil {
		t.Fatal(err)
	}
	assertObjectDeleted(t, ctx, stores.objects, icon.ID)
}

func TestSoftwarePatchRollsBackInvalidReferences(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)
	labelID := allHostsLabelID(t, ctx, labels.NewStore(db))
	icon := createMunkiIconObject(t, ctx, stores, "retain.png", "a")
	current, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "PatchTargetReferences", IconObjectID: &icon.ID})
	if err != nil {
		t.Fatal(err)
	}
	other, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "OtherTargetReferences"})
	if err != nil {
		t.Fatal(err)
	}
	otherPackage := createMunkiPackage(t, ctx, stores, other.ID, other.Name, "1.0")
	for _, data := range []string{
		`{"description":"Discard","icon_object_id":null,"targets":{"exclude":[{"label_id":999999}]}}`,
		fmt.Sprintf(`{"description":"Discard","targets":{"exclude":[{"label_id":%d}]}}`, labelID),
		fmt.Sprintf(`{"description":"Discard","targets":{"include":[{"label_id":%d,"package":{"strategy":"specific","package_id":%d},"actions":["managed_installs"]}]}}`, labelID, otherPackage.ID),
		`{"icon_object_id":999999}`,
	} {
		if _, err := stores.software.Patch(ctx, current.ID, readPatch[munkisoftware.Patch](t, data)); err == nil {
			t.Fatalf("invalid patch succeeded: %s", data)
		}
		actual, err := stores.software.GetByID(ctx, current.ID)
		if err != nil || !reflect.DeepEqual(actual.Mutation(munkisoftware.Targets{}), current.Mutation(munkisoftware.Targets{})) {
			t.Fatalf("failed patch changed metadata: %+v, %v", actual, err)
		}
		targets, err := stores.software.TargetsForSoftware(ctx, current.ID)
		if err != nil || len(targets.Include)+len(targets.Exclude) != 0 {
			t.Fatalf("failed patch changed targets: %+v, %v", targets, err)
		}
		if _, err := stores.objects.GetByID(ctx, icon.ID); err != nil {
			t.Fatalf("failed patch removed current icon: %v", err)
		}
	}
	if _, err := stores.software.Patch(ctx, 999999, munkisoftware.Patch{}); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("missing software error = %v", err)
	}
}

func TestSoftwarePatchPreservesConcurrentCommittedMutation(t *testing.T) {
	db, ctx := testdb.Open(t)
	stores := newMunkiStores(t, db)
	title, err := stores.software.Create(ctx, munkisoftware.CreateMutation{Name: "ConcurrentSoftwarePatch"})
	if err != nil {
		t.Fatal(err)
	}
	labelID := allHostsLabelID(t, ctx, labels.NewStore(db))
	document := readPatch[munkisoftware.Patch](t, `{"category":"Patched category"}`)
	runBlockedPatch(t, db, "munki_software", title.ID, func(ctx context.Context) error {
		_, err := stores.software.Patch(ctx, title.ID, document)
		return err
	}, func(ctx context.Context, tx pgx.Tx) {
		if _, err := tx.Exec(ctx, `UPDATE munki_software SET description = 'Concurrent description' WHERE id = $1`, title.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO munki_software_targets (software_id, direction, position, label_id, actions, package_selection) VALUES ($1, 'include', 0, $2, ARRAY['managed_installs']::munki_manifest_action[], 'latest')`, title.ID, labelID); err != nil {
			t.Fatal(err)
		}
	})
	actual, err := stores.software.GetByID(ctx, title.ID)
	if err != nil || actual.Description != "Concurrent description" || actual.Category != "Patched category" {
		t.Fatalf("concurrent software fields lost: %+v, %v", actual, err)
	}
	targets, err := stores.software.TargetsForSoftware(ctx, title.ID)
	if err != nil || len(targets.Include) != 1 {
		t.Fatalf("concurrent targets lost: %+v, %v", targets, err)
	}
}

func TestSoftwareWritesUseOneDatabaseConnection(t *testing.T) {
	db, ctx := testdb.Open(t)
	config := db.Config()
	config.MaxConns, config.MinConns = 1, 0
	single, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(single.Close)
	stores := newMunkiStores(t, single)
	icon := createMunkiIconObject(t, ctx, stores, "single.png", "a")
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	title, err := stores.software.Create(bounded, munkisoftware.CreateMutation{Name: "SingleConnection", IconObjectID: &icon.ID})
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := stores.packages.Create(bounded, packages.PackageCreateMutation{SoftwareID: title.ID, Version: "1.0", InstallerType: packages.InstallerTypeNoPkg})
	if err != nil {
		t.Fatal(err)
	}
	labelStore := labels.NewStore(single)
	label, err := labelStore.Create(bounded, labels.LabelMutation{Name: "SingleConnection", LabelMembershipType: labels.LabelMembershipTypeManual})
	if err != nil {
		t.Fatal(err)
	}
	mutation := title.Mutation(munkisoftware.Targets{Include: []munkisoftware.Include{{LabelID: label.ID, Package: munkisoftware.PackageSelector{Strategy: munkisoftware.PackageSpecific, PackageID: &pkg.ID}, Actions: []munkisoftware.Action{munkisoftware.ActionManagedInstalls}}}})
	if _, err := stores.software.Update(bounded, title.ID, mutation); err != nil {
		t.Fatal(err)
	}
	updated, err := stores.software.Patch(bounded, title.ID, readPatch[munkisoftware.Patch](t, `{"description":"Updated"}`))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Description != "Updated" || updated.IconObjectID == nil || *updated.IconObjectID != icon.ID {
		t.Fatalf("update lost owned fields: %+v", updated)
	}
	if err := stores.software.SetIcon(bounded, title.ID, icon.ID); err != nil {
		t.Fatal(err)
	}
}
