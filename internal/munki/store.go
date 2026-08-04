package munki

import (
	"context"
	"errors"

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

// AgentVersions returns Munki versions keyed by host ID for the requested hosts.
func (s *Store) AgentVersions(ctx context.Context, hostIDs []int64) (map[int64]string, error) {
	versions := make(map[int64]string, len(hostIDs))
	if len(hostIDs) == 0 {
		return versions, nil
	}
	rows, err := s.db.Pool().Query(ctx, `
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

// ApplyEnvelope atomically records one complete or failed Munki collection attempt.
func (s *Store) ApplyEnvelope(ctx context.Context, result EnvelopeResult) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if !result.Complete {
			_, err := tx.Exec(ctx, `
INSERT INTO munki_host_status (host_id, last_attempt_at, collection_error)
VALUES ($1, $2, $3)
ON CONFLICT (host_id) DO UPDATE SET
	last_attempt_at = EXCLUDED.last_attempt_at,
	collection_error = EXCLUDED.collection_error`, result.HostID, result.AttemptedAt, result.CollectionError)
			return err
		}

		if _, err := tx.Exec(ctx, `
INSERT INTO munki_host_status (
	host_id, last_attempt_at, last_successful_at, collection_error, has_report,
	version, manifest_name, errors, warnings, problem_installs, run_started_at, run_ended_at
)
VALUES (
	@host_id, @attempted_at, @attempted_at, '', @has_report,
	@version, @manifest_name, @errors, @warnings, @problem_installs, @run_started_at::timestamptz, @run_ended_at::timestamptz
)
ON CONFLICT (host_id) DO UPDATE SET
	last_attempt_at = EXCLUDED.last_attempt_at,
	last_successful_at = EXCLUDED.last_successful_at,
	collection_error = EXCLUDED.collection_error,
	has_report = EXCLUDED.has_report,
	version = EXCLUDED.version,
	manifest_name = EXCLUDED.manifest_name,
	errors = EXCLUDED.errors,
	warnings = EXCLUDED.warnings,
	problem_installs = EXCLUDED.problem_installs,
	run_started_at = EXCLUDED.run_started_at,
	run_ended_at = EXCLUDED.run_ended_at`, pgx.NamedArgs{
			"host_id":          result.HostID,
			"attempted_at":     result.AttemptedAt,
			"has_report":       result.HasReport,
			"version":          result.Observation.Version,
			"manifest_name":    result.Observation.ManifestName,
			"errors":           dbutil.NonNilSlice(result.Observation.Errors),
			"warnings":         dbutil.NonNilSlice(result.Observation.Warnings),
			"problem_installs": dbutil.NonNilSlice(result.Observation.ProblemInstalls),
			"run_started_at":   result.Observation.RunStartedAt,
			"run_ended_at":     result.Observation.RunEndedAt,
		}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM munki_host_items WHERE host_id = $1`, result.HostID); err != nil {
			return err
		}
		for _, item := range result.Items {
			if _, err := tx.Exec(ctx, `
INSERT INTO munki_host_items (
	host_id, name, display_name, installed, installed_version, target_version
)
VALUES (@host_id, @name, @display_name, @installed, @installed_version, @target_version)`, pgx.NamedArgs{
				"host_id":           result.HostID,
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

// LoadHostState returns the latest Munki report state for a host.
func (s *Store) LoadHostState(ctx context.Context, hostID int64) (*HostState, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT ''::TEXT AS report_state,
		       mhs.last_attempt_at,
		       mhs.last_successful_at,
		       COALESCE(mhs.collection_error, '') AS collection_error,
		       COALESCE(mhs.has_report, FALSE) AS has_report,
		       COALESCE(mhs.version, '') AS version,
		       COALESCE(mhs.manifest_name, '') AS manifest_name,
		       COALESCE(mhs.errors, ARRAY[]::TEXT[]) AS errors,
		       COALESCE(mhs.warnings, ARRAY[]::TEXT[]) AS warnings,
		       COALESCE(mhs.problem_installs, ARRAY[]::TEXT[]) AS problem_installs,
		       mhs.run_started_at,
		       mhs.run_ended_at
		FROM munki_host_status mhs
		WHERE mhs.host_id = $1`,
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
	state.ReportState = reportState(state)
	return &state, nil
}

func reportState(state HostState) ReportState {
	if state.LastSuccessfulAt == nil {
		return ReportNeverCollected
	}
	if state.CollectionError != "" {
		return ReportCollectionFailed
	}
	if state.HasReport {
		return ReportCurrent
	}
	return ReportNoReport
}
