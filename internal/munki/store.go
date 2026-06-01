package munki

import (
	"context"
	"encoding/json"
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
	params = cleanSoftwareTitleMutation(params)
	if err := params.Validate(); err != nil {
		return nil, err
	}
	row, err := s.q.CreateMunkiSoftwareTitle(ctx, sqlc.CreateMunkiSoftwareTitleParams{
		Name:        params.Name,
		DisplayName: params.DisplayName,
		Description: params.Description,
		Category:    params.Category,
		Developer:   params.Developer,
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
	params = cleanSoftwareTitleMutation(params)
	if err := params.Validate(); err != nil {
		return nil, err
	}
	row, err := s.q.UpdateMunkiSoftwareTitle(ctx, sqlc.UpdateMunkiSoftwareTitleParams{
		Name:        params.Name,
		DisplayName: params.DisplayName,
		Description: params.Description,
		Category:    params.Category,
		Developer:   params.Developer,
		ID:          id,
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
	row, err := s.q.CreateMunkiArtifact(ctx, sqlc.CreateMunkiArtifactParams{
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
	if params.InstallerArtifactID != nil {
		artifact, err := s.GetArtifact(ctx, *params.InstallerArtifactID)
		if err != nil {
			return nil, err
		}
		if artifact.Kind != ArtifactKindPackage {
			return nil, fmt.Errorf(
				"%w: installer_artifact_id must reference a package artifact",
				dbutil.ErrInvalidInput,
			)
		}
	}
	metadata, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, err
	}
	row, err := s.q.CreateMunkiPackage(ctx, sqlc.CreateMunkiPackageParams{
		SoftwareID:          params.SoftwareID,
		Name:                params.Name,
		Version:             params.Version,
		DisplayName:         params.DisplayName,
		Description:         params.Description,
		Category:            params.Category,
		Developer:           params.Developer,
		Metadata:            metadata,
		InstallerArtifactID: params.InstallerArtifactID,
		Eligible:            params.Eligible,
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
	if err := params.Validate(); err != nil {
		return nil, err
	}
	var row sqlc.MunkiDeployment
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		row, err = q.CreateMunkiDeployment(ctx, sqlc.CreateMunkiDeploymentParams{
			PackageID: params.PackageID,
			Intent:    sqlc.MunkiDeploymentIntent(params.Intent),
			AllHosts:  params.AllHosts,
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
			DeploymentID: row.DeploymentID,
			Intent:       DeploymentIntent(row.Intent),
			Position:     row.Position,
			Package:      pkg,
			scopeRank:    int(row.ScopeRank),
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
	params.Metadata = cleanPackageMetadata(params.Metadata)
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

func cleanPackageMetadata(metadata PackageMetadata) PackageMetadata {
	metadata.InstallerType = strings.TrimSpace(metadata.InstallerType)
	metadata.UninstallMethod = strings.TrimSpace(metadata.UninstallMethod)
	metadata.RestartAction = strings.TrimSpace(metadata.RestartAction)
	metadata.MinimumMunkiVersion = strings.TrimSpace(metadata.MinimumMunkiVersion)
	metadata.MinimumOSVersion = strings.TrimSpace(metadata.MinimumOSVersion)
	metadata.MaximumOSVersion = strings.TrimSpace(metadata.MaximumOSVersion)
	metadata.SupportedArchitectures = cleanStringList(metadata.SupportedArchitectures)
	metadata.BlockingApplications = cleanStringList(metadata.BlockingApplications)
	metadata.Requires = cleanStringList(metadata.Requires)
	metadata.UpdateFor = cleanStringList(metadata.UpdateFor)
	return metadata
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

func softwareTitleFromSQLC(row sqlc.MunkiSoftwareTitle) SoftwareTitle {
	return SoftwareTitle{
		ID:          row.ID,
		Name:        row.Name,
		DisplayName: row.DisplayName,
		Description: row.Description,
		Category:    row.Category,
		Developer:   row.Developer,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
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
	var metadata PackageMetadata
	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
			return Package{}, fmt.Errorf("decode munki package metadata %d: %w", row.ID, err)
		}
	}
	metadata = cleanPackageMetadata(metadata)
	pkg := Package{
		ID:                        row.ID,
		SoftwareID:                row.SoftwareID,
		SoftwareName:              row.SoftwareName,
		SoftwareDisplayName:       row.SoftwareDisplayName,
		Name:                      row.Name,
		Version:                   row.Version,
		DisplayName:               row.DisplayName,
		Description:               row.Description,
		Category:                  row.Category,
		Developer:                 row.Developer,
		Metadata:                  metadata,
		InstallerArtifactID:       row.InstallerArtifactID,
		InstallerArtifactLocation: stringPtrValue(row.InstallerArtifactLocation),
		Eligible:                  row.Eligible,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
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
		ID:                  row.ID,
		PackageID:           row.PackageID,
		PackageName:         row.PackageName,
		PackageVersion:      row.PackageVersion,
		SoftwareID:          row.SoftwareID,
		SoftwareDisplayName: row.SoftwareDisplayName,
		Intent:              DeploymentIntent(row.Intent),
		Position:            row.Position,
		AllHosts:            row.AllHosts,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
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
			OR s.name ILIKE ` + search + `
			OR s.display_name ILIKE ` + search + `
		)`)
	}
	return where.Build()
}

func deploymentListWhere(params DeploymentListParams) (string, []any) {
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
			OR s.name ILIKE ` + search + `
			OR s.display_name ILIKE ` + search + `
			OR d.intent::text ILIKE ` + search + `
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
		"name":       {SQL: "lower(COALESCE(NULLIF(p.display_name, ''), p.name))"},
		"intent":     {SQL: "d.intent"},
		"updated_at": {SQL: "d.updated_at"},
	}
}

type packageRecord struct {
	ID                        int64
	SoftwareID                int64
	SoftwareName              string
	SoftwareDisplayName       string
	Name                      string
	Version                   string
	DisplayName               string
	Description               string
	Category                  string
	Developer                 string
	Metadata                  []byte
	InstallerArtifactID       *int64
	InstallerArtifactLocation *string
	Eligible                  bool
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type deploymentRecord struct {
	ID                  int64
	PackageID           int64
	PackageName         string
	PackageVersion      string
	SoftwareID          int64
	SoftwareDisplayName string
	Intent              sqlc.MunkiDeploymentIntent
	Position            int32
	AllHosts            bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func packageRecordFromSQLC(row sqlc.GetMunkiPackageByIDRow) packageRecord {
	return packageRecord{
		ID:                        row.ID,
		SoftwareID:                row.SoftwareID,
		SoftwareName:              row.SoftwareName,
		SoftwareDisplayName:       row.SoftwareDisplayName,
		Name:                      row.Name,
		Version:                   row.Version,
		DisplayName:               row.DisplayName,
		Description:               row.Description,
		Category:                  row.Category,
		Developer:                 row.Developer,
		Metadata:                  row.Metadata,
		InstallerArtifactID:       row.InstallerArtifactID,
		InstallerArtifactLocation: row.InstallerArtifactLocation,
		Eligible:                  row.Eligible,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
	}
}

func packageRecordFromEffectiveSQLC(row sqlc.ListEffectiveMunkiPackagesForHostRow) packageRecord {
	return packageRecord{
		ID:                        row.PackageID,
		SoftwareID:                row.SoftwareID,
		Name:                      row.Name,
		Version:                   row.Version,
		DisplayName:               row.DisplayName,
		Description:               row.Description,
		Category:                  row.Category,
		Developer:                 row.Developer,
		Metadata:                  row.Metadata,
		InstallerArtifactID:       row.InstallerArtifactID,
		InstallerArtifactLocation: row.InstallerArtifactLocation,
		Eligible:                  true,
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
	st.created_at,
	st.updated_at
FROM munki_software_titles st`

const packageSelectSQL = `
SELECT
	p.id,
	p.software_id,
	s.name AS software_name,
	COALESCE(NULLIF(s.display_name, ''), s.name) AS software_display_name,
	p.name,
	p.version,
	p.display_name,
	p.description,
	p.category,
	p.developer,
	p.metadata,
	p.installer_artifact_id,
	art.location AS installer_artifact_location,
	p.eligible,
	p.created_at,
	p.updated_at
FROM munki_packages p
JOIN munki_software_titles s ON s.id = p.software_id
LEFT JOIN munki_artifacts art ON art.id = p.installer_artifact_id`

const deploymentSelectSQL = `
SELECT
	d.id,
	d.package_id,
	p.name AS package_name,
	p.version AS package_version,
	p.software_id,
	COALESCE(NULLIF(s.display_name, ''), s.name) AS software_display_name,
	d.intent,
	d.position,
	d.all_hosts,
	d.created_at,
	d.updated_at
FROM munki_deployments d
JOIN munki_packages p ON p.id = d.package_id
JOIN munki_software_titles s ON s.id = p.software_id`
