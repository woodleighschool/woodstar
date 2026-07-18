package syncstate

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func (s *Store) PromotePending(
	ctx context.Context,
	hostID int64,
	rulesReceived int32,
	rulesProcessed int32,
	syncType SyncType,
	rulesHash string,
) error {
	var validationErr error
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var state santaPendingStateRow
		err := tx.QueryRow(ctx, `
SELECT
    pending_payload_rule_count,
    pending_full_sync,
    pending_preflight_at,
    preflight_rules_hash
FROM santa_sync_state
WHERE host_id = $1
FOR UPDATE`, hostID).Scan(
			&state.PendingPayloadRuleCount,
			&state.PendingFullSync,
			&state.PendingPreflightAt,
			&state.PreflightRulesHash,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			validationErr = fmt.Errorf("%w: no pending Santa sync", dbutil.ErrInvalidInput)
			return nil
		}
		if err != nil {
			return err
		}
		if err := validatePostflight(state, rulesReceived, rulesProcessed, syncType, rulesHash); err != nil {
			if updateErr := recordPostflightAttempt(ctx, tx, hostID, rulesReceived, rulesProcessed); updateErr != nil {
				return updateErr
			}
			validationErr = err
			return nil
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM santa_sync_targets WHERE host_id = $1 AND phase = $2::santa_sync_target_phase`,
			hostID, phaseApplied,
		); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO santa_sync_targets (
    host_id, phase, position, rule_type, identifier, policy,
    cel_expression, custom_message, custom_url, notification_app_name,
    payload_hash, updated_at
)
SELECT
    host_id,
    'applied'::santa_sync_target_phase,
    position,
    rule_type,
    identifier,
    policy,
    cel_expression,
    custom_message,
    custom_url,
    notification_app_name,
    payload_hash,
    now()
FROM santa_sync_targets
WHERE santa_sync_targets.host_id = $1 AND santa_sync_targets.phase = 'desired'
ORDER BY position`,
			hostID,
		); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
UPDATE santa_sync_state
SET
    rules_received = $2,
    rules_processed = $3,
    pending_full_sync = false,
    pending_payload_rule_count = 0,
    pending_preflight_at = NULL,
    confirmed_rules_hash = $5,
    last_rule_sync_attempt_at = now(),
    last_rule_sync_success_at = now(),
    last_clean_sync_at = CASE WHEN $4::boolean THEN now() ELSE last_clean_sync_at END,
    updated_at = now()
WHERE host_id = $1`,
			hostID, rulesReceived, rulesProcessed, state.PendingFullSync, rulesHash,
		)
		return err
	})
	if err != nil {
		return err
	}
	return validationErr
}

func recordPostflightAttempt(
	ctx context.Context,
	tx pgx.Tx,
	hostID int64,
	rulesReceived int32,
	rulesProcessed int32,
) error {
	_, err := tx.Exec(ctx, `
UPDATE santa_sync_state
SET
    rules_received = $2,
    rules_processed = $3,
    last_rule_sync_attempt_at = now(),
    updated_at = now()
WHERE host_id = $1`, hostID, rulesReceived, rulesProcessed)
	return err
}

func validatePostflight(
	state santaPendingStateRow,
	rulesReceived int32,
	rulesProcessed int32,
	syncType SyncType,
	rulesHash string,
) error {
	if state.PendingPreflightAt == nil {
		return fmt.Errorf("%w: no pending Santa sync", dbutil.ErrInvalidInput)
	}
	validSyncType := syncType == SyncTypeNormal
	if state.PendingFullSync {
		validSyncType = syncType == SyncTypeClean || syncType == SyncTypeCleanAll
	}
	if !validSyncType {
		return fmt.Errorf("%w: sync_type %q does not match pending sync", dbutil.ErrInvalidInput, syncType)
	}
	if rulesReceived != state.PendingPayloadRuleCount || rulesProcessed != state.PendingPayloadRuleCount {
		return fmt.Errorf(
			"%w: rules_received and rules_processed must equal pending rule count %d",
			dbutil.ErrInvalidInput,
			state.PendingPayloadRuleCount,
		)
	}
	if err := validateRulesHash(rulesHash); err != nil {
		return err
	}
	if state.PendingPayloadRuleCount == 0 && rulesHash != state.PreflightRulesHash {
		return fmt.Errorf("%w: rules_hash changed during an empty sync", dbutil.ErrInvalidInput)
	}
	return nil
}
