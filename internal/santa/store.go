package santa

import (
	"context"
	"errors"

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
	if observation.ClientModeReported == "" {
		observation.ClientModeReported = configurations.ReportedClientModeUnknown
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

// Santa machine_id comes from Woodstar's enrollment profile and must match the
// canonical host hardware UUID. Custom Santa MachineID profiles are not enrolled.
func (s *Store) hostIDByMachineID(ctx context.Context, machineID string) (int64, error) {
	hostID, err := s.q.GetHostIDByMachineID(ctx, sqlc.GetHostIDByMachineIDParams{MachineID: machineID})
	return hostID, dbutil.GetError(err)
}

func (s *Store) LoadObservedHostState(ctx context.Context, hostID int64) (*HostState, error) {
	row, err := s.q.GetObservedSantaHostState(ctx, sqlc.GetObservedSantaHostStateParams{HostID: hostID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	detail := HostState{
		Version:            row.SantaVersion,
		LastSyncAt:         row.LastSeenAt,
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
	row, err := s.q.GetSantaSyncSummary(ctx, sqlc.GetSantaSyncSummaryParams{SummaryHostID: hostID})
	return RuleSyncSummary{
		DesiredCount: row.DesiredCount,
		AppliedCount: row.AppliedCount,
		PendingCount: row.PendingCount,
	}, err
}
