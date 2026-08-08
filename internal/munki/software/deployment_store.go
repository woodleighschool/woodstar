package software

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

type deploymentFactRow struct {
	SoftwareID                   int64      `db:"software_id"`
	HostID                       int64      `db:"host_id"`
	DisplayName                  string     `db:"display_name"`
	HardwareSerial               string     `db:"hardware_serial"`
	HasInstallationDetector      bool       `db:"has_installation_detector"`
	ObservedInstallationEligible bool       `db:"observed_installation_eligible"`
	SoftwareInventoryUpdatedAt   *time.Time `db:"software_inventory_updated_at"`
	ObservedApplication          bool       `db:"observed_application"`
	ObservedVersions             []string   `db:"observed_versions"`
	MunkiLastSeenAt              *time.Time `db:"munki_last_seen_at"`
	MunkiLastSuccessfulAt        *time.Time `db:"munki_last_successful_at"`
	MunkiCollectionError         string     `db:"munki_collection_error"`
	MunkiHasReport               bool       `db:"munki_has_report"`
	Actions                      []string   `db:"actions"`
	PackageSelection             string     `db:"package_selection"`
	PinnedPackageID              *int64     `db:"pinned_package_id"`
	PackageVersion               *string    `db:"package_version"`
	ObservationName              *string    `db:"observation_name"`
	MunkiInstalled               *bool      `db:"munki_installed"`
	MunkiTargetVersion           *string    `db:"munki_target_version"`
}

// ListWithDeployment returns the normal software page enriched from one batched fact read.
func (s *Store) ListWithDeployment(
	ctx context.Context,
	params dbutil.ListParams,
) ([]SoftwareWithDeployment, int, error) {
	rows, count, err := s.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int64, len(rows))
	for i, software := range rows {
		ids[i] = software.ID
	}
	facts, err := s.loadDeploymentFacts(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	factsBySoftware := make(map[int64][]deploymentFact, len(ids))
	for _, fact := range facts {
		factsBySoftware[fact.SoftwareID] = append(factsBySoftware[fact.SoftwareID], fact)
	}

	result := make([]SoftwareWithDeployment, len(rows))
	for i, software := range rows {
		result[i] = SoftwareWithDeployment{
			Software:   software,
			Deployment: deploymentSummary(factsBySoftware[software.ID]),
		}
	}
	return result, count, nil
}

// GetDeployment returns the current installation counts for one software title.
func (s *Store) GetDeployment(ctx context.Context, softwareID int64) (DeploymentSummary, error) {
	if _, err := s.GetByID(ctx, softwareID); err != nil {
		return DeploymentSummary{}, err
	}
	facts, err := s.loadDeploymentFacts(ctx, []int64{softwareID})
	if err != nil {
		return DeploymentSummary{}, err
	}
	return deploymentSummary(facts), nil
}

// ListDeploymentHosts returns the effective hosts for one software title.
func (s *Store) ListDeploymentHosts(
	ctx context.Context,
	softwareID int64,
	params DeploymentHostListParams,
) ([]DeploymentHost, int, error) {
	params.ListParams = dbutil.NormalizeListParams(params.ListParams)
	if err := validateDeploymentHostListParams(params); err != nil {
		return nil, 0, err
	}
	if _, err := s.GetByID(ctx, softwareID); err != nil {
		return nil, 0, err
	}
	facts, err := s.loadDeploymentFacts(ctx, []int64{softwareID})
	if err != nil {
		return nil, 0, err
	}

	hosts := make([]DeploymentHost, 0, len(facts))
	for _, fact := range facts {
		host := deploymentHostFromFact(fact)
		if params.Status != nil && host.Status != *params.Status {
			continue
		}
		if params.MunkiResult != nil && host.MunkiResult != *params.MunkiResult {
			continue
		}
		if params.Action != nil && !slices.Contains(fact.Actions, *params.Action) {
			continue
		}
		if params.Q != "" && !deploymentHostMatchesQuery(fact, params.Q) {
			continue
		}
		hosts = append(hosts, host)
	}
	sortDeploymentHosts(hosts, params.Sort)
	count := len(hosts)
	start := min(int(params.PageIndex)*int(params.PageSize), count)
	end := min(start+int(params.PageSize), count)
	return hosts[start:end], count, nil
}

func (s *Store) loadDeploymentFacts(ctx context.Context, softwareIDs []int64) ([]deploymentFact, error) {
	if len(softwareIDs) == 0 {
		return nil, nil
	}
	qrows, err := s.db.Pool().Query(ctx, `
SELECT
	software.id AS software_id,
	h.id AS host_id,
	h.display_name,
	h.hardware_serial,
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
	contact.last_seen_at AS munki_last_seen_at,
	envelope.last_successful_at AS munki_last_successful_at,
	COALESCE(envelope.collection_error, '') AS munki_collection_error,
	COALESCE(envelope.has_report, FALSE) AS munki_has_report,
	resolved.actions,
	resolved.package_selection,
	resolved.pinned_package_id,
	pinned.version AS package_version,
	munki_item.name AS observation_name,
	munki_item.installed AS munki_installed,
	munki_item.target_version AS munki_target_version
FROM hosts h
CROSS JOIN LATERAL munki_resolved_software_for_host(h.id) resolved
JOIN munki_software software ON software.id = resolved.software_id
LEFT JOIN munki_packages pinned ON pinned.id = resolved.pinned_package_id
LEFT JOIN host_heartbeats contact ON contact.host_id = h.id AND contact.source = 'munki'
LEFT JOIN munki_host_status envelope ON envelope.host_id = h.id
LEFT JOIN munki_host_items munki_item
	ON munki_item.host_id = h.id
	AND munki_item.name = resolved.name
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
) application ON software.installation_detector_bundle_identifier IS NOT NULL
WHERE software.id = ANY($1::BIGINT[])
ORDER BY software.id, h.id`, softwareIDs)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[deploymentFactRow])
	if err != nil {
		return nil, err
	}
	facts := make([]deploymentFact, len(rows))
	for i, row := range rows {
		facts[i] = deploymentFactFromRow(row)
	}
	return facts, nil
}

func deploymentFactFromRow(row deploymentFactRow) deploymentFact {
	selector := packageSelectorFromStorage(row.PackageSelection, row.PinnedPackageID)
	fact := deploymentFact{
		SoftwareID:                   row.SoftwareID,
		HostID:                       row.HostID,
		DisplayName:                  row.DisplayName,
		HardwareSerial:               row.HardwareSerial,
		HasInstallationDetector:      row.HasInstallationDetector,
		ObservedInstallationEligible: row.ObservedInstallationEligible,
		SoftwareInventoryUpdatedAt:   row.SoftwareInventoryUpdatedAt,
		ObservedApplication:          row.ObservedApplication,
		ObservedVersions:             slices.Clone(row.ObservedVersions),
		MunkiLastSeenAt:              row.MunkiLastSeenAt,
		MunkiLastSuccessfulAt:        row.MunkiLastSuccessfulAt,
		MunkiCollectionError:         row.MunkiCollectionError,
		MunkiHasReport:               row.MunkiHasReport,
		Actions:                      actionsFromStorage(row.Actions),
		Package: HostManifestPackage{
			Strategy: selector.Strategy,
		},
	}
	if selector.Strategy == PackageSpecific {
		fact.Package.ID = selector.PackageID
		fact.Package.Version = valueOrEmpty(row.PackageVersion)
	}
	if row.ObservationName != nil {
		fact.Observation = &munkiObservation{
			Installed:     valueOrFalse(row.MunkiInstalled),
			TargetVersion: valueOrEmpty(row.MunkiTargetVersion),
		}
	}
	return fact
}

func deploymentHostFromFact(fact deploymentFact) DeploymentHost {
	return DeploymentHost{
		HostID:           fact.HostID,
		DisplayName:      fact.DisplayName,
		HardwareSerial:   fact.HardwareSerial,
		Actions:          slices.Clone(fact.Actions),
		Package:          fact.Package,
		Status:           installationStatus(fact),
		InstalledVersion: installedVersion(fact),
		MunkiResult:      munkiResult(fact),
		TargetVersion:    targetVersion(fact),
		LastCollectedAt:  fact.SoftwareInventoryUpdatedAt,
	}
}

func validateDeploymentHostListParams(params DeploymentHostListParams) error {
	if err := dbutil.ValidateListParams(params.ListParams); err != nil {
		return err
	}
	if params.Status != nil && !slices.Contains(installationStatusValues, *params.Status) {
		return fmt.Errorf("%w: invalid installation status %q", dbutil.ErrInvalidInput, *params.Status)
	}
	if params.MunkiResult != nil && !slices.Contains(munkiResultValues, *params.MunkiResult) {
		return fmt.Errorf("%w: invalid Munki result %q", dbutil.ErrInvalidInput, *params.MunkiResult)
	}
	if params.Action != nil && !slices.Contains(actionValues, *params.Action) {
		return fmt.Errorf("%w: invalid Munki action %q", dbutil.ErrInvalidInput, *params.Action)
	}
	_, err := dbutil.OrderBy(params.ListParams, deploymentHostOrderKeys(), nil)
	return err
}

func deploymentHostMatchesQuery(fact deploymentFact, query string) bool {
	query = strings.ToLower(query)
	return strings.Contains(strings.ToLower(fact.DisplayName), query) ||
		strings.Contains(strings.ToLower(fact.HardwareSerial), query)
}

func deploymentHostOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"display_name":      {SQL: "display_name"},
		"hardware_serial":   {SQL: "hardware_serial"},
		"status":            {SQL: "status"},
		"installed_version": {SQL: "installed_version"},
		"munki_result":      {SQL: "munki_result"},
		"target_version":    {SQL: "target_version"},
		"last_collected_at": {SQL: "last_collected_at"},
	}
}

func sortDeploymentHosts(hosts []DeploymentHost, order string) {
	key, descending := deploymentHostSort(order)
	slices.SortStableFunc(hosts, func(a, b DeploymentHost) int {
		result := deploymentHostCompare(a, b, key)
		if descending {
			result = -result
		}
		if result != 0 {
			return result
		}
		if result = cmp.Compare(strings.ToLower(a.DisplayName), strings.ToLower(b.DisplayName)); result != 0 {
			return result
		}
		return cmp.Compare(a.HostID, b.HostID)
	})
}

func deploymentHostSort(order string) (string, bool) {
	if order == "" {
		return "display_name", false
	}
	key, suffix, found := strings.Cut(order, ".")
	if !found {
		return key, false
	}
	return key, suffix == "desc"
}

func deploymentHostCompare(a, b DeploymentHost, key string) int {
	switch key {
	case "display_name":
		return cmp.Compare(strings.ToLower(a.DisplayName), strings.ToLower(b.DisplayName))
	case "hardware_serial":
		return cmp.Compare(strings.ToLower(a.HardwareSerial), strings.ToLower(b.HardwareSerial))
	case "status":
		return cmp.Compare(a.Status, b.Status)
	case "installed_version":
		return cmp.Compare(a.InstalledVersion, b.InstalledVersion)
	case "munki_result":
		return cmp.Compare(a.MunkiResult, b.MunkiResult)
	case "target_version":
		return cmp.Compare(a.TargetVersion, b.TargetVersion)
	case "last_collected_at":
		return compareOptionalTimes(a.LastCollectedAt, b.LastCollectedAt)
	default:
		return 0
	}
}

func compareOptionalTimes(a, b *time.Time) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	return a.Compare(*b)
}
