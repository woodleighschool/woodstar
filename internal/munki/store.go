package munki

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

func (s *Store) ListSoftwareTitles(ctx context.Context, params dbutil.ListParams) ([]SoftwareTitle, int, error) {
	params = dbutil.CleanListParams(params)
	count, err := s.q.CountMunkiSoftwareTitles(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.q.ListMunkiSoftwareTitles(ctx, sqlc.ListMunkiSoftwareTitlesParams{
		OffsetRows: int32(params.PageIndex * params.PageSize),
		LimitRows:  int32(params.PageSize),
	})
	if err != nil {
		return nil, 0, err
	}
	titles := make([]SoftwareTitle, len(rows))
	for i, row := range rows {
		titles[i] = softwareTitleFromSQLC(row)
	}
	return titles, int(count), nil
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

func (s *Store) CreateRelease(ctx context.Context, params ReleaseMutation) (*Release, error) {
	params = cleanReleaseMutation(params)
	if err := params.Validate(); err != nil {
		return nil, err
	}
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
	row, err := s.q.CreateMunkiRelease(ctx, sqlc.CreateMunkiReleaseParams{
		SoftwareID:          params.SoftwareID,
		Name:                params.Name,
		Version:             params.Version,
		DisplayName:         params.DisplayName,
		Pkginfo:             params.Pkginfo,
		InstallerArtifactID: params.InstallerArtifactID,
		Eligible:            params.Eligible,
	})
	if err != nil {
		return nil, mapDesiredMutationError(err)
	}
	release := releaseFromSQLC(row)
	return &release, nil
}

func (s *Store) ListReleases(ctx context.Context, params dbutil.ListParams) ([]Release, int, error) {
	params = dbutil.CleanListParams(params)
	count, err := s.q.CountMunkiReleases(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.q.ListMunkiReleases(ctx, sqlc.ListMunkiReleasesParams{
		OffsetRows: int32(params.PageIndex * params.PageSize),
		LimitRows:  int32(params.PageSize),
	})
	if err != nil {
		return nil, 0, err
	}
	releases := make([]Release, len(rows))
	for i, row := range rows {
		releases[i] = releaseFromSQLC(row)
	}
	return releases, int(count), nil
}

func (s *Store) CreateAssignment(ctx context.Context, params AssignmentMutation) (*Assignment, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	var row sqlc.MunkiAssignment
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		row, err = q.CreateMunkiAssignment(ctx, sqlc.CreateMunkiAssignmentParams{
			ReleaseID: params.ReleaseID,
			Intent:    sqlc.MunkiAssignmentIntent(params.Intent),
			AllHosts:  params.AllHosts,
		})
		if err != nil {
			return err
		}
		return insertAssignmentScope(ctx, q, row.ID, params)
	})
	if err != nil {
		return nil, mapDesiredMutationError(err)
	}
	assignment := assignmentFromSQLC(row)
	if err := s.attachAssignmentScope(ctx, &assignment); err != nil {
		return nil, err
	}
	return &assignment, nil
}

func (s *Store) ListAssignments(ctx context.Context, params dbutil.ListParams) ([]Assignment, int, error) {
	params = dbutil.CleanListParams(params)
	count, err := s.q.CountMunkiAssignments(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.q.ListMunkiAssignments(ctx, sqlc.ListMunkiAssignmentsParams{
		OffsetRows: int32(params.PageIndex * params.PageSize),
		LimitRows:  int32(params.PageSize),
	})
	if err != nil {
		return nil, 0, err
	}
	assignments := make([]Assignment, len(rows))
	for i, row := range rows {
		assignments[i] = assignmentFromSQLC(row)
		if err := s.attachAssignmentScope(ctx, &assignments[i]); err != nil {
			return nil, 0, err
		}
	}
	return assignments, int(count), nil
}

func (s *Store) EffectiveReleasesForHost(ctx context.Context, hostID int64) ([]EffectiveRelease, error) {
	rows, err := s.q.ListEffectiveMunkiReleasesForHost(ctx, sqlc.ListEffectiveMunkiReleasesForHostParams{
		HostID: hostID,
	})
	if err != nil {
		return nil, err
	}
	releases := make([]EffectiveRelease, len(rows))
	for i, row := range rows {
		releases[i] = EffectiveRelease{
			AssignmentID: row.AssignmentID,
			Intent:       AssignmentIntent(row.Intent),
			Release: Release{
				ID:                        row.ReleaseID,
				SoftwareID:                row.SoftwareID,
				Name:                      row.Name,
				Version:                   row.Version,
				DisplayName:               row.DisplayName,
				Pkginfo:                   row.Pkginfo,
				InstallerArtifactID:       row.InstallerArtifactID,
				InstallerArtifactLocation: stringPtrValue(row.InstallerArtifactLocation),
				Eligible:                  true,
			},
			scopeRank: int(row.ScopeRank),
		}
	}
	return resolveEffectiveReleases(releases), nil
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

func cleanReleaseMutation(params ReleaseMutation) ReleaseMutation {
	params.Name = strings.TrimSpace(params.Name)
	params.Version = strings.TrimSpace(params.Version)
	params.DisplayName = strings.TrimSpace(params.DisplayName)
	if params.DisplayName == "" {
		params.DisplayName = params.Name
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

func releaseFromSQLC(row sqlc.MunkiRelease) Release {
	return Release{
		ID:                  row.ID,
		SoftwareID:          row.SoftwareID,
		Name:                row.Name,
		Version:             row.Version,
		DisplayName:         row.DisplayName,
		Pkginfo:             row.Pkginfo,
		InstallerArtifactID: row.InstallerArtifactID,
		Eligible:            row.Eligible,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func assignmentFromSQLC(row sqlc.MunkiAssignment) Assignment {
	return Assignment{
		ID:        row.ID,
		ReleaseID: row.ReleaseID,
		Intent:    AssignmentIntent(row.Intent),
		AllHosts:  row.AllHosts,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func insertAssignmentScope(
	ctx context.Context,
	q *sqlc.Queries,
	assignmentID int64,
	params AssignmentMutation,
) error {
	if len(params.IncludeLabelIDs) > 0 {
		if err := q.InsertMunkiAssignmentIncludeLabels(ctx, sqlc.InsertMunkiAssignmentIncludeLabelsParams{
			AssignmentID: assignmentID,
			LabelIds:     params.IncludeLabelIDs,
		}); err != nil {
			return err
		}
	}
	if len(params.ExcludeLabelIDs) > 0 {
		if err := q.InsertMunkiAssignmentExcludeLabels(ctx, sqlc.InsertMunkiAssignmentExcludeLabelsParams{
			AssignmentID: assignmentID,
			LabelIds:     params.ExcludeLabelIDs,
		}); err != nil {
			return err
		}
	}
	if len(params.IncludeHostIDs) > 0 {
		if err := q.InsertMunkiAssignmentIncludeHosts(ctx, sqlc.InsertMunkiAssignmentIncludeHostsParams{
			AssignmentID: assignmentID,
			HostIds:      params.IncludeHostIDs,
		}); err != nil {
			return err
		}
	}
	if len(params.ExcludeHostIDs) > 0 {
		if err := q.InsertMunkiAssignmentExcludeHosts(ctx, sqlc.InsertMunkiAssignmentExcludeHostsParams{
			AssignmentID: assignmentID,
			HostIds:      params.ExcludeHostIDs,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) attachAssignmentScope(ctx context.Context, assignment *Assignment) error {
	var err error
	assignment.IncludeLabelIDs, err = s.q.ListMunkiAssignmentIncludeLabelIDs(
		ctx,
		sqlc.ListMunkiAssignmentIncludeLabelIDsParams{AssignmentID: assignment.ID},
	)
	if err != nil {
		return err
	}
	assignment.ExcludeLabelIDs, err = s.q.ListMunkiAssignmentExcludeLabelIDs(
		ctx,
		sqlc.ListMunkiAssignmentExcludeLabelIDsParams{AssignmentID: assignment.ID},
	)
	if err != nil {
		return err
	}
	assignment.IncludeHostIDs, err = s.q.ListMunkiAssignmentIncludeHostIDs(
		ctx,
		sqlc.ListMunkiAssignmentIncludeHostIDsParams{AssignmentID: assignment.ID},
	)
	if err != nil {
		return err
	}
	assignment.ExcludeHostIDs, err = s.q.ListMunkiAssignmentExcludeHostIDs(
		ctx,
		sqlc.ListMunkiAssignmentExcludeHostIDsParams{AssignmentID: assignment.ID},
	)
	return err
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
