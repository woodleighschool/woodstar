package munki

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store persists Munki desired state and observed host state.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

func (s *Store) CreateSoftwareTitle(ctx context.Context, params SoftwareTitleMutation) (*SoftwareTitle, error) {
	var err error
	params = cleanSoftwareTitleMutation(params)
	if err := params.Validate(); err != nil {
		return nil, err
	}
	params, err = s.normalizeSoftwareTitleIcon(ctx, params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.CreateMunkiSoftwareTitle(ctx, sqlc.CreateMunkiSoftwareTitleParams{
		Name:           params.Name,
		DisplayName:    params.DisplayName,
		Description:    params.Description,
		Category:       params.Category,
		Developer:      params.Developer,
		IconName:       params.IconName,
		IconHash:       params.IconHash,
		IconArtifactID: params.IconArtifactID,
	})
	if err != nil {
		return nil, mapDesiredMutationError(err)
	}
	title := softwareTitleFromSQLC(row)
	return &title, nil
}

func (s *Store) UpdateSoftwareTitle(
	ctx context.Context,
	id int64,
	params SoftwareTitleMutation,
) (*SoftwareTitle, error) {
	var err error
	params = cleanSoftwareTitleMutation(params)
	if err := params.Validate(); err != nil {
		return nil, err
	}
	params, err = s.normalizeSoftwareTitleIcon(ctx, params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.UpdateMunkiSoftwareTitle(ctx, sqlc.UpdateMunkiSoftwareTitleParams{
		Name:           params.Name,
		DisplayName:    params.DisplayName,
		Description:    params.Description,
		Category:       params.Category,
		Developer:      params.Developer,
		IconName:       params.IconName,
		IconHash:       params.IconHash,
		IconArtifactID: params.IconArtifactID,
		ID:             id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, mapDesiredMutationError(err)
	}
	title := softwareTitleFromSQLC(row)
	return &title, nil
}

func (s *Store) GetSoftwareTitle(ctx context.Context, id int64) (*SoftwareTitle, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.q.GetMunkiSoftwareTitleByID(ctx, sqlc.GetMunkiSoftwareTitleByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	title := softwareTitleFromSQLC(row)
	return &title, nil
}

func (s *Store) GetSoftwareTitleByName(ctx context.Context, name string) (*SoftwareTitle, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.q.GetMunkiSoftwareTitleByName(ctx, sqlc.GetMunkiSoftwareTitleByNameParams{Name: name})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	title := softwareTitleFromSQLC(row)
	return &title, nil
}

func (s *Store) LoadSoftwareTitleDetail(ctx context.Context, id int64) (*SoftwareTitleDetail, error) {
	title, err := s.GetSoftwareTitle(ctx, id)
	if err != nil {
		return nil, err
	}
	packages, _, err := s.ListPackages(ctx, PackageListParams{
		ListParams: dbutil.ListParams{PageSize: 1000},
		SoftwareID: id,
	})
	if err != nil {
		return nil, err
	}
	deployments, _, err := s.ListDeployments(ctx, DeploymentListParams{
		ListParams: dbutil.ListParams{PageSize: 1000, Sort: "position.asc"},
		SoftwareID: id,
	})
	if err != nil {
		return nil, err
	}
	return &SoftwareTitleDetail{
		SoftwareTitle: *title,
		Packages:      packages,
		Deployments:   deployments,
	}, nil
}

func (s *Store) ListSoftwareTitles(ctx context.Context, params dbutil.ListParams) ([]SoftwareTitle, int, error) {
	params = dbutil.CleanListParams(params)
	where, args := softwareTitleListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL: softwareTitleSelectSQL,
		WhereSQL:  where,
		Args:      args,
		OrderKeys: map[string]dbutil.OrderExpr{
			"name":       {SQL: "lower(COALESCE(NULLIF(st.display_name, ''), st.name))"},
			"category":   {SQL: "lower(st.category)"},
			"developer":  {SQL: "lower(st.developer)"},
			"updated_at": {SQL: "st.updated_at"},
		},
		DefaultOrder: []dbutil.OrderExpr{
			{SQL: "lower(COALESCE(NULLIF(st.display_name, ''), st.name))"},
			{SQL: "lower(st.name)"},
			{SQL: "st.id"},
		},
		Params: params,
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
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[sqlc.MunkiSoftwareTitle])
	if err != nil {
		return nil, 0, err
	}
	titles := make([]SoftwareTitle, len(records))
	for i, row := range records {
		titles[i] = softwareTitleFromSQLC(row)
	}
	return titles, count, nil
}

func (s *Store) CreateArtifact(ctx context.Context, params ArtifactMutation) (*Artifact, error) {
	params = cleanArtifactMutation(params)
	if err := params.Validate(); err != nil {
		return nil, err
	}
	row, err := s.q.UpsertMunkiArtifact(ctx, sqlc.UpsertMunkiArtifactParams{
		Kind:        sqlc.MunkiArtifactKind(params.Kind),
		DisplayName: params.DisplayName,
		Location:    params.Location,
		ContentType: params.ContentType,
		SizeBytes:   params.SizeBytes,
		Sha256:      params.SHA256,
		StorageKey:  params.StorageKey,
	})
	if err != nil {
		return nil, mapDesiredMutationError(err)
	}
	artifact := artifactFromSQLC(row)
	return &artifact, nil
}

func (s *Store) ListArtifacts(ctx context.Context, params dbutil.ListParams) ([]Artifact, int, error) {
	params = dbutil.CleanListParams(params)
	count, err := s.q.CountMunkiArtifacts(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.q.ListMunkiArtifacts(ctx, sqlc.ListMunkiArtifactsParams{
		OffsetRows: int32(params.PageIndex * params.PageSize),
		LimitRows:  int32(params.PageSize),
	})
	if err != nil {
		return nil, 0, err
	}
	artifacts := make([]Artifact, len(rows))
	for i, row := range rows {
		artifacts[i] = artifactFromSQLC(row)
	}
	return artifacts, int(count), nil
}

func (s *Store) GetArtifact(ctx context.Context, id int64) (*Artifact, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.q.GetMunkiArtifactByID(ctx, sqlc.GetMunkiArtifactByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	artifact := artifactFromSQLC(row)
	return &artifact, nil
}

func (s *Store) GetArtifactByLocation(ctx context.Context, kind ArtifactKind, location string) (*Artifact, error) {
	if !validArtifactKind(kind) || !validArtifactLocation(location) {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.q.GetMunkiArtifactByKindAndLocation(ctx, sqlc.GetMunkiArtifactByKindAndLocationParams{
		Kind:     sqlc.MunkiArtifactKind(kind),
		Location: strings.TrimSpace(location),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	artifact := artifactFromSQLC(row)
	return &artifact, nil
}

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

func (s *Store) CreateDeployment(ctx context.Context, params DeploymentMutation) (*Deployment, error) {
	var err error
	params, err = s.normalizeDeploymentMutation(ctx, params)
	if err != nil {
		return nil, err
	}
	var row sqlc.MunkiDeployment
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		row, err = q.CreateMunkiDeployment(ctx, sqlc.CreateMunkiDeploymentParams{
			SoftwareID:       params.SoftwareID,
			Action:           sqlc.MunkiDeploymentAction(params.Action),
			OptionalInstall:  params.OptionalInstall,
			FeaturedItem:     params.FeaturedItem,
			PackageSelection: sqlc.MunkiPackageSelection(params.PackageSelection),
			PinnedPackageID:  params.PinnedPackageID,
			AllHosts:         params.AllHosts,
		})
		if err != nil {
			return err
		}
		return insertDeploymentScope(ctx, q, row.ID, params)
	})
	if err != nil {
		return nil, mapDesiredMutationError(err)
	}
	return s.GetDeployment(ctx, row.ID)
}

func (s *Store) UpdateDeployment(ctx context.Context, id int64, params DeploymentMutation) (*Deployment, error) {
	existing, err := s.GetDeployment(ctx, id)
	if err != nil {
		return nil, err
	}
	if params.SoftwareID != 0 && params.SoftwareID != existing.SoftwareID {
		return nil, fmt.Errorf("%w: software_id cannot be changed", dbutil.ErrInvalidInput)
	}
	params.SoftwareID = existing.SoftwareID
	params, err = s.normalizeDeploymentMutation(ctx, params)
	if err != nil {
		return nil, err
	}
	var row sqlc.MunkiDeployment
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		row, err = q.UpdateMunkiDeployment(ctx, sqlc.UpdateMunkiDeploymentParams{
			ID:               id,
			Action:           sqlc.MunkiDeploymentAction(params.Action),
			OptionalInstall:  params.OptionalInstall,
			FeaturedItem:     params.FeaturedItem,
			PackageSelection: sqlc.MunkiPackageSelection(params.PackageSelection),
			PinnedPackageID:  params.PinnedPackageID,
			AllHosts:         params.AllHosts,
		})
		if err != nil {
			return err
		}
		if err := deleteDeploymentScope(ctx, q, row.ID); err != nil {
			return err
		}
		return insertDeploymentScope(ctx, q, row.ID, params)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, mapDesiredMutationError(err)
	}
	return s.GetDeployment(ctx, row.ID)
}

func (s *Store) GetDeployment(ctx context.Context, id int64) (*Deployment, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.db.Pool().Query(ctx, deploymentSelectSQL+"\nWHERE d.id = $1", id)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(row, pgx.RowToStructByName[deploymentRecord])
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, dbutil.ErrNotFound
	}
	deployments := []Deployment{deploymentFromRecord(records[0])}
	if err := s.attachDeploymentScopes(ctx, deployments); err != nil {
		return nil, err
	}
	return &deployments[0], nil
}

func (s *Store) ListDeployments(ctx context.Context, params DeploymentListParams) ([]Deployment, int, error) {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	where, args := deploymentListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL:    deploymentSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    deploymentOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{{SQL: "d.position"}, {SQL: "d.id"}},
		Params:       params.ListParams,
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
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[deploymentRecord])
	if err != nil {
		return nil, 0, err
	}
	deployments := make([]Deployment, len(records))
	for i, row := range records {
		deployments[i] = deploymentFromRecord(row)
	}
	if err := s.attachDeploymentScopes(ctx, deployments); err != nil {
		return nil, 0, err
	}
	return deployments, count, nil
}

func (s *Store) ReorderDeployments(ctx context.Context, softwareID int64, orderedIDs []int64) error {
	if softwareID <= 0 {
		return fmt.Errorf("%w: software_id is required", dbutil.ErrInvalidInput)
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		currentIDs, err := q.ListMunkiDeploymentIDsBySoftware(
			ctx,
			sqlc.ListMunkiDeploymentIDsBySoftwareParams{SoftwareID: softwareID},
		)
		if err != nil {
			return err
		}
		if !dbutil.SameInt64Set(orderedIDs, currentIDs) {
			return fmt.Errorf("%w: ordered_ids must exactly match existing deployment IDs", dbutil.ErrInvalidInput)
		}
		if err := q.SetMunkiDeploymentPositions(ctx, sqlc.SetMunkiDeploymentPositionsParams{
			SoftwareID: softwareID,
			OrderedIds: orderedIDs,
		}); err != nil {
			return err
		}
		return q.NormalizeMunkiDeploymentPositions(
			ctx,
			sqlc.NormalizeMunkiDeploymentPositionsParams{SoftwareID: softwareID},
		)
	})
}

func (s *Store) EffectivePackagesForHost(ctx context.Context, hostID int64) ([]EffectivePackage, error) {
	rows, err := s.q.ListEffectiveMunkiPackagesForHost(ctx, sqlc.ListEffectiveMunkiPackagesForHostParams{
		HostID: hostID,
	})
	if err != nil {
		return nil, err
	}
	packages := make([]EffectivePackage, len(rows))
	for i, row := range rows {
		pkg, err := packageFromRecord(packageRecordFromEffectiveSQLC(row))
		if err != nil {
			return nil, err
		}
		packages[i] = EffectivePackage{
			DeploymentID:     row.DeploymentID,
			SoftwareID:       row.DeploymentSoftwareID,
			Action:           DeploymentAction(row.Action),
			OptionalInstall:  row.OptionalInstall,
			FeaturedItem:     row.FeaturedItem,
			PackageSelection: PackageSelection(row.PackageSelection),
			PinnedPackageID:  row.PinnedPackageID,
			Position:         row.Position,
			Package:          pkg,
			scopeRank:        int(row.ScopeRank),
		}
	}
	return resolveEffectivePackages(packages), nil
}

func (s *Store) UpsertHostStatus(ctx context.Context, status HostStatusObservation) error {
	return s.q.UpsertMunkiHostStatus(ctx, sqlc.UpsertMunkiHostStatusParams{
		HostID:          status.HostID,
		Version:         status.Version,
		ManifestName:    status.ManifestName,
		Success:         status.Success,
		Errors:          nonNilStrings(status.Errors),
		Warnings:        nonNilStrings(status.Warnings),
		ProblemInstalls: nonNilStrings(status.ProblemInstalls),
		RunStartedAt:    status.RunStartedAt,
		RunEndedAt:      status.RunEndedAt,
	})
}

func (s *Store) ClearHostStatus(ctx context.Context, hostID int64) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.DeleteMunkiHostItems(ctx, sqlc.DeleteMunkiHostItemsParams{HostID: hostID}); err != nil {
			return err
		}
		return q.ClearMunkiHostStatus(ctx, sqlc.ClearMunkiHostStatusParams{HostID: hostID})
	})
}

func (s *Store) ReplaceHostItems(ctx context.Context, hostID int64, items []HostItem) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.DeleteMunkiHostItems(ctx, sqlc.DeleteMunkiHostItemsParams{HostID: hostID}); err != nil {
			return err
		}
		for _, item := range items {
			if item.Name == "" {
				continue
			}
			if err := q.InsertMunkiHostItem(ctx, sqlc.InsertMunkiHostItemParams{
				HostID:           hostID,
				Name:             item.Name,
				Installed:        item.Installed,
				InstalledVersion: item.InstalledVersion,
				RunEndedAt:       item.RunEndedAt,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) LoadHostState(ctx context.Context, hostID int64) (*HostState, error) {
	status, err := s.q.GetMunkiHostStatus(ctx, sqlc.GetMunkiHostStatusParams{HostID: hostID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil //nolint:nilnil // missing Munki observation is represented by a nil state.
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListMunkiHostItems(ctx, sqlc.ListMunkiHostItemsParams{HostID: hostID})
	if err != nil {
		return nil, err
	}
	items := make([]HostItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, hostItemFromRecord(row))
	}
	return &HostState{
		Version:         status.Version,
		ManifestName:    status.ManifestName,
		Success:         status.Success,
		Errors:          nonNilStrings(status.Errors),
		Warnings:        nonNilStrings(status.Warnings),
		ProblemInstalls: nonNilStrings(status.ProblemInstalls),
		RunStartedAt:    status.RunStartedAt,
		RunEndedAt:      status.RunEndedAt,
		LastSeenAt:      status.LastSeenAt,
		Items:           items,
	}, nil
}

func (s *Store) normalizeSoftwareTitleIcon(
	ctx context.Context,
	params SoftwareTitleMutation,
) (SoftwareTitleMutation, error) {
	if params.IconArtifactID == nil {
		return params, nil
	}
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
	return params, nil
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

func hostItemFromRecord(row sqlc.MunkiHostItem) HostItem {
	return HostItem{
		HostID:           row.HostID,
		Name:             row.Name,
		Installed:        row.Installed,
		InstalledVersion: row.InstalledVersion,
		RunEndedAt:       row.RunEndedAt,
		LastSeenAt:       row.LastSeenAt,
	}
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func cleanSoftwareTitleMutation(params SoftwareTitleMutation) SoftwareTitleMutation {
	params.Name = strings.TrimSpace(params.Name)
	params.DisplayName = strings.TrimSpace(params.DisplayName)
	if params.DisplayName == "" {
		params.DisplayName = params.Name
	}
	params.Description = strings.TrimSpace(params.Description)
	params.Category = strings.TrimSpace(params.Category)
	params.Developer = strings.TrimSpace(params.Developer)
	params.IconName = strings.TrimSpace(params.IconName)
	params.IconHash = strings.TrimSpace(params.IconHash)
	return params
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
	if params.IconArtifactID == nil {
		params.IconArtifactID = existing.IconArtifactID
	}
	if params.IconName == "" {
		params.IconName = existing.IconName
	}
	if params.IconHash == "" {
		params.IconHash = existing.IconHash
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

func cleanArtifactMutation(params ArtifactMutation) ArtifactMutation {
	params.DisplayName = strings.TrimSpace(params.DisplayName)
	params.Location = strings.TrimSpace(params.Location)
	if params.DisplayName == "" {
		params.DisplayName = params.Location
	}
	params.ContentType = strings.TrimSpace(params.ContentType)
	params.SHA256 = strings.TrimSpace(params.SHA256)
	params.StorageKey = strings.TrimSpace(params.StorageKey)
	return params
}

func cleanDeploymentMutation(params DeploymentMutation) DeploymentMutation {
	params.Action = DeploymentAction(strings.TrimSpace(string(params.Action)))
	if params.Action == "" {
		params.Action = DeploymentActionInstall
	}
	params.PackageSelection = PackageSelection(strings.TrimSpace(string(params.PackageSelection)))
	if params.PackageSelection == "" {
		params.PackageSelection = PackageSelectionLatestEligible
	}
	params.IncludeLabelIDs = cleanInt64List(params.IncludeLabelIDs)
	params.ExcludeLabelIDs = cleanInt64List(params.ExcludeLabelIDs)
	params.IncludeHostIDs = cleanInt64List(params.IncludeHostIDs)
	params.ExcludeHostIDs = cleanInt64List(params.ExcludeHostIDs)
	return params
}

func (s *Store) normalizeDeploymentMutation(
	ctx context.Context,
	params DeploymentMutation,
) (DeploymentMutation, error) {
	params = cleanDeploymentMutation(params)
	if err := params.Validate(); err != nil {
		return params, err
	}
	if _, err := s.GetSoftwareTitle(ctx, params.SoftwareID); err != nil {
		return params, err
	}
	if params.PackageSelection != PackageSelectionSpecific {
		return params, nil
	}
	pkg, err := s.GetPackage(ctx, *params.PinnedPackageID)
	if err != nil {
		return params, err
	}
	if pkg.SoftwareID != params.SoftwareID {
		return params, fmt.Errorf(
			"%w: pinned_package_id must belong to software_id",
			dbutil.ErrInvalidInput,
		)
	}
	return params, nil
}

func cleanInt64List(values []int64) []int64 {
	out := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func softwareTitleFromSQLC(row sqlc.MunkiSoftwareTitle) SoftwareTitle {
	return SoftwareTitle{
		ID:             row.ID,
		Name:           row.Name,
		DisplayName:    row.DisplayName,
		Description:    row.Description,
		Category:       row.Category,
		Developer:      row.Developer,
		IconName:       row.IconName,
		IconHash:       row.IconHash,
		IconArtifactID: row.IconArtifactID,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func artifactFromSQLC(row sqlc.MunkiArtifact) Artifact {
	return Artifact{
		ID:          row.ID,
		Kind:        ArtifactKind(row.Kind),
		DisplayName: row.DisplayName,
		Location:    row.Location,
		ContentType: row.ContentType,
		SizeBytes:   row.SizeBytes,
		SHA256:      row.Sha256,
		StorageKey:  row.StorageKey,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
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

func deploymentFromRecord(row deploymentRecord) Deployment {
	return Deployment{
		ID:                   row.ID,
		SoftwareID:           row.SoftwareID,
		SoftwareDisplayName:  row.SoftwareDisplayName,
		Action:               DeploymentAction(row.Action),
		OptionalInstall:      row.OptionalInstall,
		FeaturedItem:         row.FeaturedItem,
		PackageSelection:     PackageSelection(row.PackageSelection),
		PinnedPackageID:      row.PinnedPackageID,
		PinnedPackageName:    stringPtrValue(row.PinnedPackageName),
		PinnedPackageVersion: stringPtrValue(row.PinnedPackageVersion),
		Position:             row.Position,
		AllHosts:             row.AllHosts,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func deleteDeploymentScope(ctx context.Context, q *sqlc.Queries, deploymentID int64) error {
	if err := q.DeleteMunkiDeploymentIncludeLabels(
		ctx,
		sqlc.DeleteMunkiDeploymentIncludeLabelsParams{DeploymentID: deploymentID},
	); err != nil {
		return err
	}
	if err := q.DeleteMunkiDeploymentExcludeLabels(
		ctx,
		sqlc.DeleteMunkiDeploymentExcludeLabelsParams{DeploymentID: deploymentID},
	); err != nil {
		return err
	}
	if err := q.DeleteMunkiDeploymentIncludeHosts(
		ctx,
		sqlc.DeleteMunkiDeploymentIncludeHostsParams{DeploymentID: deploymentID},
	); err != nil {
		return err
	}
	return q.DeleteMunkiDeploymentExcludeHosts(
		ctx,
		sqlc.DeleteMunkiDeploymentExcludeHostsParams{DeploymentID: deploymentID},
	)
}

func insertDeploymentScope(
	ctx context.Context,
	q *sqlc.Queries,
	deploymentID int64,
	params DeploymentMutation,
) error {
	if len(params.IncludeLabelIDs) > 0 {
		if err := q.InsertMunkiDeploymentIncludeLabels(ctx, sqlc.InsertMunkiDeploymentIncludeLabelsParams{
			DeploymentID: deploymentID,
			LabelIds:     params.IncludeLabelIDs,
		}); err != nil {
			return err
		}
	}
	if len(params.ExcludeLabelIDs) > 0 {
		if err := q.InsertMunkiDeploymentExcludeLabels(ctx, sqlc.InsertMunkiDeploymentExcludeLabelsParams{
			DeploymentID: deploymentID,
			LabelIds:     params.ExcludeLabelIDs,
		}); err != nil {
			return err
		}
	}
	if len(params.IncludeHostIDs) > 0 {
		if err := q.InsertMunkiDeploymentIncludeHosts(ctx, sqlc.InsertMunkiDeploymentIncludeHostsParams{
			DeploymentID: deploymentID,
			HostIds:      params.IncludeHostIDs,
		}); err != nil {
			return err
		}
	}
	if len(params.ExcludeHostIDs) > 0 {
		if err := q.InsertMunkiDeploymentExcludeHosts(ctx, sqlc.InsertMunkiDeploymentExcludeHostsParams{
			DeploymentID: deploymentID,
			HostIds:      params.ExcludeHostIDs,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) attachDeploymentScopes(ctx context.Context, deployments []Deployment) error {
	if len(deployments) == 0 {
		return nil
	}
	indexes := make(map[int64]int, len(deployments))
	ids := make([]int64, len(deployments))
	for i := range deployments {
		indexes[deployments[i].ID] = i
		ids[i] = deployments[i].ID
	}
	rows, err := s.q.ListMunkiDeploymentScopeIDs(
		ctx,
		sqlc.ListMunkiDeploymentScopeIDsParams{DeploymentIds: ids},
	)
	if err != nil {
		return err
	}
	for _, row := range rows {
		i, ok := indexes[row.DeploymentID]
		if !ok {
			continue
		}
		switch row.Scope {
		case "include_label":
			deployments[i].IncludeLabelIDs = append(deployments[i].IncludeLabelIDs, row.ID)
		case "exclude_label":
			deployments[i].ExcludeLabelIDs = append(deployments[i].ExcludeLabelIDs, row.ID)
		case "include_host":
			deployments[i].IncludeHostIDs = append(deployments[i].IncludeHostIDs, row.ID)
		case "exclude_host":
			deployments[i].ExcludeHostIDs = append(deployments[i].ExcludeHostIDs, row.ID)
		}
	}
	return nil
}

func mapDesiredMutationError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) || isForeignKeyViolation(err) {
		return dbutil.ErrNotFound
	}
	if dbutil.IsUniqueViolation(err) {
		return dbutil.ErrAlreadyExists
	}
	if dbutil.IsInvalidInputViolation(err) {
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return err
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func sqlcString[S ~string](value S) string {
	return string(value)
}

func softwareTitleListWhere(params dbutil.ListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			st.name ILIKE ` + search + `
			OR st.display_name ILIKE ` + search + `
			OR st.description ILIKE ` + search + `
			OR st.category ILIKE ` + search + `
			OR st.developer ILIKE ` + search + `
		)`)
	}
	return where.Build()
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

func deploymentListWhere(params DeploymentListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.SoftwareID > 0 {
		where.Add("d.software_id = " + where.Arg(params.SoftwareID))
	}
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			p.name ILIKE ` + search + `
			OR p.version ILIKE ` + search + `
			OR p.display_name ILIKE ` + search + `
			OR s.name ILIKE ` + search + `
			OR s.display_name ILIKE ` + search + `
			OR d.action::text ILIKE ` + search + `
			OR d.package_selection::text ILIKE ` + search + `
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

func deploymentOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"position":   {SQL: "d.position"},
		"name":       {SQL: "lower(COALESCE(NULLIF(s.display_name, ''), s.name))"},
		"action":     {SQL: "d.action"},
		"optional":   {SQL: "d.optional_install"},
		"featured":   {SQL: "d.featured_item"},
		"updated_at": {SQL: "d.updated_at"},
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

type deploymentRecord struct {
	ID                   int64
	SoftwareID           int64
	SoftwareDisplayName  string
	Action               sqlc.MunkiDeploymentAction
	OptionalInstall      bool
	FeaturedItem         bool
	PackageSelection     sqlc.MunkiPackageSelection
	PinnedPackageID      *int64
	PinnedPackageName    *string
	PinnedPackageVersion *string
	Position             int32
	AllHosts             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
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

const softwareTitleSelectSQL = `
SELECT
	st.id,
	st.name,
	st.display_name,
	st.description,
	st.category,
	st.developer,
	st.icon_name,
	st.icon_hash,
	st.icon_artifact_id,
	st.created_at,
	st.updated_at
FROM munki_software_titles st`

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

const deploymentSelectSQL = `
SELECT
	d.id,
	d.software_id,
	COALESCE(NULLIF(s.display_name, ''), s.name) AS software_display_name,
	d.action,
	d.optional_install,
	d.featured_item,
	d.package_selection,
	d.pinned_package_id,
	p.name AS pinned_package_name,
	p.version AS pinned_package_version,
	d.position,
	d.all_hosts,
	d.created_at,
	d.updated_at
FROM munki_deployments d
JOIN munki_software_titles s ON s.id = d.software_id
LEFT JOIN munki_packages p ON p.id = d.pinned_package_id`
