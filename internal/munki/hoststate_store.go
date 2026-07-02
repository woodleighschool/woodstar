package munki

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

func (s *Store) UpsertHostObservation(ctx context.Context, observation HostObservation) error {
	_, err := s.db.Pool().Exec(ctx, `
INSERT INTO munki_host_status (
	host_id,
	version,
	manifest_name,
	success,
	errors,
	warnings,
	problem_installs,
	run_started_at,
	run_ended_at,
	last_seen_at
)
VALUES (
	@host_id,
	@version,
	@manifest_name,
	@success,
	@errors,
	@warnings,
	@problem_installs,
	@run_started_at::timestamptz,
	@run_ended_at::timestamptz,
	now()
)
ON CONFLICT (host_id) DO UPDATE SET
	version = EXCLUDED.version,
	manifest_name = EXCLUDED.manifest_name,
	success = EXCLUDED.success,
	errors = EXCLUDED.errors,
	warnings = EXCLUDED.warnings,
	problem_installs = EXCLUDED.problem_installs,
	run_started_at = EXCLUDED.run_started_at,
	run_ended_at = EXCLUDED.run_ended_at,
	last_seen_at = now(),
	updated_at = now()`,
		pgx.NamedArgs{
			"host_id":          observation.HostID,
			"version":          observation.Version,
			"manifest_name":    observation.ManifestName,
			"success":          observation.Success,
			"errors":           dbutil.NonNilSlice(observation.Errors),
			"warnings":         dbutil.NonNilSlice(observation.Warnings),
			"problem_installs": dbutil.NonNilSlice(observation.ProblemInstalls),
			"run_started_at":   observation.RunStartedAt,
			"run_ended_at":     observation.RunEndedAt,
		})
	return err
}

func (s *Store) ClearHostObservation(ctx context.Context, hostID int64) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM munki_host_items WHERE host_id = $1`, hostID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM munki_host_status WHERE host_id = $1`, hostID)
		return err
	})
}

func (s *Store) ReplaceHostItems(ctx context.Context, hostID int64, items []Item) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM munki_host_items WHERE host_id = $1`, hostID); err != nil {
			return err
		}
		for _, item := range items {
			if _, err := tx.Exec(ctx, `
INSERT INTO munki_host_items (
	host_id,
	name,
	installed,
	installed_version,
	run_ended_at,
	last_seen_at
)
VALUES (
	@host_id,
	@name,
	@installed,
	@installed_version,
	@run_ended_at::timestamptz,
	now()
)
ON CONFLICT (host_id, name) DO UPDATE SET
	installed = EXCLUDED.installed,
	installed_version = EXCLUDED.installed_version,
	run_ended_at = EXCLUDED.run_ended_at,
	last_seen_at = now(),
	updated_at = now()`,
				pgx.NamedArgs{
					"host_id":           hostID,
					"name":              item.Name,
					"installed":         item.Installed,
					"installed_version": item.InstalledVersion,
					"run_ended_at":      item.RunEndedAt,
				}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) LoadHostState(ctx context.Context, hostID int64) (*HostState, error) {
	type statusRow struct {
		Version         string     `db:"version"`
		ManifestName    string     `db:"manifest_name"`
		Success         bool       `db:"success"`
		Errors          []string   `db:"errors"`
		Warnings        []string   `db:"warnings"`
		ProblemInstalls []string   `db:"problem_installs"`
		RunStartedAt    *time.Time `db:"run_started_at"`
		RunEndedAt      *time.Time `db:"run_ended_at"`
		LastSeenAt      time.Time  `db:"last_seen_at"`
	}

	statusRows, err := s.db.Pool().Query(ctx, `
		SELECT version, manifest_name, success, errors, warnings, problem_installs,
		       run_started_at, run_ended_at, last_seen_at
		FROM munki_host_status
		WHERE host_id = $1`,
		hostID,
	)
	if err != nil {
		return nil, err
	}
	status, err := pgx.CollectExactlyOneRow(statusRows, pgx.RowToStructByName[statusRow])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	itemRows, err := s.db.Pool().Query(ctx, `
		SELECT host_id, name, installed, installed_version, run_ended_at, last_seen_at
		FROM munki_host_items
		WHERE host_id = $1
		ORDER BY lower(name), name`,
		hostID,
	)
	if err != nil {
		return nil, err
	}
	type itemRow struct {
		HostID           int64      `db:"host_id"`
		Name             string     `db:"name"`
		Installed        bool       `db:"installed"`
		InstalledVersion string     `db:"installed_version"`
		RunEndedAt       *time.Time `db:"run_ended_at"`
		LastSeenAt       time.Time  `db:"last_seen_at"`
	}
	scannedItems, err := pgx.CollectRows(itemRows, pgx.RowToStructByName[itemRow])
	if err != nil {
		return nil, err
	}
	items := make([]Item, len(scannedItems))
	for i, row := range scannedItems {
		items[i] = Item(row)
	}

	return &HostState{
		Version:         status.Version,
		ManifestName:    status.ManifestName,
		Success:         status.Success,
		Errors:          dbutil.NonNilSlice(status.Errors),
		Warnings:        dbutil.NonNilSlice(status.Warnings),
		ProblemInstalls: dbutil.NonNilSlice(status.ProblemInstalls),
		RunStartedAt:    status.RunStartedAt,
		RunEndedAt:      status.RunEndedAt,
		LastSeenAt:      status.LastSeenAt,
		Items:           items,
	}, nil
}
