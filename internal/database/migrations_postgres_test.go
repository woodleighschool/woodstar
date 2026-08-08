//go:build postgres

package database_test

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestInstallationDetectorMigrationBackfillsOnlyUnambiguousApplications(t *testing.T) {
	databaseURL := testdb.Create(t, os.Getenv("WOODSTAR_TEST_DATABASE_URL"))
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	goose.SetBaseFS(os.DirFS("."))
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("set migration dialect: %v", err)
	}
	if err := goose.UpToContext(t.Context(), db, "migrations", 13); err != nil {
		t.Fatalf("migrate through version 13: %v", err)
	}

	if _, err := db.ExecContext(t.Context(), `
INSERT INTO munki_software (id, name) VALUES
    (101, 'Backfill Unique'),
    (102, 'Backfill Varying Paths'),
    (103, 'Backfill Ambiguous'),
    (104, 'Backfill NoPkg');

INSERT INTO storage_objects (
    id, prefix, filename, content_type, size_bytes, sha256, available_at
) VALUES
    (201, 'munki/packages', 'unique.pkg', 'application/octet-stream', 1, repeat('a', 64), now()),
    (202, 'munki/packages', 'varying-one.pkg', 'application/octet-stream', 1, repeat('b', 64), now()),
    (203, 'munki/packages', 'varying-two.pkg', 'application/octet-stream', 1, repeat('c', 64), now()),
    (204, 'munki/packages', 'ambiguous.pkg', 'application/octet-stream', 1, repeat('d', 64), now());

INSERT INTO munki_packages (
    software_id, version, installer_type, installer_object_id, installs
) VALUES
    (101, '1.0', 'pkg', 201, '[{
        "type": "application",
        "path": "/Applications/Unique.app",
        "bundle_identifier": " com.example.unique ",
        "bundle_short_version": "1.0"
    }]'::jsonb),
    (102, '1.0', 'pkg', 202, '[{
        "type": "application",
        "path": "/Applications/Varying.app",
        "bundle_identifier": "com.example.varying",
        "version_comparison_key": "CFBundleVersion"
    }]'::jsonb),
    (102, '2.0', 'pkg', 203, '[{
        "type": "application",
        "path": "/Applications/Utilities/Varying.app",
        "bundle_identifier": "com.example.varying",
        "version_comparison_key": "CFBundleVersion"
    }]'::jsonb),
    (103, '1.0', 'pkg', 204, '[
        {
            "type": "application",
            "path": "/Applications/One.app",
            "bundle_identifier": "com.example.one",
            "version_comparison_key": "CFBundleVersion"
        },
        {
            "type": "application",
            "path": "/Applications/Two.app",
            "bundle_identifier": "com.example.two",
            "version_comparison_key": "CFBundleVersion"
        }
    ]'::jsonb),
    (104, '1.0', 'nopkg', NULL, '[{
        "type": "application",
        "path": "/Applications/NoPkg.app",
        "bundle_identifier": "com.example.nopkg",
        "version_comparison_key": "CFBundleVersion"
    }]'::jsonb);`); err != nil {
		t.Fatalf("seed version 13 packages: %v", err)
	}

	if err := goose.UpContext(t.Context(), db, "migrations"); err != nil {
		t.Fatalf("apply detector migration: %v", err)
	}

	assertMigratedDetector(t, db, 101, "com.example.unique", "/Applications/Unique.app", "bundle_short_version", true)
	assertMigratedDetector(t, db, 102, "com.example.varying", "", "bundle_version", true)
	assertMigratedDetector(t, db, 103, "", "", "", false)
	assertMigratedDetector(t, db, 104, "", "", "", false)
}

func assertMigratedDetector(
	t *testing.T,
	db *sql.DB,
	softwareID int64,
	wantBundleIdentifier, wantExpectedPath, wantVersionSource string,
	wantAutomatic bool,
) {
	t.Helper()
	var bundleIdentifier, expectedPath, versionSource sql.NullString
	var automatic bool
	if err := db.QueryRowContext(t.Context(), `
SELECT
    installation_detector_bundle_identifier,
    installation_detector_expected_path,
    installation_detector_version_source,
    installation_detector_automatic
FROM munki_software
WHERE id = $1`, softwareID).Scan(&bundleIdentifier, &expectedPath, &versionSource, &automatic); err != nil {
		t.Fatalf("read migrated detector for software %d: %v", softwareID, err)
	}
	if bundleIdentifier.String != wantBundleIdentifier ||
		expectedPath.String != wantExpectedPath ||
		versionSource.String != wantVersionSource ||
		automatic != wantAutomatic {
		t.Fatalf(
			"migrated detector for software %d = %q/%q/%q automatic=%t, want %q/%q/%q automatic=%t",
			softwareID,
			bundleIdentifier.String,
			expectedPath.String,
			versionSource.String,
			automatic,
			wantBundleIdentifier,
			wantExpectedPath,
			wantVersionSource,
			wantAutomatic,
		)
	}
}
