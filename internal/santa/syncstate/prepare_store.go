package syncstate

import (
	"context"
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

	var pendingSyncType SyncType
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		applied, err := loadPriorState(ctx, tx, hostID)
		if err != nil {
			return err
		}
		changes := normalSyncPayload(desired, applied.targets)
		pendingSyncType = applied.requiredSyncType(
			policyDigest,
			desired,
			len(changes) != 0,
			reported,
			requestCleanSync,
			clientRulesHash,
		)

		// A successful postflight leaves desired and applied identical.
		if applied.hasPending || applied.policyDigest == nil {
			current, err := loadTargets(ctx, tx, hostID, phaseDesired)
			if err != nil {
				return err
			}
			changes = normalSyncPayload(desired, current)
		}
		_, err = tx.Exec(ctx, `
UPDATE santa_sync_state
SET pending_sync_type = $2::santa_sync_type, pending_policy_digest = $3,
    preflight_rules_hash = $4, updated_at = now()
WHERE host_id = $1 AND (pending_sync_type, pending_policy_digest, preflight_rules_hash)
    IS DISTINCT FROM ($2::santa_sync_type, $3, $4)`, hostID, pendingSyncType, policyDigest, clientRulesHash)
		if err != nil {
			return err
		}
		return applyTargetChanges(ctx, tx, hostID, phaseDesired, changes)
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
	hasPending         bool
}

func loadPriorState(ctx context.Context, tx pgx.Tx, hostID int64) (priorState, error) {
	// The state row serializes preflight and postflight, including a host's first sync.
	_, err := tx.Exec(ctx, `INSERT INTO santa_sync_state (host_id) VALUES ($1) ON CONFLICT DO NOTHING`, hostID)
	if err != nil {
		return priorState{}, err
	}
	var state priorState
	err = tx.QueryRow(ctx, `
SELECT applied_policy_digest, confirmed_rules_hash, pending_sync_type IS NOT NULL
FROM santa_sync_state WHERE host_id = $1 FOR UPDATE`, hostID).Scan(
		&state.policyDigest, &state.confirmedRulesHash, &state.hasPending,
	)
	if err != nil {
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
	targetsChanged bool,
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
		(!targetsChanged && !countTargets(desired).MatchesReported(reported)) {
		return SyncTypeClean
	}
	return SyncTypeNormal
}

func validateRuleCounts(counts RuleCounts) error {
	if counts.Transitive > counts.Binary {
		return fmt.Errorf("%w: invalid Santa rule counts", fault.ErrInvalidInput)
	}
	return nil
}
