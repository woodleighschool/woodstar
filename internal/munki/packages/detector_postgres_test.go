//go:build postgres

package packages

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestPackageDetectorIgnoresNoPkgAndReconcilesAllEligibleVersions(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := newDetectorPackageStore(db)
	softwareID := insertDetectorSoftware(t, db, "Reconciled Detector", nil)

	if _, err := store.Create(ctx, PackageCreateMutation{
		SoftwareID: softwareID,
		PackageMutation: PackageMutation{
			Version:       "0.9",
			InstallerType: InstallerTypeNoPkg,
			Installs:      detectorInstalls("com.example.nopkg", "/Applications/NoPkg.app"),
		},
	}); err != nil {
		t.Fatalf("create nopkg: %v", err)
	}
	assertStoredInstallationDetector(t, db, softwareID, nil, false)

	firstObjectID := insertDetectorInstallerObject(t, db, "first.pkg", "a")
	if _, err := store.Create(ctx, detectorPackageCreate(
		softwareID,
		"1.0",
		firstObjectID,
		detectorInstalls("com.example.reconciled", "/Applications/Reconciled.app"),
	)); err != nil {
		t.Fatalf("create first eligible package: %v", err)
	}
	assertStoredInstallationDetector(t, db, softwareID, &installationDetector{
		BundleIdentifier: "com.example.reconciled",
		ExpectedPath:     "/Applications/Reconciled.app",
		VersionSource:    installationVersionSourceBundleVersion,
	}, true)

	secondObjectID := insertDetectorInstallerObject(t, db, "second.pkg", "b")
	second, err := store.Create(ctx, detectorPackageCreate(
		softwareID,
		"2.0",
		secondObjectID,
		detectorInstalls("com.example.reconciled", "/Applications/Utilities/Reconciled.app"),
	))
	if err != nil {
		t.Fatalf("create second eligible package: %v", err)
	}
	assertStoredInstallationDetector(t, db, softwareID, &installationDetector{
		BundleIdentifier: "com.example.reconciled",
		VersionSource:    installationVersionSourceBundleVersion,
	}, true)

	if _, err := store.Update(ctx, second.ID, detectorPackageMutation(
		"2.0",
		secondObjectID,
		detectorInstalls("com.example.conflict", "/Applications/Conflict.app"),
	)); err != nil {
		t.Fatalf("make package metadata conflicting: %v", err)
	}
	assertStoredInstallationDetector(t, db, softwareID, nil, false)

	if _, err := store.Update(ctx, second.ID, detectorPackageMutation(
		"2.0",
		secondObjectID,
		detectorInstalls("com.example.reconciled", "/Applications/Utilities/Reconciled.app"),
	)); err != nil {
		t.Fatalf("restore consistent package metadata: %v", err)
	}
	assertStoredInstallationDetector(t, db, softwareID, &installationDetector{
		BundleIdentifier: "com.example.reconciled",
		VersionSource:    installationVersionSourceBundleVersion,
	}, true)
}

func TestPackageDetectorDoesNotOverwriteManualDetector(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := newDetectorPackageStore(db)
	manual := &installationDetector{
		BundleIdentifier: "com.example.manual",
		VersionSource:    installationVersionSourceBundleShortVersion,
	}
	softwareID := insertDetectorSoftware(t, db, "Manual Detector", manual)
	objectID := insertDetectorInstallerObject(t, db, "manual.pkg", "c")

	if _, err := store.Create(ctx, detectorPackageCreate(
		softwareID,
		"1.0",
		objectID,
		detectorInstalls("com.example.other", "/Applications/Other.app"),
	)); err != nil {
		t.Fatalf("create package for manual detector: %v", err)
	}
	assertStoredInstallationDetector(t, db, softwareID, manual, false)
}

func TestConcurrentPackageCreatesDoNotChooseFirstDetector(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := newDetectorPackageStore(db)
	softwareID := insertDetectorSoftware(t, db, "Concurrent Detector", nil)
	firstObjectID := insertDetectorInstallerObject(t, db, "concurrent-first.pkg", "d")
	secondObjectID := insertDetectorInstallerObject(t, db, "concurrent-second.pkg", "e")

	creates := []PackageCreateMutation{
		detectorPackageCreate(
			softwareID,
			"1.0",
			firstObjectID,
			detectorInstalls("com.example.concurrent-one", "/Applications/One.app"),
		),
		detectorPackageCreate(
			softwareID,
			"2.0",
			secondObjectID,
			detectorInstalls("com.example.concurrent-two", "/Applications/Two.app"),
		),
	}
	start := make(chan struct{})
	errs := make(chan error, len(creates))
	for _, create := range creates {
		go func(params PackageCreateMutation) {
			<-start
			_, err := store.Create(ctx, params)
			errs <- err
		}(create)
	}
	close(start)
	for range creates {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent create: %v", err)
		}
	}

	assertStoredInstallationDetector(t, db, softwareID, nil, false)
}

func TestPackageDeleteReconcilesAutomaticDetector(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := newDetectorPackageStore(db)
	softwareID := insertDetectorSoftware(t, db, "Delete Reconciliation", nil)
	firstObjectID := insertDetectorInstallerObject(t, db, "delete-first.pkg", "f")
	secondObjectID := insertDetectorInstallerObject(t, db, "delete-second.pkg", "0")

	first, err := store.Create(ctx, detectorPackageCreate(
		softwareID,
		"1.0",
		firstObjectID,
		detectorInstalls("com.example.delete", "/Applications/Delete.app"),
	))
	if err != nil {
		t.Fatalf("create first package: %v", err)
	}
	second, err := store.Create(ctx, detectorPackageCreate(
		softwareID,
		"2.0",
		secondObjectID,
		detectorInstalls("com.example.conflict", "/Applications/Conflict.app"),
	))
	if err != nil {
		t.Fatalf("create conflicting package: %v", err)
	}
	assertStoredInstallationDetector(t, db, softwareID, nil, false)

	if deleted, err := store.DeleteMany(ctx, []int64{second.ID}); err != nil || deleted != 1 {
		t.Fatalf("delete conflicting package = %d, %v, want one", deleted, err)
	}
	assertStoredInstallationDetector(t, db, softwareID, &installationDetector{
		BundleIdentifier: "com.example.delete",
		ExpectedPath:     "/Applications/Delete.app",
		VersionSource:    installationVersionSourceBundleVersion,
	}, true)

	if deleted, err := store.DeleteMany(ctx, []int64{first.ID}); err != nil || deleted != 1 {
		t.Fatalf("delete last eligible package = %d, %v, want one", deleted, err)
	}
	assertStoredInstallationDetector(t, db, softwareID, nil, false)
}

func TestPackageBulkDeleteSameSoftwareReconcilesOnceAndPreservesManualDetector(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := newDetectorPackageStore(db)
	automaticSoftwareID := insertDetectorSoftware(t, db, "Bulk Automatic Detector", nil)
	firstObjectID := insertDetectorInstallerObject(t, db, "bulk-first.pkg", "1")
	secondObjectID := insertDetectorInstallerObject(t, db, "bulk-second.pkg", "2")
	first, err := store.Create(ctx, detectorPackageCreate(
		automaticSoftwareID,
		"1.0",
		firstObjectID,
		detectorInstalls("com.example.bulk", "/Applications/Bulk.app"),
	))
	if err != nil {
		t.Fatalf("create first bulk package: %v", err)
	}
	second, err := store.Create(ctx, detectorPackageCreate(
		automaticSoftwareID,
		"2.0",
		secondObjectID,
		detectorInstalls("com.example.bulk", "/Applications/Bulk.app"),
	))
	if err != nil {
		t.Fatalf("create second bulk package: %v", err)
	}

	manual := &installationDetector{
		BundleIdentifier: "com.example.manual-delete",
		VersionSource:    installationVersionSourceBundleShortVersion,
	}
	manualSoftwareID := insertDetectorSoftware(t, db, "Manual Delete Detector", manual)
	manualObjectID := insertDetectorInstallerObject(t, db, "manual-delete.pkg", "3")
	manualPackage, err := store.Create(ctx, detectorPackageCreate(
		manualSoftwareID,
		"1.0",
		manualObjectID,
		detectorInstalls("com.example.other", "/Applications/Other.app"),
	))
	if err != nil {
		t.Fatalf("create manual-detector package: %v", err)
	}

	deleted, err := store.DeleteMany(ctx, []int64{first.ID, second.ID, manualPackage.ID})
	if err != nil || deleted != 3 {
		t.Fatalf("bulk delete packages = %d, %v, want three", deleted, err)
	}
	assertStoredInstallationDetector(t, db, automaticSoftwareID, nil, false)
	assertStoredInstallationDetector(t, db, manualSoftwareID, manual, false)
}

func newDetectorPackageStore(db *database.DB) *Store {
	objects := storage.NewObjectStore(db, nil, slog.New(slog.DiscardHandler))
	return NewStore(db, objects)
}

func insertDetectorSoftware(
	t *testing.T,
	db *database.DB,
	name string,
	detector *installationDetector,
) int64 {
	t.Helper()
	var bundleIdentifier, versionSource *string
	var expectedPath *string
	if detector != nil {
		bundleIdentifier = &detector.BundleIdentifier
		versionSource = &detector.VersionSource
		if detector.ExpectedPath != "" {
			expectedPath = &detector.ExpectedPath
		}
	}
	var softwareID int64
	if err := db.Pool().QueryRow(t.Context(), `
INSERT INTO munki_software (
    name,
    installation_detector_bundle_identifier,
    installation_detector_expected_path,
    installation_detector_version_source
) VALUES ($1, $2, $3, $4)
RETURNING id`, name, bundleIdentifier, expectedPath, versionSource).Scan(&softwareID); err != nil {
		t.Fatalf("insert software: %v", err)
	}
	return softwareID
}

func insertDetectorInstallerObject(t *testing.T, db *database.DB, filename, hashCharacter string) int64 {
	t.Helper()
	var objectID int64
	if err := db.Pool().QueryRow(t.Context(), `
INSERT INTO storage_objects (
    prefix, filename, content_type, size_bytes, sha256, available_at
) VALUES ('munki/packages', $1, 'application/octet-stream', 512, $2, now())
RETURNING id`, filename, strings.Repeat(hashCharacter, 64)).Scan(&objectID); err != nil {
		t.Fatalf("insert installer object: %v", err)
	}
	return objectID
}

func detectorPackageCreate(
	softwareID int64,
	version string,
	objectID int64,
	installs []PackageInstallItem,
) PackageCreateMutation {
	return PackageCreateMutation{
		SoftwareID:      softwareID,
		PackageMutation: detectorPackageMutation(version, objectID, installs),
	}
}

func detectorPackageMutation(
	version string,
	objectID int64,
	installs []PackageInstallItem,
) PackageMutation {
	return PackageMutation{
		Version:           version,
		InstallerType:     InstallerTypePkg,
		InstallerObjectID: &objectID,
		Installs:          installs,
	}
}

func detectorInstalls(bundleIdentifier, path string) []PackageInstallItem {
	return []PackageInstallItem{{
		Type:                 PackageInstallItemApplication,
		Path:                 path,
		BundleIdentifier:     bundleIdentifier,
		VersionComparisonKey: "CFBundleVersion",
	}}
}

func assertStoredInstallationDetector(
	t *testing.T,
	db *database.DB,
	softwareID int64,
	want *installationDetector,
	wantAutomatic bool,
) {
	t.Helper()
	got, automatic := loadStoredInstallationDetector(t.Context(), t, db, softwareID)
	if automatic != wantAutomatic {
		t.Fatalf("stored installation detector automatic = %t, want %t", automatic, wantAutomatic)
	}
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("stored installation detector = %#v, want %#v", got, want)
		}
		return
	}
	if *got != *want {
		t.Fatalf("stored installation detector = %#v, want %#v", got, want)
	}
}

func loadStoredInstallationDetector(
	ctx context.Context,
	t *testing.T,
	db *database.DB,
	softwareID int64,
) (*installationDetector, bool) {
	t.Helper()
	var bundleIdentifier, expectedPath, versionSource *string
	var automatic bool
	if err := db.Pool().QueryRow(ctx, `
SELECT
    installation_detector_bundle_identifier,
    installation_detector_expected_path,
    installation_detector_version_source,
    installation_detector_automatic
FROM munki_software
WHERE id = $1`, softwareID).Scan(
		&bundleIdentifier,
		&expectedPath,
		&versionSource,
		&automatic,
	); err != nil {
		t.Fatalf("read installation detector: %v", err)
	}
	if bundleIdentifier == nil || versionSource == nil {
		return nil, automatic
	}
	detector := &installationDetector{
		BundleIdentifier: *bundleIdentifier,
		VersionSource:    *versionSource,
	}
	if expectedPath != nil {
		detector.ExpectedPath = *expectedPath
	}
	return detector, automatic
}
