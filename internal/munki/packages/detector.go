package packages

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const (
	installationVersionSourceBundleShortVersion = "bundle_short_version"
	installationVersionSourceBundleVersion      = "bundle_version"
)

type installationDetector struct {
	BundleIdentifier string
	ExpectedPath     string
	VersionSource    string
}

func installationDetectorFromInstalls(installs []PackageInstallItem) *installationDetector {
	detector, eligible := installationDetectorFromEligibleInstalls(installs)
	if !eligible {
		return nil
	}
	return detector
}

func installationDetectorFromEligibleInstalls(
	installs []PackageInstallItem,
) (*installationDetector, bool) {
	var detector *installationDetector
	eligible := false
	for _, item := range installs {
		if item.Type != PackageInstallItemApplication {
			continue
		}
		eligible = true
		candidate, ok := installationDetectorCandidate(item)
		if !ok {
			return nil, true
		}
		if detector == nil {
			detector = &candidate
			continue
		}
		if detector.BundleIdentifier != candidate.BundleIdentifier ||
			detector.VersionSource != candidate.VersionSource {
			return nil, true
		}
		if detector.ExpectedPath != candidate.ExpectedPath {
			detector.ExpectedPath = ""
		}
	}
	return detector, eligible
}

func installationDetectorCandidate(item PackageInstallItem) (installationDetector, bool) {
	bundleIdentifier := strings.TrimSpace(item.BundleIdentifier)
	if bundleIdentifier == "" {
		return installationDetector{}, false
	}
	versionSource := installationDetectorVersionSource(item)
	if versionSource == "" {
		return installationDetector{}, false
	}
	return installationDetector{
		BundleIdentifier: bundleIdentifier,
		ExpectedPath:     strings.TrimSpace(item.Path),
		VersionSource:    versionSource,
	}, true
}

func installationDetectorVersionSource(item PackageInstallItem) string {
	switch strings.TrimSpace(item.VersionComparisonKey) {
	case "CFBundleShortVersionString":
		return installationVersionSourceBundleShortVersion
	case "CFBundleVersion":
		return installationVersionSourceBundleVersion
	case "":
		shortVersion := strings.TrimSpace(item.BundleShortVersion)
		bundleVersion := strings.TrimSpace(item.BundleVersion)
		switch {
		case shortVersion != "" && bundleVersion == "":
			return installationVersionSourceBundleShortVersion
		case shortVersion == "" && bundleVersion != "":
			return installationVersionSourceBundleVersion
		}
	}
	return ""
}

func lockSoftwareForInstallationDetector(ctx context.Context, tx pgx.Tx, softwareID int64) error {
	var lockedID int64
	if err := tx.QueryRow(ctx, `
SELECT id
FROM munki_software
WHERE id = $1
FOR UPDATE`, softwareID).Scan(&lockedID); err != nil {
		return dbutil.GetError(err)
	}
	return nil
}

func reconcileInstallationDetector(ctx context.Context, tx pgx.Tx, softwareID int64) error {
	var bundleIdentifier, versionSource *string
	var automatic bool
	if err := tx.QueryRow(ctx, `
SELECT
    installation_detector_bundle_identifier,
    installation_detector_version_source,
    installation_detector_automatic
FROM munki_software
WHERE id = $1`, softwareID).Scan(&bundleIdentifier, &versionSource, &automatic); err != nil {
		return dbutil.GetError(err)
	}
	if !automatic && bundleIdentifier != nil && versionSource != nil {
		return nil
	}

	detector, err := softwareInstallationDetector(ctx, tx, softwareID)
	if err != nil {
		return err
	}
	if detector == nil {
		if !automatic {
			return nil
		}
		_, err := tx.Exec(ctx, `
UPDATE munki_software
SET
    installation_detector_bundle_identifier = NULL,
    installation_detector_expected_path = NULL,
    installation_detector_version_source = NULL,
    installation_detector_automatic = FALSE,
    updated_at = now()
WHERE id = $1`, softwareID)
		return err
	}

	var expectedPath *string
	if detector.ExpectedPath != "" {
		expectedPath = &detector.ExpectedPath
	}
	_, err = tx.Exec(ctx, `
UPDATE munki_software
SET
    installation_detector_bundle_identifier = $2,
    installation_detector_expected_path = $3,
    installation_detector_version_source = $4,
    installation_detector_automatic = TRUE,
    updated_at = now()
WHERE id = $1`,
		softwareID,
		detector.BundleIdentifier,
		expectedPath,
		detector.VersionSource,
	)
	return err
}

func softwareInstallationDetector(
	ctx context.Context,
	tx pgx.Tx,
	softwareID int64,
) (*installationDetector, error) {
	rows, err := tx.Query(ctx, `
SELECT installs
FROM munki_packages
WHERE software_id = $1
  AND installer_type <> 'nopkg'
ORDER BY id`, softwareID)
	if err != nil {
		return nil, err
	}
	packages, err := pgx.CollectRows(
		rows,
		pgx.RowTo[dbutil.JSONSlice[PackageInstallItem]],
	)
	if err != nil {
		return nil, err
	}

	var detector *installationDetector
	for _, installs := range packages {
		candidate, eligible := installationDetectorFromEligibleInstalls(installs)
		if !eligible {
			continue
		}
		if candidate == nil {
			return nil, nil
		}
		if detector == nil {
			detector = candidate
			continue
		}
		if detector.BundleIdentifier != candidate.BundleIdentifier ||
			detector.VersionSource != candidate.VersionSource {
			return nil, nil
		}
		if detector.ExpectedPath != candidate.ExpectedPath {
			detector.ExpectedPath = ""
		}
	}
	return detector, nil
}
