package syncstate

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/woodleighschool/woodstar/internal/fault"
)

func (s *Store) PromotePending(
	ctx context.Context,
	hostID int64,
	rulesReceived uint32,
	rulesProcessed uint32,
	syncType SyncType,
	rulesHash string,
) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var state santaPendingStateRow
		err := tx.QueryRow(ctx, `
SELECT
    COALESCE(pending_sync_type::text, ''),
    COALESCE(pending_policy_digest, ''),
    COALESCE(preflight_rules_hash, '')
FROM santa_sync_state
WHERE host_id = $1
FOR UPDATE`, hostID).Scan(
			&state.PendingSyncType,
			&state.PendingPolicyDigest,
			&state.PreflightRulesHash,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: no pending Santa sync", fault.ErrInvalidInput)
		}
		if err != nil {
			return err
		}
		desired, err := loadTargets(ctx, tx, hostID, phaseDesired)
		if err != nil {
			return err
		}
		applied, err := loadTargets(ctx, tx, hostID, phaseApplied)
		if err != nil {
			return err
		}
		changes := normalSyncPayload(desired, applied)
		payloadCount := len(changes)
		if state.PendingSyncType != SyncTypeNormal {
			// Empty clean syncs deliver the tombstone rule.
			payloadCount = max(1, len(desired))
		}
		if err := validatePostflight(
			state,
			uint64(payloadCount),
			rulesReceived,
			rulesProcessed,
			syncType,
			rulesHash,
		); err != nil {
			return err
		}
		if err := applyTargetChanges(ctx, tx, hostID, phaseApplied, changes); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
UPDATE santa_sync_state
SET
    pending_sync_type = NULL,
    pending_policy_digest = NULL,
    preflight_rules_hash = NULL,
    applied_policy_digest = $2,
    confirmed_rules_hash = $3,
    last_clean_sync_at = CASE WHEN $4::boolean THEN now() ELSE last_clean_sync_at END,
    updated_at = now()
WHERE host_id = $1`,
			hostID,
			state.PendingPolicyDigest,
			rulesHash,
			state.PendingSyncType != SyncTypeNormal,
		)
		return err
	})
}

func validatePostflight(
	state santaPendingStateRow,
	pendingPayloadRuleCount uint64,
	rulesReceived uint32,
	rulesProcessed uint32,
	syncType SyncType,
	rulesHash string,
) error {
	if state.PendingSyncType == "" {
		return fmt.Errorf("%w: no pending Santa sync", fault.ErrInvalidInput)
	}
	validSyncType := syncType == state.PendingSyncType
	if state.PendingSyncType == SyncTypeClean {
		validSyncType = syncType == SyncTypeClean || syncType == SyncTypeCleanAll
	}
	if !validSyncType {
		return fmt.Errorf("%w: sync_type %q does not match pending sync", fault.ErrInvalidInput, syncType)
	}
	if uint64(rulesReceived) != pendingPayloadRuleCount || uint64(rulesProcessed) != pendingPayloadRuleCount {
		return fmt.Errorf(
			"%w: rules_received and rules_processed must equal pending rule count %d",
			fault.ErrInvalidInput,
			pendingPayloadRuleCount,
		)
	}
	if err := validateRulesHash(rulesHash); err != nil {
		return err
	}
	if pendingPayloadRuleCount == 0 && rulesHash != state.PreflightRulesHash {
		return fmt.Errorf("%w: rules_hash changed during an empty sync", fault.ErrInvalidInput)
	}
	return nil
}
