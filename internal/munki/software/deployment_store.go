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
	SoftwareID         int64      `db:"software_id"`
	HostID             int64      `db:"host_id"`
	DisplayName        string     `db:"display_name"`
	HardwareSerial     string     `db:"hardware_serial"`
	MunkiLastSeenAt    *time.Time `db:"munki_last_seen_at"`
	LastAttemptAt      *time.Time `db:"last_attempt_at"`
	LastSuccessfulAt   *time.Time `db:"last_successful_at"`
	CollectionError    string     `db:"collection_error"`
	HasReport          bool       `db:"has_report"`
	Actions            []string   `db:"actions"`
	PackageSelection   string     `db:"package_selection"`
	PinnedPackageID    *int64     `db:"pinned_package_id"`
	PackageVersion     *string    `db:"package_version"`
	ObservationName    *string    `db:"observation_name"`
	ObservationDisplay *string    `db:"observation_display_name"`
	Installed          *bool      `db:"installed"`
	InstalledVersion   *string    `db:"installed_version"`
	TargetVersion      *string    `db:"target_version"`
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
		if params.Status != nil && (host.Status == nil || *host.Status != *params.Status) {
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
	contact.last_seen_at AS munki_last_seen_at,
	envelope.last_attempt_at,
	envelope.last_successful_at,
	COALESCE(envelope.collection_error, '') AS collection_error,
	COALESCE(envelope.has_report, FALSE) AS has_report,
	resolved.actions,
	resolved.package_selection,
	resolved.pinned_package_id,
	pinned.version AS package_version,
	observed.name AS observation_name,
	observed.display_name AS observation_display_name,
	observed.installed,
	observed.installed_version,
	observed.target_version
FROM hosts h
CROSS JOIN LATERAL munki_resolved_software_for_host(h.id) resolved
JOIN munki_software software ON software.id = resolved.software_id
LEFT JOIN munki_packages pinned ON pinned.id = resolved.pinned_package_id
LEFT JOIN host_heartbeats contact ON contact.host_id = h.id AND contact.source = 'munki'
LEFT JOIN munki_host_status envelope ON envelope.host_id = h.id
LEFT JOIN munki_host_items observed
	ON observed.host_id = h.id
	AND observed.name = resolved.name
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
		SoftwareID:       row.SoftwareID,
		HostID:           row.HostID,
		DisplayName:      row.DisplayName,
		HardwareSerial:   row.HardwareSerial,
		MunkiLastSeenAt:  row.MunkiLastSeenAt,
		LastAttemptAt:    row.LastAttemptAt,
		LastSuccessfulAt: row.LastSuccessfulAt,
		CollectionError:  row.CollectionError,
		HasReport:        row.HasReport,
		Actions:          actionsFromStorage(row.Actions),
		Package: HostManifestPackage{
			Strategy: selector.Strategy,
		},
	}
	if selector.Strategy == PackageSpecific {
		fact.Package.ID = selector.PackageID
		fact.Package.Version = valueOrEmpty(row.PackageVersion)
	}
	if row.ObservationName != nil {
		fact.Observation = &HostManifestSoftwareObservation{
			DisplayName:      valueOrEmpty(row.ObservationDisplay),
			Installed:        valueOrFalse(row.Installed),
			InstalledVersion: valueOrEmpty(row.InstalledVersion),
			TargetVersion:    valueOrEmpty(row.TargetVersion),
		}
	}
	return fact
}

func deploymentHostFromFact(fact deploymentFact) DeploymentHost {
	state := deploymentReportState(fact)
	host := DeploymentHost{
		HostID:           fact.HostID,
		DisplayName:      fact.DisplayName,
		HardwareSerial:   fact.HardwareSerial,
		Actions:          slices.Clone(fact.Actions),
		Package:          fact.Package,
		ReportState:      state,
		Status:           deploymentStatus(fact),
		LastAttemptAt:    fact.LastAttemptAt,
		LastSuccessfulAt: fact.LastSuccessfulAt,
		CollectionError:  fact.CollectionError,
	}
	if state == ReportCurrent {
		if fact.Observation != nil {
			host.Installed = fact.Observation.Installed
			host.InstalledVersion = canonicalDeploymentVersion(fact.Observation.InstalledVersion)
			host.TargetVersion = canonicalDeploymentVersion(fact.Observation.TargetVersion)
		}
	}
	return host
}

func validateDeploymentHostListParams(params DeploymentHostListParams) error {
	if err := dbutil.ValidateListParams(params.ListParams); err != nil {
		return err
	}
	if params.Status != nil && !slices.Contains(deploymentStatusValues, *params.Status) {
		return fmt.Errorf("%w: invalid deployment status %q", dbutil.ErrInvalidInput, *params.Status)
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
		"display_name":       {SQL: "display_name"},
		"hardware_serial":    {SQL: "hardware_serial"},
		"status":             {SQL: "status"},
		"installed_version":  {SQL: "installed_version"},
		"target_version":     {SQL: "target_version"},
		"last_successful_at": {SQL: "last_successful_at"},
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
		return cmp.Compare(deploymentStatusValue(a.Status), deploymentStatusValue(b.Status))
	case "installed_version":
		return cmp.Compare(a.InstalledVersion, b.InstalledVersion)
	case "target_version":
		return cmp.Compare(a.TargetVersion, b.TargetVersion)
	case "last_successful_at":
		return compareOptionalTimes(a.LastSuccessfulAt, b.LastSuccessfulAt)
	default:
		return 0
	}
}

func deploymentStatusValue(status *DeploymentStatus) DeploymentStatus {
	if status == nil {
		return ""
	}
	return *status
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
