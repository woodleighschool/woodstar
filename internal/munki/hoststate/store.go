package hoststate

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

func (s *Store) UpsertHostStatus(ctx context.Context, status Observation) error {
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

func (s *Store) ReplaceHostItems(ctx context.Context, hostID int64, items []Item) error {
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

func (s *Store) LoadHostState(ctx context.Context, hostID int64) (*State, error) {
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
	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, hostItemFromRecord(row))
	}
	return &State{
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

func hostItemFromRecord(row sqlc.MunkiHostItem) Item {
	return Item{
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
