package artifacts

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

func (s *Store) Create(ctx context.Context, params ArtifactMutation) (*Artifact, error) {
	params = cleanMutation(params)
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
		return nil, dbutil.MutationError(err)
	}
	artifact := artifactFromSQLC(row)
	return &artifact, nil
}

func (s *Store) List(ctx context.Context, params dbutil.ListParams) ([]Artifact, int, error) {
	params = dbutil.CleanListParams(params)
	count, err := s.q.CountMunkiArtifacts(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.q.ListMunkiArtifacts(ctx, sqlc.ListMunkiArtifactsParams{
		OffsetRows: params.PageIndex * params.PageSize,
		LimitRows:  params.PageSize,
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

func (s *Store) GetByID(ctx context.Context, id int64) (*Artifact, error) {
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

func (s *Store) GetByLocation(ctx context.Context, kind ArtifactKind, location string) (*Artifact, error) {
	if !ValidArtifactKind(kind) || !ValidArtifactLocation(location) {
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

func (s *Store) Delete(ctx context.Context, id int64) error {
	rows, err := s.q.DeleteMunkiArtifact(ctx, sqlc.DeleteMunkiArtifactParams{ID: id})
	if err != nil {
		return dbutil.DeleteConflict(err, "Munki artifact is still referenced")
	}
	if rows == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

func cleanMutation(params ArtifactMutation) ArtifactMutation {
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
