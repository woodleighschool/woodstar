package software

import (
	"context"
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

type hostSoftwareRow struct {
	SoftwareID                   int64      `db:"software_id"`
	SoftwareName                 string     `db:"software_name"`
	SoftwareDisplayName          *string    `db:"software_display_name"`
	SoftwareDescription          string     `db:"software_description"`
	SoftwareCategory             string     `db:"software_category"`
	SoftwareDeveloper            string     `db:"software_developer"`
	SoftwareIconObjectID         *int64     `db:"software_icon_object_id"`
	HasInstallationDetector      bool       `db:"has_installation_detector"`
	ObservedInstallationEligible bool       `db:"observed_installation_eligible"`
	SoftwareInventoryUpdatedAt   *time.Time `db:"software_inventory_updated_at"`
	ObservedApplication          bool       `db:"observed_application"`
	ObservedVersions             []string   `db:"observed_versions"`
	Actions                      []string   `db:"actions"`
	PackageSelection             string     `db:"package_selection"`
	PinnedPackageID              *int64     `db:"pinned_package_id"`
	PackageVersion               *string    `db:"package_version"`
	ObservationName              *string    `db:"observation_name"`
	MunkiInstalled               *bool      `db:"munki_installed"`
	MunkiTargetVersion           *string    `db:"munki_target_version"`
	MunkiLastSeenAt              *time.Time `db:"munki_last_seen_at"`
	MunkiLastSuccessfulAt        *time.Time `db:"munki_last_successful_at"`
	MunkiCollectionError         string     `db:"munki_collection_error"`
	MunkiHasReport               bool       `db:"munki_has_report"`
}

// ListForHost returns the exact software enumeration used to render a host manifest.
func (s *Store) ListForHost(
	ctx context.Context,
	hostID int64,
	params HostManifestSoftwareListParams,
) ([]HostManifestSoftware, int, error) {
	params.ListParams = dbutil.NormalizeListParams(params.ListParams)
	whereSQL := "WHERE h.id = $1"
	args := []any{hostID}
	if params.Q != "" {
		whereSQL += " AND software.name ILIKE $2"
		args = append(args, "%"+params.Q+"%")
	}
	query := dbutil.ListQuery{
		SelectSQL: `
SELECT
	resolved.software_id,
	software.name AS software_name,
	software.display_name AS software_display_name,
	software.description AS software_description,
	software.category AS software_category,
	software.developer AS software_developer,
	software.icon_object_id AS software_icon_object_id,
	software.installation_detector_bundle_identifier IS NOT NULL AS has_installation_detector,
	CASE resolved.package_selection
		WHEN 'specific' THEN COALESCE(pinned.installer_type <> 'nopkg', FALSE)
		WHEN 'latest' THEN
			EXISTS (
				SELECT 1 FROM munki_packages candidate
				WHERE candidate.software_id = software.id
			)
			AND NOT EXISTS (
				SELECT 1 FROM munki_packages candidate
				WHERE candidate.software_id = software.id
					AND candidate.installer_type = 'nopkg'
			)
		ELSE FALSE
	END AS observed_installation_eligible,
	h.software_inventory_updated_at,
	COALESCE(application.present, FALSE) AS observed_application,
	COALESCE(application.versions, ARRAY[]::TEXT[]) AS observed_versions,
	resolved.actions,
	resolved.package_selection,
	resolved.pinned_package_id,
	pinned.version AS package_version,
	observed.name AS observation_name,
	observed.installed AS munki_installed,
	observed.target_version AS munki_target_version,
	contact.last_seen_at AS munki_last_seen_at,
	envelope.last_successful_at AS munki_last_successful_at,
	COALESCE(envelope.collection_error, '') AS munki_collection_error,
	COALESCE(envelope.has_report, FALSE) AS munki_has_report
FROM hosts h
CROSS JOIN LATERAL munki_resolved_software_for_host(h.id) resolved
JOIN munki_software software ON software.id = resolved.software_id
LEFT JOIN munki_packages pinned ON pinned.id = resolved.pinned_package_id
LEFT JOIN host_heartbeats contact ON contact.host_id = h.id AND contact.source = 'munki'
LEFT JOIN munki_host_status envelope ON envelope.host_id = h.id
LEFT JOIN munki_host_items observed
	ON observed.host_id = h.id
	AND observed.name = resolved.name
LEFT JOIN LATERAL (
	SELECT
		COUNT(*) > 0 AS present,
		COALESCE(
			array_agg(DISTINCT NULLIF(
				CASE software.installation_detector_version_source
					WHEN 'bundle_short_version' THEN path.bundle_short_version
					WHEN 'bundle_version' THEN path.bundle_version
				END,
				''
			)) FILTER (
				WHERE CASE software.installation_detector_version_source
					WHEN 'bundle_short_version' THEN path.bundle_short_version
					WHEN 'bundle_version' THEN path.bundle_version
					END <> ''
			),
			ARRAY[]::TEXT[]
		) AS versions
	FROM host_software host_application
	JOIN software application ON application.id = host_application.software_id
	LEFT JOIN host_software_installed_paths path
		ON path.host_id = host_application.host_id
		AND path.software_id = host_application.software_id
	WHERE host_application.host_id = h.id
		AND application.source = 'apps'
		AND application.bundle_identifier = software.installation_detector_bundle_identifier
		AND (
			COALESCE(software.installation_detector_expected_path, '') = ''
			OR path.installed_path = software.installation_detector_expected_path
		)
) application ON software.installation_detector_bundle_identifier IS NOT NULL`,
		WhereSQL: whereSQL,
		Args:     args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name": {SQL: "lower(software.name)"},
		},
		DefaultOrder: []dbutil.OrderExpr{
			{SQL: "lower(software.name)"},
			{SQL: "resolved.software_id"},
		},
		Params: params.ListParams,
	}
	rows, count, err := dbutil.ListWithCount[hostSoftwareRow](ctx, s.db.Pool(), query)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		var hostExists bool
		if err := s.db.Pool().QueryRow(
			ctx,
			`SELECT EXISTS (SELECT 1 FROM hosts WHERE id = $1)`,
			hostID,
		).Scan(&hostExists); err != nil {
			return nil, 0, err
		}
		if !hostExists {
			return nil, 0, dbutil.ErrNotFound
		}
	}

	software := make([]HostManifestSoftware, len(rows))
	for i, row := range rows {
		software[i] = hostSoftwareFromRow(row)
	}
	return software, count, nil
}

func hostSoftwareFromRow(row hostSoftwareRow) HostManifestSoftware {
	selector := packageSelectorFromStorage(row.PackageSelection, row.PinnedPackageID)
	pkg := HostManifestPackage{Strategy: selector.Strategy}
	if selector.Strategy == PackageSpecific {
		pkg.ID = selector.PackageID
		pkg.Version = valueOrEmpty(row.PackageVersion)
	}
	fact := deploymentFact{
		SoftwareID:                   row.SoftwareID,
		HasInstallationDetector:      row.HasInstallationDetector,
		ObservedInstallationEligible: row.ObservedInstallationEligible,
		SoftwareInventoryUpdatedAt:   row.SoftwareInventoryUpdatedAt,
		ObservedApplication:          row.ObservedApplication,
		ObservedVersions:             row.ObservedVersions,
		MunkiLastSeenAt:              row.MunkiLastSeenAt,
		MunkiLastSuccessfulAt:        row.MunkiLastSuccessfulAt,
		MunkiCollectionError:         row.MunkiCollectionError,
		MunkiHasReport:               row.MunkiHasReport,
		Actions:                      actionsFromStorage(row.Actions),
		Package:                      pkg,
	}
	software := HostManifestSoftware{
		Software: packages.PackageSoftware{
			ID:           row.SoftwareID,
			Name:         row.SoftwareName,
			DisplayName:  row.SoftwareDisplayName,
			Description:  row.SoftwareDescription,
			Category:     row.SoftwareCategory,
			Developer:    row.SoftwareDeveloper,
			IconObjectID: row.SoftwareIconObjectID,
			IconURL:      IconURL(row.SoftwareIconObjectID),
		},
		Package: pkg,
		Actions: fact.Actions,
	}
	if row.ObservationName != nil {
		fact.Observation = &munkiObservation{
			Installed:     valueOrFalse(row.MunkiInstalled),
			TargetVersion: valueOrEmpty(row.MunkiTargetVersion),
		}
	}
	software.Status = installationStatus(fact)
	software.InstalledVersion = installedVersion(fact)
	software.MunkiResult = munkiResult(fact)
	software.TargetVersion = targetVersion(fact)
	software.LastCollectedAt = fact.SoftwareInventoryUpdatedAt
	return software
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func valueOrFalse(value *bool) bool {
	return value != nil && *value
}
