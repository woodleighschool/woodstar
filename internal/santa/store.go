package santa

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
)

// Store persists Santa state.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

func (s *Store) UpsertHostObservation(ctx context.Context, observation HostObservation) error {
	if observation.ClientModeReported == "" {
		observation.ClientModeReported = configurations.ReportedClientModeUnknown
	}
	if observation.PrimaryUserGroups == nil {
		observation.PrimaryUserGroups = []string{}
	}

	_, err := s.db.Pool().Exec(ctx, `
		INSERT INTO santa_hosts (
			host_id,
			machine_id,
			serial_number,
			santa_version,
			client_mode_reported,
			primary_user,
			primary_user_groups,
			sip_status,
			last_seen_at
		)
		VALUES (
			@host_id,
			@machine_id,
			@serial_number,
			@santa_version,
			@client_mode_reported::santa_client_mode,
			@primary_user,
			@primary_user_groups,
			@sip_status,
			COALESCE(@last_seen_at::timestamptz, now())
		)
		ON CONFLICT (host_id) DO UPDATE SET
			machine_id = EXCLUDED.machine_id,
			serial_number = EXCLUDED.serial_number,
			santa_version = EXCLUDED.santa_version,
			client_mode_reported = EXCLUDED.client_mode_reported,
			primary_user = EXCLUDED.primary_user,
			primary_user_groups = EXCLUDED.primary_user_groups,
			sip_status = EXCLUDED.sip_status,
			last_seen_at = EXCLUDED.last_seen_at,
			updated_at = now()`, pgx.NamedArgs{
		"host_id":              observation.HostID,
		"machine_id":           observation.MachineID,
		"serial_number":        observation.SerialNumber,
		"santa_version":        observation.Version,
		"client_mode_reported": string(observation.ClientModeReported),
		"primary_user":         observation.PrimaryUser,
		"primary_user_groups":  observation.PrimaryUserGroups,
		"sip_status":           observation.SIPStatus,
		"last_seen_at":         observation.LastSeenAt,
	})
	return err
}

// Santa's default MachineID is the hardware UUID reported by Orbit or osquery.
func (s *Store) hostIDByMachineID(ctx context.Context, machineID string) (int64, error) {
	var id int64
	err := s.db.Pool().QueryRow(ctx, `SELECT id FROM hosts WHERE hardware_uuid = $1`, machineID).Scan(&id)
	return id, dbutil.GetError(err)
}

type observedSantaHostStateRow struct {
	SantaVersion       string     `db:"santa_version"`
	ClientModeReported string     `db:"client_mode_reported"`
	LastSeenAt         *time.Time `db:"last_seen_at"`
	LastCleanSyncAt    *time.Time `db:"last_clean_sync_at"`
}

func (s *Store) LoadObservedHostState(ctx context.Context, hostID int64) (*HostState, error) {
	row, err := dbutil.GetOne[observedSantaHostStateRow](ctx, s.db.Pool(), `
		SELECT
			sh.santa_version,
			sh.client_mode_reported::text AS client_mode_reported,
			sh.last_seen_at,
			ss.last_clean_sync_at
		FROM santa_hosts sh
		LEFT JOIN santa_sync_state ss ON ss.host_id = sh.host_id
		WHERE sh.host_id = $1`, hostID)
	if errors.Is(err, dbutil.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	detail := HostState{
		Version:            row.SantaVersion,
		LastSeenAt:         row.LastSeenAt,
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
	var desired, applied, pending int32
	err := s.db.Pool().QueryRow(ctx, `
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
			) AS applied_count,
			COALESCE(
				(SELECT ss.pending_payload_rule_count FROM santa_sync_state ss WHERE ss.host_id = $1),
				0
			)::integer AS pending_count`, hostID).Scan(&desired, &applied, &pending)
	return RuleSyncSummary{
		DesiredCount: desired,
		AppliedCount: applied,
		PendingCount: pending,
	}, err
}
