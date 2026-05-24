package santa

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
)

// Store persists Santa state.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

func (s *Store) UpsertHostObservation(ctx context.Context, observation HostObservation) error {
	observation.MachineID = strings.TrimSpace(observation.MachineID)
	observation.SerialNumber = strings.TrimSpace(observation.SerialNumber)
	if observation.HostID <= 0 || observation.MachineID == "" || observation.SerialNumber == "" {
		return dbutil.ErrInvalidInput
	}
	if observation.ClientModeReported == "" {
		observation.ClientModeReported = configurations.ClientModeUnknown
	}
	if observation.PrimaryUserGroups == nil {
		observation.PrimaryUserGroups = []string{}
	}

	return s.q.UpsertSantaHostObservation(ctx, sqlc.UpsertSantaHostObservationParams{
		HostID:             observation.HostID,
		MachineID:          observation.MachineID,
		SerialNumber:       observation.SerialNumber,
		SantaVersion:       observation.Version,
		ClientModeReported: sqlc.SantaClientMode(observation.ClientModeReported),
		PrimaryUser:        observation.PrimaryUser,
		PrimaryUserGroups:  observation.PrimaryUserGroups,
		SipStatus:          observation.SIPStatus,
		OSBuild:            observation.OSBuild,
		ModelIdentifier:    observation.ModelIdentifier,
		LastSeenAt:         observation.LastSeenAt,
	})
}

func (s *Store) LoadObservedHostState(ctx context.Context, hostID int64) (*HostState, error) {
	var detail HostState
	var clientMode string
	err := s.db.Pool().QueryRow(ctx, `
		SELECT
			sh.santa_version,
			sh.client_mode_reported::text,
			sh.last_seen_at,
			ss.last_clean_sync_at
		FROM santa_hosts sh
		LEFT JOIN santa_sync_state ss ON ss.host_id = sh.host_id
		WHERE sh.host_id = $1
	`, hostID).Scan(
		&detail.Version,
		&clientMode,
		&detail.LastSyncAt,
		&detail.RuleSync.LastCleanSyncAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil //nolint:nilnil // missing Santa observation is represented by a nil state.
	}
	if err != nil {
		return nil, err
	}

	ruleSync, err := s.syncSummary(ctx, hostID)
	if err != nil {
		return nil, err
	}
	ruleSync.LastCleanSyncAt = detail.RuleSync.LastCleanSyncAt

	detail.ClientModeReported = configurations.ClientMode(clientMode)
	detail.RuleSync = ruleSync
	return &detail, nil
}

func (s *Store) syncSummary(ctx context.Context, hostID int64) (RuleSyncSummary, error) {
	var summary RuleSyncSummary
	err := s.db.Pool().QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM santa_sync_targets WHERE host_id = $1 AND phase = 'desired'),
			(SELECT count(*) FROM santa_sync_targets WHERE host_id = $1 AND phase = 'applied'),
			(SELECT count(*) FROM santa_sync_pending_rules WHERE host_id = $1)
	`, hostID).Scan(&summary.DesiredCount, &summary.AppliedCount, &summary.PendingCount)
	return summary, err
}
