//go:build postgres

package software

import (
	"log/slog"
	"testing"

	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestSoftwareInstallationDetectorRoundTripsAndFullUpdateClearsIt(t *testing.T) {
	db, ctx := testdb.Open(t)
	objects := storage.NewObjectStore(db, nil, slog.New(slog.DiscardHandler))
	store := NewStore(db, objects, nil)

	created, err := store.Create(ctx, CreateMutation{
		Name: "Detector App",
		InstallationDetector: &InstallationDetector{
			BundleIdentifier: " com.example.detector ",
			ExpectedPath:     " /Applications/Detector.app ",
			VersionSource:    " bundle_version ",
		},
	})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	if detector := created.InstallationDetector; detector == nil ||
		detector.BundleIdentifier != "com.example.detector" ||
		detector.ExpectedPath != "/Applications/Detector.app" ||
		detector.VersionSource != InstallationVersionSourceBundleVersion {
		t.Fatalf("created installation detector = %#v, want normalized persisted detector", detector)
	}
	if _, err := db.Pool().Exec(ctx, `
UPDATE munki_software
SET installation_detector_automatic = TRUE
WHERE id = $1`, created.ID); err != nil {
		t.Fatalf("mark detector automatic: %v", err)
	}
	automatic, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("reload automatic detector: %v", err)
	}
	preserved, err := store.Update(ctx, created.ID, UpdateMutation{
		InstallationDetector: automatic.InstallationDetector,
	})
	if err != nil {
		t.Fatalf("preserve automatic detector with full update: %v", err)
	}
	if !preserved.InstallationDetectorAutomatic {
		t.Fatal("unchanged automatic detector became manual during full update")
	}

	updated, err := store.Update(ctx, created.ID, UpdateMutation{})
	if err != nil {
		t.Fatalf("clear detector with full update: %v", err)
	}
	if updated.InstallationDetector != nil {
		t.Fatalf("updated installation detector = %#v, want nil", updated.InstallationDetector)
	}
	if updated.InstallationDetectorAutomatic {
		t.Fatal("cleared installation detector remained automatic")
	}
}
