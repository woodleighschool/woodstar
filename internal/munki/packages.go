package munki

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func (s *Store) CreatePackage(ctx context.Context, params PackageMutation) (*Package, error) {
	params = cleanPackageMutation(params)
	if err := params.Validate(); err != nil {
		return nil, err
	}
	title, err := s.GetSoftwareTitle(ctx, params.SoftwareID)
	if err != nil {
		return nil, err
	}
	params = fillPackageDefaults(params, *title)
	params, err = s.normalizePackageArtifacts(ctx, params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.CreateMunkiPackage(ctx, sqlc.CreateMunkiPackageParams{
		SoftwareID:             params.SoftwareID,
		Name:                   params.Name,
		Version:                params.Version,
		DisplayName:            params.DisplayName,
		Description:            params.Description,
		Category:               params.Category,
		Developer:              params.Developer,
		InstallerType:          sqlcString(params.InstallerType),
		UninstallMethod:        params.UninstallMethod,
		RestartAction:          sqlcString(params.RestartAction),
		MinimumMunkiVersion:    params.MinimumMunkiVersion,
		MinimumOSVersion:       params.MinimumOSVersion,
		MaximumOSVersion:       params.MaximumOSVersion,
		SupportedArchitectures: params.SupportedArchitectures,
		BlockingApplications:   params.BlockingApplications,
		Requires:               params.Requires,
		UpdateFor:              params.UpdateFor,
		UnattendedInstall:      params.UnattendedInstall,
		UnattendedUninstall:    params.UnattendedUninstall,
		Uninstallable:          params.Uninstallable,
		OnDemand:               params.OnDemand,
		Precache:               params.Precache,
		IconName:               params.IconName,
		IconHash:               params.IconHash,
		ExtraPkginfo:           cleanExtraPkginfo(params.ExtraPkginfo),
		InstallerArtifactID:    params.InstallerArtifactID,
		IconArtifactID:         params.IconArtifactID,
		Eligible:               params.Eligible,
	})
	if err != nil {
		return nil, mapDesiredMutationError(err)
	}
	return s.GetPackage(ctx, row.ID)
}

func (s *Store) UpdatePackage(ctx context.Context, id int64, params PackageMutation) (*Package, error) {
	existing, err := s.GetPackage(ctx, id)
	if err != nil {
		return nil, err
	}
	if params.SoftwareID != 0 && params.SoftwareID != existing.SoftwareID {
		return nil, fmt.Errorf("%w: software_id cannot be changed", dbutil.ErrInvalidInput)
	}
	params = mergePackageUpdate(*existing, params)
	params = cleanPackageMutation(params)
	if err := params.Validate(); err != nil {
		return nil, err
	}
	title, err := s.GetSoftwareTitle(ctx, params.SoftwareID)
	if err != nil {
		return nil, err
	}
	params = fillPackageDefaults(params, *title)
	params, err = s.normalizePackageArtifacts(ctx, params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.UpdateMunkiPackage(ctx, sqlc.UpdateMunkiPackageParams{
		ID:                     id,
		Name:                   params.Name,
		Version:                params.Version,
		DisplayName:            params.DisplayName,
		Description:            params.Description,
		Category:               params.Category,
		Developer:              params.Developer,
		InstallerType:          sqlcString(params.InstallerType),
		UninstallMethod:        params.UninstallMethod,
		RestartAction:          sqlcString(params.RestartAction),
		MinimumMunkiVersion:    params.MinimumMunkiVersion,
		MinimumOSVersion:       params.MinimumOSVersion,
		MaximumOSVersion:       params.MaximumOSVersion,
		SupportedArchitectures: params.SupportedArchitectures,
		BlockingApplications:   params.BlockingApplications,
		Requires:               params.Requires,
		UpdateFor:              params.UpdateFor,
		UnattendedInstall:      params.UnattendedInstall,
		UnattendedUninstall:    params.UnattendedUninstall,
		Uninstallable:          params.Uninstallable,
		OnDemand:               params.OnDemand,
		Precache:               params.Precache,
		IconName:               params.IconName,
		IconHash:               params.IconHash,
		ExtraPkginfo:           cleanExtraPkginfo(params.ExtraPkginfo),
		InstallerArtifactID:    params.InstallerArtifactID,
		IconArtifactID:         params.IconArtifactID,
		Eligible:               params.Eligible,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, mapDesiredMutationError(err)
	}
	return s.GetPackage(ctx, row.ID)
}

func (s *Store) UpsertPackage(ctx context.Context, params PackageMutation) (*Package, error) {
	params = cleanPackageMutation(params)
	if err := params.Validate(); err != nil {
		return nil, err
	}
	title, err := s.GetSoftwareTitle(ctx, params.SoftwareID)
	if err != nil {
		return nil, err
	}
	params = fillPackageDefaults(params, *title)
	params, err = s.normalizePackageArtifacts(ctx, params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.UpsertMunkiPackage(ctx, sqlc.UpsertMunkiPackageParams{
		SoftwareID:             params.SoftwareID,
		Name:                   params.Name,
		Version:                params.Version,
		DisplayName:            params.DisplayName,
		Description:            params.Description,
		Category:               params.Category,
		Developer:              params.Developer,
		InstallerType:          sqlcString(params.InstallerType),
		UninstallMethod:        params.UninstallMethod,
		RestartAction:          sqlcString(params.RestartAction),
		MinimumMunkiVersion:    params.MinimumMunkiVersion,
		MinimumOSVersion:       params.MinimumOSVersion,
		MaximumOSVersion:       params.MaximumOSVersion,
		SupportedArchitectures: params.SupportedArchitectures,
		BlockingApplications:   params.BlockingApplications,
		Requires:               params.Requires,
		UpdateFor:              params.UpdateFor,
		UnattendedInstall:      params.UnattendedInstall,
		UnattendedUninstall:    params.UnattendedUninstall,
		Uninstallable:          params.Uninstallable,
		OnDemand:               params.OnDemand,
		Precache:               params.Precache,
		IconName:               params.IconName,
		IconHash:               params.IconHash,
		ExtraPkginfo:           cleanExtraPkginfo(params.ExtraPkginfo),
		InstallerArtifactID:    params.InstallerArtifactID,
		IconArtifactID:         params.IconArtifactID,
		Eligible:               params.Eligible,
	})
	if err != nil {
		return nil, mapDesiredMutationError(err)
	}
	return s.GetPackage(ctx, row.ID)
}

func (s *Store) GetPackage(ctx context.Context, id int64) (*Package, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.q.GetMunkiPackageByID(ctx, sqlc.GetMunkiPackageByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	pkg, err := packageFromRecord(packageRecordFromSQLC(row))
	if err != nil {
		return nil, err
	}
	return &pkg, nil
}

func (s *Store) ListPackages(ctx context.Context, params PackageListParams) ([]Package, int, error) {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	where, args := packageListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL: packageSelectSQL,
		WhereSQL:  where,
		Args:      args,
		OrderKeys: packageOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{
			{SQL: "lower(COALESCE(NULLIF(p.display_name, ''), p.name))"},
			{SQL: "lower(p.version)"},
			{SQL: "p.id"},
		},
		Params: params.ListParams,
	}
	var count int
	countSQL, countArgs := listQuery.BuildCount()
	if err := s.db.Pool().QueryRow(ctx, countSQL, countArgs...).Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := listQuery.Build()
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[packageRecord])
	if err != nil {
		return nil, 0, err
	}
	packages, err := packagesFromRecords(records)
	if err != nil {
		return nil, 0, err
	}
	return packages, count, nil
}

func (s *Store) EffectivePackagesForHost(ctx context.Context, hostID int64) ([]EffectivePackage, error) {
	rows, err := s.q.ListEffectiveMunkiPackagesForHost(ctx, sqlc.ListEffectiveMunkiPackagesForHostParams{
		HostID: hostID,
	})
	if err != nil {
		return nil, err
	}
	packages := make([]EffectivePackage, 0, len(rows))
	for _, row := range rows {
		pkg := Package{}
		if row.AssignmentEffect == sqlc.MunkiAssignmentEffectInclude {
			resolvedPackage, err := packageFromRecord(packageRecordFromEffectiveSQLC(row))
			if err != nil {
				return nil, err
			}
			pkg = resolvedPackage
		}
		packages = append(packages, EffectivePackage{
			AssignmentID:     row.AssignmentID,
			SoftwareID:       row.AssignmentSoftwareID,
			AssignmentEffect: AssignmentEffect(row.AssignmentEffect),
			Action:           assignmentActionValue(row.Action),
			OptionalInstall:  row.OptionalInstall,
			FeaturedItem:     row.FeaturedItem,
			PackageSelection: packageSelectionValue(row.PackageSelection),
			PinnedPackageID:  row.PinnedPackageID,
			Priority:         row.Priority,
			Package:          pkg,
		})
	}
	return resolveEffectivePackages(packages), nil
}

func (s *Store) normalizePackageArtifacts(ctx context.Context, params PackageMutation) (PackageMutation, error) {
	if params.InstallerArtifactID != nil {
		artifact, err := s.GetArtifact(ctx, *params.InstallerArtifactID)
		if err != nil {
			return params, err
		}
		if artifact.Kind != ArtifactKindPackage {
			return params, fmt.Errorf(
				"%w: installer_artifact_id must reference a package artifact",
				dbutil.ErrInvalidInput,
			)
		}
	}
	if params.IconArtifactID != nil {
		artifact, err := s.GetArtifact(ctx, *params.IconArtifactID)
		if err != nil {
			return params, err
		}
		if artifact.Kind != ArtifactKindIcon {
			return params, fmt.Errorf(
				"%w: icon_artifact_id must reference an icon artifact",
				dbutil.ErrInvalidInput,
			)
		}
		params.IconName = artifact.Location
		params.IconHash = artifact.SHA256
	}
	return params, nil
}

func cleanPackageMutation(params PackageMutation) PackageMutation {
	params.Name = strings.TrimSpace(params.Name)
	params.Version = strings.TrimSpace(params.Version)
	params.DisplayName = strings.TrimSpace(params.DisplayName)
	if params.DisplayName == "" {
		params.DisplayName = params.Name
	}
	params.Description = strings.TrimSpace(params.Description)
	params.Category = strings.TrimSpace(params.Category)
	params.Developer = strings.TrimSpace(params.Developer)
	params.InstallerType = InstallerType(strings.TrimSpace(string(params.InstallerType)))
	if params.InstallerType == "" {
		params.InstallerType = InstallerTypePkg
	}
	params.UninstallMethod = strings.TrimSpace(params.UninstallMethod)
	params.RestartAction = RestartAction(strings.TrimSpace(string(params.RestartAction)))
	params.MinimumMunkiVersion = strings.TrimSpace(params.MinimumMunkiVersion)
	params.MinimumOSVersion = strings.TrimSpace(params.MinimumOSVersion)
	params.MaximumOSVersion = strings.TrimSpace(params.MaximumOSVersion)
	params.SupportedArchitectures = cleanStringList(params.SupportedArchitectures)
	params.BlockingApplications = cleanStringList(params.BlockingApplications)
	params.Requires = cleanStringList(params.Requires)
	params.UpdateFor = cleanStringList(params.UpdateFor)
	params.IconName = strings.TrimSpace(params.IconName)
	params.IconHash = strings.TrimSpace(params.IconHash)
	params.ExtraPkginfo = cleanExtraPkginfo(params.ExtraPkginfo)
	return params
}

func mergePackageUpdate(existing Package, params PackageMutation) PackageMutation {
	params.SoftwareID = existing.SoftwareID
	if params.MinimumMunkiVersion == "" {
		params.MinimumMunkiVersion = existing.MinimumMunkiVersion
	}
	if params.BlockingApplications == nil {
		params.BlockingApplications = existing.BlockingApplications
	}
	if params.Requires == nil {
		params.Requires = existing.Requires
	}
	if params.UpdateFor == nil {
		params.UpdateFor = existing.UpdateFor
	}
	if len(params.ExtraPkginfo) == 0 {
		params.ExtraPkginfo = existing.ExtraPkginfo
	}
	if params.InstallerArtifactID == nil {
		params.InstallerArtifactID = existing.InstallerArtifactID
	}
	return params
}

func fillPackageDefaults(params PackageMutation, title SoftwareTitle) PackageMutation {
	if params.DisplayName == "" {
		params.DisplayName = title.DisplayName
	}
	if params.Description == "" {
		params.Description = title.Description
	}
	if params.Category == "" {
		params.Category = title.Category
	}
	if params.Developer == "" {
		params.Developer = title.Developer
	}
	return params
}

func packageFromRecord(row packageRecord) (Package, error) {
	pkg := Package{
		ID:                           row.ID,
		SoftwareID:                   row.SoftwareID,
		SoftwareName:                 row.SoftwareName,
		SoftwareDisplayName:          row.SoftwareDisplayName,
		Name:                         row.Name,
		Version:                      row.Version,
		DisplayName:                  row.DisplayName,
		Description:                  row.Description,
		Category:                     row.Category,
		Developer:                    row.Developer,
		InstallerType:                InstallerType(row.InstallerType),
		UnattendedInstall:            row.UnattendedInstall,
		UnattendedUninstall:          row.UnattendedUninstall,
		Uninstallable:                row.Uninstallable,
		UninstallMethod:              row.UninstallMethod,
		RestartAction:                RestartAction(row.RestartAction),
		MinimumMunkiVersion:          row.MinimumMunkiVersion,
		MinimumOSVersion:             row.MinimumOSVersion,
		MaximumOSVersion:             row.MaximumOSVersion,
		SupportedArchitectures:       nonNilStrings(row.SupportedArchitectures),
		BlockingApplications:         nonNilStrings(row.BlockingApplications),
		Requires:                     nonNilStrings(row.Requires),
		UpdateFor:                    nonNilStrings(row.UpdateFor),
		OnDemand:                     row.OnDemand,
		Precache:                     row.Precache,
		IconName:                     row.IconName,
		IconHash:                     row.IconHash,
		ExtraPkginfo:                 cleanExtraPkginfo(row.ExtraPkginfo),
		InstallerArtifactID:          row.InstallerArtifactID,
		InstallerArtifactLocation:    stringPtrValue(row.InstallerArtifactLocation),
		IconArtifactID:               row.IconArtifactID,
		IconArtifactLocation:         stringPtrValue(row.IconArtifactLocation),
		SoftwareIconName:             row.SoftwareIconName,
		SoftwareIconHash:             row.SoftwareIconHash,
		SoftwareIconArtifactID:       row.SoftwareIconArtifactID,
		SoftwareIconArtifactLocation: stringPtrValue(row.SoftwareIconArtifactLocation),
		Eligible:                     row.Eligible,
		CreatedAt:                    row.CreatedAt,
		UpdatedAt:                    row.UpdatedAt,
	}
	pkginfo, err := packagePkginfo(pkg)
	if err != nil {
		return Package{}, err
	}
	pkg.Pkginfo = pkginfo
	return pkg, nil
}

func packagesFromRecords(records []packageRecord) ([]Package, error) {
	packages := make([]Package, len(records))
	for i, row := range records {
		pkg, err := packageFromRecord(row)
		if err != nil {
			return nil, err
		}
		packages[i] = pkg
	}
	return packages, nil
}

func sqlcString[S ~string](value S) string {
	return string(value)
}

func packageListWhere(params PackageListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.SoftwareID > 0 {
		where.Add("p.software_id = " + where.Arg(params.SoftwareID))
	}
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			p.name ILIKE ` + search + `
			OR p.version ILIKE ` + search + `
			OR p.display_name ILIKE ` + search + `
			OR p.description ILIKE ` + search + `
			OR p.category ILIKE ` + search + `
			OR p.developer ILIKE ` + search + `
			OR p.installer_type ILIKE ` + search + `
			OR p.icon_name ILIKE ` + search + `
			OR s.name ILIKE ` + search + `
			OR s.display_name ILIKE ` + search + `
		)`)
	}
	return where.Build()
}

func packageOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"name":       {SQL: "lower(COALESCE(NULLIF(p.display_name, ''), p.name))"},
		"version":    {SQL: "lower(p.version)"},
		"software":   {SQL: "lower(COALESCE(NULLIF(s.display_name, ''), s.name))"},
		"updated_at": {SQL: "p.updated_at"},
	}
}

type packageRecord struct {
	ID                           int64
	SoftwareID                   int64
	SoftwareName                 string
	SoftwareDisplayName          string
	Name                         string
	Version                      string
	DisplayName                  string
	Description                  string
	Category                     string
	Developer                    string
	InstallerType                string
	UninstallMethod              string
	RestartAction                string
	MinimumMunkiVersion          string
	MinimumOSVersion             string
	MaximumOSVersion             string
	SupportedArchitectures       []string
	BlockingApplications         []string
	Requires                     []string
	UpdateFor                    []string
	UnattendedInstall            bool
	UnattendedUninstall          bool
	Uninstallable                bool
	OnDemand                     bool
	Precache                     bool
	IconName                     string
	IconHash                     string
	ExtraPkginfo                 []byte
	InstallerArtifactID          *int64
	InstallerArtifactLocation    *string
	IconArtifactID               *int64
	IconArtifactLocation         *string
	SoftwareIconName             string
	SoftwareIconHash             string
	SoftwareIconArtifactID       *int64
	SoftwareIconArtifactLocation *string
	Eligible                     bool
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
}

func packageRecordFromSQLC(row sqlc.GetMunkiPackageByIDRow) packageRecord {
	return packageRecord{
		ID:                           row.ID,
		SoftwareID:                   row.SoftwareID,
		SoftwareName:                 row.SoftwareName,
		SoftwareDisplayName:          row.SoftwareDisplayName,
		SoftwareIconName:             row.SoftwareIconName,
		SoftwareIconHash:             row.SoftwareIconHash,
		SoftwareIconArtifactID:       row.SoftwareIconArtifactID,
		SoftwareIconArtifactLocation: row.SoftwareIconArtifactLocation,
		Name:                         row.Name,
		Version:                      row.Version,
		DisplayName:                  row.DisplayName,
		Description:                  row.Description,
		Category:                     row.Category,
		Developer:                    row.Developer,
		InstallerType:                row.InstallerType,
		UninstallMethod:              row.UninstallMethod,
		RestartAction:                row.RestartAction,
		MinimumMunkiVersion:          row.MinimumMunkiVersion,
		MinimumOSVersion:             row.MinimumOSVersion,
		MaximumOSVersion:             row.MaximumOSVersion,
		SupportedArchitectures:       row.SupportedArchitectures,
		BlockingApplications:         row.BlockingApplications,
		Requires:                     row.Requires,
		UpdateFor:                    row.UpdateFor,
		UnattendedInstall:            row.UnattendedInstall,
		UnattendedUninstall:          row.UnattendedUninstall,
		Uninstallable:                row.Uninstallable,
		OnDemand:                     row.OnDemand,
		Precache:                     row.Precache,
		IconName:                     row.IconName,
		IconHash:                     row.IconHash,
		ExtraPkginfo:                 row.ExtraPkginfo,
		InstallerArtifactID:          row.InstallerArtifactID,
		InstallerArtifactLocation:    row.InstallerArtifactLocation,
		IconArtifactID:               row.IconArtifactID,
		IconArtifactLocation:         row.IconArtifactLocation,
		Eligible:                     row.Eligible,
		CreatedAt:                    row.CreatedAt,
		UpdatedAt:                    row.UpdatedAt,
	}
}

func packageRecordFromEffectiveSQLC(row sqlc.ListEffectiveMunkiPackagesForHostRow) packageRecord {
	return packageRecord{
		ID:                           row.PackageID,
		SoftwareID:                   row.SoftwareID,
		SoftwareName:                 row.SoftwareName,
		SoftwareDisplayName:          row.SoftwareDisplayName,
		SoftwareIconName:             row.SoftwareIconName,
		SoftwareIconHash:             row.SoftwareIconHash,
		SoftwareIconArtifactID:       row.SoftwareIconArtifactID,
		SoftwareIconArtifactLocation: row.SoftwareIconArtifactLocation,
		Name:                         row.Name,
		Version:                      row.Version,
		DisplayName:                  row.DisplayName,
		Description:                  row.Description,
		Category:                     row.Category,
		Developer:                    row.Developer,
		InstallerType:                row.InstallerType,
		UninstallMethod:              row.UninstallMethod,
		RestartAction:                row.RestartAction,
		MinimumMunkiVersion:          row.MinimumMunkiVersion,
		MinimumOSVersion:             row.MinimumOSVersion,
		MaximumOSVersion:             row.MaximumOSVersion,
		SupportedArchitectures:       row.SupportedArchitectures,
		BlockingApplications:         row.BlockingApplications,
		Requires:                     row.Requires,
		UpdateFor:                    row.UpdateFor,
		UnattendedInstall:            row.UnattendedInstall,
		UnattendedUninstall:          row.UnattendedUninstall,
		Uninstallable:                row.Uninstallable,
		OnDemand:                     row.OnDemand,
		Precache:                     row.Precache,
		IconName:                     row.IconName,
		IconHash:                     row.IconHash,
		ExtraPkginfo:                 row.ExtraPkginfo,
		InstallerArtifactID:          row.InstallerArtifactID,
		InstallerArtifactLocation:    row.InstallerArtifactLocation,
		IconArtifactID:               row.IconArtifactID,
		IconArtifactLocation:         row.IconArtifactLocation,
		Eligible:                     true,
	}
}

const packageSelectSQL = `
SELECT
	p.id,
	p.software_id,
	s.name AS software_name,
	COALESCE(NULLIF(s.display_name, ''), s.name) AS software_display_name,
	s.icon_name AS software_icon_name,
	s.icon_hash AS software_icon_hash,
	s.icon_artifact_id AS software_icon_artifact_id,
	p.name,
	p.version,
	p.display_name,
	p.description,
	p.category,
	p.developer,
	p.installer_type,
	p.uninstall_method,
	p.restart_action,
	p.minimum_munki_version,
	p.minimum_os_version,
	p.maximum_os_version,
	p.supported_architectures,
	p.blocking_applications,
	p.requires,
	p.update_for,
	p.unattended_install,
	p.unattended_uninstall,
	p.uninstallable,
	p.on_demand,
	p.precache,
	p.icon_name,
	p.icon_hash,
	p.extra_pkginfo,
	p.installer_artifact_id,
	art.location AS installer_artifact_location,
	p.icon_artifact_id,
	icon.location AS icon_artifact_location,
	software_icon.location AS software_icon_artifact_location,
	p.eligible,
	p.created_at,
	p.updated_at
FROM munki_packages p
JOIN munki_software_titles s ON s.id = p.software_id
LEFT JOIN munki_artifacts art ON art.id = p.installer_artifact_id
LEFT JOIN munki_artifacts icon ON icon.id = p.icon_artifact_id
LEFT JOIN munki_artifacts software_icon ON software_icon.id = s.icon_artifact_id`
