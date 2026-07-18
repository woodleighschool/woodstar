package syncstate

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func (s *Store) PreparePending(
	ctx context.Context,
	hostID int64,
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
	desired = sortedTargets(desired)

	var pendingFullSync bool
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		applied, err := loadPriorState(ctx, tx, hostID)
		if err != nil {
			return err
		}
		pendingFullSync = applied.requiresFullSync(desired, reported, requestCleanSync, clientRulesHash)

		payload := normalSyncPayload(desired, applied.targets)
		if pendingFullSync {
			payload = fullSyncPayload(desired)
		}

		if err := upsertPreflight(ctx, tx, preflightParams{
			hostID:          hostID,
			pendingFullSync: pendingFullSync,
			payload:         payload,
			desired:         desired,
			reported:        reported,
			clientRulesHash: clientRulesHash,
		}); err != nil {
			return err
		}
		return rewritePendingState(ctx, tx, hostID, desired)
	})
	if err != nil {
		return "", err
	}
	if pendingFullSync {
		return SyncTypeClean, nil
	}
	return SyncTypeNormal, nil
}

// priorState is the applied sync state loaded at the start of a preflight.
type priorState struct {
	exists             bool
	targets            []Target
	confirmedRulesHash string
}

func loadPriorState(ctx context.Context, tx pgx.Tx, hostID int64) (priorState, error) {
	var confirmedRulesHash string
	err := tx.QueryRow(
		ctx,
		`SELECT confirmed_rules_hash FROM santa_sync_state WHERE host_id = $1`,
		hostID,
	).Scan(&confirmedRulesHash)
	exists := true
	if errors.Is(err, pgx.ErrNoRows) {
		exists = false
	} else if err != nil {
		return priorState{}, err
	}
	targets, err := loadTargets(ctx, tx, hostID, phaseApplied)
	if err != nil {
		return priorState{}, err
	}
	return priorState{exists: exists, targets: targets, confirmedRulesHash: confirmedRulesHash}, nil
}

func (p priorState) requiresFullSync(
	desired []Target,
	reported RuleCounts,
	requestCleanSync bool,
	clientRulesHash string,
) bool {
	return requestCleanSync ||
		!p.exists ||
		(p.confirmedRulesHash != "" && p.confirmedRulesHash != clientRulesHash) ||
		(targetsEqual(desired, p.targets) && !countTargets(desired).MatchesReported(reported))
}

type preflightParams struct {
	hostID          int64
	pendingFullSync bool
	payload         []PayloadRule
	desired         []Target
	reported        RuleCounts
	clientRulesHash string
}

const upsertPreflightSQL = `
INSERT INTO santa_sync_state (
    host_id,
    pending_full_sync,
    pending_payload_rule_count,
    pending_preflight_at,
    preflight_rules_hash,
    desired_binary_rule_count,
    desired_certificate_rule_count,
    desired_teamid_rule_count,
    desired_signingid_rule_count,
    desired_cdhash_rule_count,
    desired_compiler_rule_count,
    binary_rule_count,
    certificate_rule_count,
    teamid_rule_count,
    signingid_rule_count,
    cdhash_rule_count,
    compiler_rule_count,
    transitive_rule_count,
    last_rule_sync_attempt_at,
    last_reported_counts_match_at,
    updated_at
)
VALUES (
    $1, $2, $3, now(), $4,
    $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17,
    now(),
    CASE WHEN $18::boolean THEN now() ELSE NULL END,
    now()
)
ON CONFLICT (host_id) DO UPDATE SET
    pending_full_sync = EXCLUDED.pending_full_sync,
    pending_payload_rule_count = EXCLUDED.pending_payload_rule_count,
    pending_preflight_at = EXCLUDED.pending_preflight_at,
    preflight_rules_hash = EXCLUDED.preflight_rules_hash,
    desired_binary_rule_count = EXCLUDED.desired_binary_rule_count,
    desired_certificate_rule_count = EXCLUDED.desired_certificate_rule_count,
    desired_teamid_rule_count = EXCLUDED.desired_teamid_rule_count,
    desired_signingid_rule_count = EXCLUDED.desired_signingid_rule_count,
    desired_cdhash_rule_count = EXCLUDED.desired_cdhash_rule_count,
    desired_compiler_rule_count = EXCLUDED.desired_compiler_rule_count,
    binary_rule_count = EXCLUDED.binary_rule_count,
    certificate_rule_count = EXCLUDED.certificate_rule_count,
    teamid_rule_count = EXCLUDED.teamid_rule_count,
    signingid_rule_count = EXCLUDED.signingid_rule_count,
    cdhash_rule_count = EXCLUDED.cdhash_rule_count,
    compiler_rule_count = EXCLUDED.compiler_rule_count,
    transitive_rule_count = EXCLUDED.transitive_rule_count,
    last_rule_sync_attempt_at = EXCLUDED.last_rule_sync_attempt_at,
    last_reported_counts_match_at = CASE
        WHEN $18::boolean THEN EXCLUDED.last_reported_counts_match_at
        ELSE santa_sync_state.last_reported_counts_match_at
    END,
    updated_at = now()`

func upsertPreflight(ctx context.Context, tx pgx.Tx, p preflightParams) error {
	desiredCounts := countTargets(p.desired)
	countsMatch := desiredCounts.MatchesReported(p.reported)
	_, err := tx.Exec(ctx, upsertPreflightSQL,
		p.hostID,
		p.pendingFullSync,
		int32(len(p.payload)),
		p.clientRulesHash,
		desiredCounts.Binary,
		desiredCounts.Certificate,
		desiredCounts.TeamID,
		desiredCounts.SigningID,
		desiredCounts.CDHash,
		desiredCounts.Compiler,
		p.reported.Binary,
		p.reported.Certificate,
		p.reported.TeamID,
		p.reported.SigningID,
		p.reported.CDHash,
		p.reported.Compiler,
		p.reported.Transitive,
		countsMatch,
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
	if counts.Binary < 0 ||
		counts.Certificate < 0 ||
		counts.TeamID < 0 ||
		counts.SigningID < 0 ||
		counts.CDHash < 0 ||
		counts.Compiler < 0 ||
		counts.Transitive < 0 ||
		counts.Transitive > counts.Binary {
		return fmt.Errorf("%w: invalid Santa rule counts", dbutil.ErrInvalidInput)
	}
	return nil
}
