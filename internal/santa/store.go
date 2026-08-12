package santa

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/postgres"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
)

// Store persists Santa state.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// AgentVersions returns Santa versions keyed by host ID for the requested hosts.
func (s *Store) AgentVersions(ctx context.Context, hostIDs []int64) (map[int64]string, error) {
	versions := make(map[int64]string, len(hostIDs))
	if len(hostIDs) == 0 {
		return versions, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT host_id, santa_version
FROM santa_hosts
WHERE host_id = ANY($1::bigint[])`, hostIDs)
	if err != nil {
		return nil, err
	}
	type agentVersionRow struct {
		HostID  int64  `db:"host_id"`
		Version string `db:"santa_version"`
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
	if observation.ClientModeReported == "" {
		observation.ClientModeReported = configurations.ReportedClientModeUnknown
	}
	if observation.PrimaryUserGroups == nil {
		observation.PrimaryUserGroups = []string{}
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO santa_hosts (
			host_id,
			machine_id,
			serial_number,
			santa_version,
			client_mode_reported,
			primary_user,
			primary_user_groups,
			sip_status
		)
		VALUES (
			@host_id,
			@machine_id,
			@serial_number,
			@santa_version,
			@client_mode_reported::santa_client_mode,
			@primary_user,
			@primary_user_groups,
			@sip_status
		)
		ON CONFLICT (host_id) DO UPDATE SET
			machine_id = EXCLUDED.machine_id,
			serial_number = EXCLUDED.serial_number,
			santa_version = EXCLUDED.santa_version,
			client_mode_reported = EXCLUDED.client_mode_reported,
			primary_user = EXCLUDED.primary_user,
			primary_user_groups = EXCLUDED.primary_user_groups,
			sip_status = EXCLUDED.sip_status,
			updated_at = now()`, pgx.NamedArgs{
		"host_id":              observation.HostID,
		"machine_id":           observation.MachineID,
		"serial_number":        observation.SerialNumber,
		"santa_version":        observation.Version,
		"client_mode_reported": string(observation.ClientModeReported),
		"primary_user":         observation.PrimaryUser,
		"primary_user_groups":  observation.PrimaryUserGroups,
		"sip_status":           observation.SIPStatus,
	})
	return err
}

// Santa's default MachineID is the hardware UUID reported by Orbit or osquery.
func (s *Store) hostIDByMachineID(ctx context.Context, machineID string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM hosts WHERE hardware_uuid = $1`, machineID).Scan(&id)
	return id, postgres.GetError(err)
}

type observedSantaHostStateRow struct {
	SantaVersion       string     `db:"santa_version"`
	ClientModeReported string     `db:"client_mode_reported"`
	LastCleanSyncAt    *time.Time `db:"last_clean_sync_at"`
}

func (s *Store) LoadObservedHostState(ctx context.Context, hostID int64) (*HostState, error) {
	row, err := postgres.GetOne[observedSantaHostStateRow](ctx, s.pool, `
		SELECT
			sh.santa_version,
			sh.client_mode_reported::text AS client_mode_reported,
			ss.last_clean_sync_at
		FROM santa_hosts sh
		LEFT JOIN santa_sync_state ss ON ss.host_id = sh.host_id
		WHERE sh.host_id = $1`, hostID)
	if errors.Is(err, fault.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	detail := HostState{
		Version:            row.SantaVersion,
		ClientModeReported: configurations.ReportedClientMode(row.ClientModeReported),
	}

	ruleSync, err := s.syncSummary(ctx, hostID)
	if err != nil {
		return nil, err
	}
	ruleSync.LastCleanSyncAt = row.LastCleanSyncAt

	detail.RuleSync = ruleSync
	return &detail, nil
}

func (s *Store) syncSummary(ctx context.Context, hostID int64) (RuleSyncSummary, error) {
	var desired, applied int32
	err := s.pool.QueryRow(ctx, `
		SELECT
			(
				SELECT count(*)::integer
				FROM santa_sync_targets st
				WHERE st.host_id = $1 AND st.phase = 'desired'
			) AS desired_count,
			(
				SELECT count(*)::integer
				FROM santa_sync_targets st
				WHERE st.host_id = $1 AND st.phase = 'applied'
			) AS applied_count`, hostID).Scan(&desired, &applied)
	return RuleSyncSummary{
		DesiredCount: desired,
		AppliedCount: applied,
	}, err
}
