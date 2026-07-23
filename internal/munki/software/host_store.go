package software

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

type hostSoftwareRow struct {
	SoftwareID           int64    `db:"software_id"`
	SoftwareName         string   `db:"software_name"`
	SoftwareDisplayName  *string  `db:"software_display_name"`
	SoftwareDescription  string   `db:"software_description"`
	SoftwareCategory     string   `db:"software_category"`
	SoftwareDeveloper    string   `db:"software_developer"`
	SoftwareIconObjectID *int64   `db:"software_icon_object_id"`
	Actions              []string `db:"actions"`
	PackageSelection     string   `db:"package_selection"`
	PinnedPackageID      *int64   `db:"pinned_package_id"`
	PackageVersion       *string  `db:"package_version"`
	ObservationName      *string  `db:"observation_name"`
	DisplayName          *string  `db:"display_name"`
	Installed            *bool    `db:"installed"`
	InstalledVersion     *string  `db:"installed_version"`
	TargetVersion        *string  `db:"target_version"`
}

// ListForHost returns the exact software enumeration used to render a host manifest.
func (s *Store) ListForHost(
	ctx context.Context,
	hostID int64,
	params HostManifestSoftwareListParams,
) ([]HostManifestSoftware, int, error) {
	params.ListParams = dbutil.NormalizeListParams(params.ListParams)
	whereSQL := ""
	args := []any{hostID}
	if params.Q != "" {
		whereSQL = "WHERE software.name ILIKE $2"
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
	resolved.actions,
	resolved.package_selection,
	resolved.pinned_package_id,
	pinned.version AS package_version,
	observed.name AS observation_name,
	observed.display_name,
	observed.installed,
	observed.installed_version,
	observed.target_version
FROM munki_resolved_software_for_host($1) resolved
JOIN munki_software software ON software.id = resolved.software_id
LEFT JOIN munki_packages pinned ON pinned.id = resolved.pinned_package_id
LEFT JOIN munki_host_items observed
	ON observed.host_id = $1
	AND observed.name = resolved.name`,
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
		Actions: actionsFromStorage(row.Actions),
	}
	if row.ObservationName != nil {
		software.Observation = &HostManifestSoftwareObservation{
			DisplayName:      valueOrEmpty(row.DisplayName),
			Installed:        valueOrFalse(row.Installed),
			InstalledVersion: valueOrEmpty(row.InstalledVersion),
			TargetVersion:    valueOrEmpty(row.TargetVersion),
		}
	}
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
