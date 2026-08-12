package syncstate

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/woodleighschool/woodstar/internal/fault"
)

func (s *Store) PreparePending(
	ctx context.Context,
	hostID int64,
	policyDigest string,
	desired []Target,
	reported RuleCounts,
	requestCleanSync bool,
	clientRulesHash string,
) (SyncType, error) {
	if err := validateRuleCounts(reported); err != nil {
		return "", err
	}
	if err := validateRulesHash(clientRulesHash); err != nil {
		return "", err
	}
	if err := validatePolicyDigest(policyDigest); err != nil {
		return "", err
	}
	desired = sortedTargets(desired)

	var pendingSyncType SyncType
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		applied, err := loadPriorState(ctx, tx, hostID)
		if err != nil {
			return err
		}
		pendingSyncType = applied.requiredSyncType(
			policyDigest,
			desired,
			reported,
			requestCleanSync,
			clientRulesHash,
		)

		if err := upsertPreflight(ctx, tx, preflightParams{
			hostID:          hostID,
			policyDigest:    policyDigest,
			pendingSyncType: pendingSyncType,
			clientRulesHash: clientRulesHash,
		}); err != nil {
			return err
		}
		return rewritePendingState(ctx, tx, hostID, desired)
	})
	if err != nil {
		return "", err
	}
	return pendingSyncType, nil
}

// priorState is the applied sync state loaded at the start of a preflight.
type priorState struct {
	policyDigest       *string
	targets            []Target
	confirmedRulesHash *string
}

func loadPriorState(ctx context.Context, tx pgx.Tx, hostID int64) (priorState, error) {
	var state priorState
	err := tx.QueryRow(
		ctx,
		`SELECT applied_policy_digest, confirmed_rules_hash
		 FROM santa_sync_state WHERE host_id = $1`,
		hostID,
	).Scan(&state.policyDigest, &state.confirmedRulesHash)
	if errors.Is(err, pgx.ErrNoRows) {
		state = priorState{}
	} else if err != nil {
		return priorState{}, err
	}
	targets, err := loadTargets(ctx, tx, hostID, phaseApplied)
	if err != nil {
		return priorState{}, err
	}
	state.targets = targets
	return state, nil
}

func (p priorState) requiredSyncType(
	policyDigest string,
	desired []Target,
	reported RuleCounts,
	requestCleanSync bool,
	clientRulesHash string,
) SyncType {
	if p.policyDigest == nil || *p.policyDigest != policyDigest {
		return SyncTypeCleanAll
	}
	if transitiveAuthorityRemoved(p.targets, desired) {
		return SyncTypeCleanAll
	}
	if requestCleanSync ||
		(p.confirmedRulesHash != nil && *p.confirmedRulesHash != clientRulesHash) ||
		(targetsEqual(desired, p.targets) && !countTargets(desired).MatchesReported(reported)) {
		return SyncTypeClean
	}
	return SyncTypeNormal
}

type preflightParams struct {
	hostID          int64
	policyDigest    string
	pendingSyncType SyncType
	clientRulesHash string
}

const upsertPreflightSQL = `
INSERT INTO santa_sync_state (
    host_id,
    pending_sync_type,
    pending_policy_digest,
    preflight_rules_hash,
    updated_at
)
VALUES (
    $1, $2::santa_sync_type, $3, $4, now()
)
ON CONFLICT (host_id) DO UPDATE SET
    pending_sync_type = EXCLUDED.pending_sync_type,
    pending_policy_digest = EXCLUDED.pending_policy_digest,
    preflight_rules_hash = EXCLUDED.preflight_rules_hash,
    updated_at = now()`

func upsertPreflight(ctx context.Context, tx pgx.Tx, p preflightParams) error {
	_, err := tx.Exec(ctx, upsertPreflightSQL,
		p.hostID,
		p.pendingSyncType,
		p.policyDigest,
		p.clientRulesHash,
	)
	return err
}

func rewritePendingState(
	ctx context.Context,
	tx pgx.Tx,
	hostID int64,
	desired []Target,
) error {
	if _, err := tx.Exec(ctx,
		`DELETE FROM santa_sync_targets WHERE host_id = $1 AND phase = $2::santa_sync_target_phase`,
		hostID, phaseDesired,
	); err != nil {
		return err
	}
	return insertTargets(ctx, tx, hostID, phaseDesired, desired)
}

func validateRuleCounts(counts RuleCounts) error {
	if counts.Transitive > counts.Binary {
		return fmt.Errorf("%w: invalid Santa rule counts", fault.ErrInvalidInput)
	}
	return nil
}
