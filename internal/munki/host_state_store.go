package munki

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
)

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
