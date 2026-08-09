package munki

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// AgentVersions returns Munki versions keyed by host ID for the requested hosts.
func (s *Store) AgentVersions(ctx context.Context, hostIDs []int64) (map[int64]string, error) {
	versions := make(map[int64]string, len(hostIDs))
	if len(hostIDs) == 0 {
		return versions, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT host_id, version
FROM munki_host_status
WHERE host_id = ANY($1::bigint[])`, hostIDs)
	if err != nil {
		return nil, err
	}
	type agentVersionRow struct {
		HostID  int64  `db:"host_id"`
		Version string `db:"version"`
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[agentVersionRow])
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		versions[record.HostID] = record.Version
	}
	return versions, nil
}

func (s *Store) UpsertHostObservation(ctx context.Context, observation HostObservation) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO munki_host_status (
	host_id,
	version,
	manifest_name,
	errors,
	warnings,
	problem_installs,
	run_started_at,
	run_ended_at
)
VALUES (
	@host_id,
	@version,
	@manifest_name,
	@errors,
	@warnings,
	@problem_installs,
	@run_started_at::timestamptz,
	@run_ended_at::timestamptz
)
ON CONFLICT (host_id) DO UPDATE SET
	version = EXCLUDED.version,
	manifest_name = EXCLUDED.manifest_name,
	errors = EXCLUDED.errors,
	warnings = EXCLUDED.warnings,
	problem_installs = EXCLUDED.problem_installs,
	run_started_at = EXCLUDED.run_started_at,
	run_ended_at = EXCLUDED.run_ended_at`,
		pgx.NamedArgs{
			"host_id":          observation.HostID,
			"version":          observation.Version,
			"manifest_name":    observation.ManifestName,
			"errors":           dbutil.NonNilSlice(observation.Errors),
			"warnings":         dbutil.NonNilSlice(observation.Warnings),
			"problem_installs": dbutil.NonNilSlice(observation.ProblemInstalls),
			"run_started_at":   observation.RunStartedAt,
			"run_ended_at":     observation.RunEndedAt,
		})
	return err
}

func (s *Store) ClearHostObservation(ctx context.Context, hostID int64) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM munki_host_items WHERE host_id = $1`, hostID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM munki_host_status WHERE host_id = $1`, hostID)
		return err
	})
}

func (s *Store) ReplaceHostItems(ctx context.Context, hostID int64, items []ItemObservation) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM munki_host_items WHERE host_id = $1`, hostID); err != nil {
			return err
		}
		for _, item := range items {
			if _, err := tx.Exec(ctx, `
INSERT INTO munki_host_items (
	host_id,
	name,
	display_name,
	installed,
	installed_version,
	target_version
)
VALUES (
	@host_id,
	@name,
	@display_name,
	@installed,
	@installed_version,
	@target_version
)
ON CONFLICT (host_id, name) DO UPDATE SET
	display_name = EXCLUDED.display_name,
	installed = EXCLUDED.installed,
	installed_version = EXCLUDED.installed_version,
	target_version = EXCLUDED.target_version`,
				pgx.NamedArgs{
					"host_id":           hostID,
					"name":              item.Name,
					"display_name":      item.DisplayName,
					"installed":         item.Installed,
					"installed_version": item.InstalledVersion,
					"target_version":    item.TargetVersion,
				}); err != nil {
				return err
			}
		}
		return nil
	})
}

// LoadHostState returns the latest Munki run summary reported for a host.
func (s *Store) LoadHostState(ctx context.Context, hostID int64) (*HostState, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT version, manifest_name, errors, warnings, problem_installs,
		       run_started_at, run_ended_at
		FROM munki_host_status
		WHERE host_id = $1`,
		hostID,
	)
	if err != nil {
		return nil, err
	}
	state, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[HostState])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	state.Errors = dbutil.NonNilSlice(state.Errors)
	state.Warnings = dbutil.NonNilSlice(state.Warnings)
	state.ProblemInstalls = dbutil.NonNilSlice(state.ProblemInstalls)
	return &state, nil
}
