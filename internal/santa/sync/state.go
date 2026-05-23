package sync

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

const (
	phaseDesired = "desired"
	phasePending = "pending"
	phaseApplied = "applied"
)

type Target struct {
	RuleType      string `json:"rule_type"`
	Identifier    string `json:"identifier"`
	Policy        string `json:"policy"`
	CELExpression string `json:"cel_expression,omitempty"`
	CustomMessage string `json:"custom_message,omitempty"`
	CustomURL     string `json:"custom_url,omitempty"`
	PayloadHash   string `json:"payload_hash"`
}

func (target Target) key() string {
	return target.RuleType + "\x00" + target.Identifier + "\x00" + target.PayloadHash
}

func TargetSet(targets []Target) map[string]bool {
	out := make(map[string]bool, len(targets))
	for _, target := range targets {
		out[target.key()] = true
	}
	return out
}

func (s *Store) ReplacePending(
	ctx context.Context,
	hostID int64,
	clientRulesHash string,
	desired []Target,
	pending []Target,
	pendingFullSync bool,
) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO santa_sync_state (
				host_id,
				client_rules_hash,
				pending_full_sync,
				pending_preflight_at,
				last_rule_sync_attempt_at,
				updated_at
			)
			VALUES ($1, $2, $3, now(), now(), now())
			ON CONFLICT (host_id) DO UPDATE SET
				client_rules_hash = EXCLUDED.client_rules_hash,
				pending_full_sync = EXCLUDED.pending_full_sync,
				pending_preflight_at = EXCLUDED.pending_preflight_at,
				last_rule_sync_attempt_at = EXCLUDED.last_rule_sync_attempt_at,
				updated_at = now()
		`, hostID, clientRulesHash, pendingFullSync); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM santa_sync_targets
			WHERE host_id = $1 AND phase IN ('desired', 'pending')
		`, hostID); err != nil {
			return err
		}
		if err := insertTargets(ctx, tx, hostID, phaseDesired, desired); err != nil {
			return err
		}
		return insertTargets(ctx, tx, hostID, phasePending, pending)
	})
}

func (s *Store) LoadPendingTargets(ctx context.Context, hostID int64) ([]Target, error) {
	targets, err := s.loadTargets(ctx, hostID, phasePending)
	if err != nil {
		return nil, err
	}
	return targets, nil
}

func (s *Store) PromotePending(
	ctx context.Context,
	hostID int64,
	clientRulesHash string,
	rulesReceived int,
	rulesProcessed int,
) error {
	var pendingCount int
	var pendingFullSync bool
	err := s.db.Pool().QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM santa_sync_targets WHERE host_id = ss.host_id AND phase = 'pending'),
			ss.pending_full_sync
		FROM santa_sync_state ss
		WHERE ss.host_id = $1
	`, hostID).Scan(&pendingCount, &pendingFullSync)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if rulesReceived != pendingCount || rulesProcessed != pendingCount {
		_, err = s.db.Pool().Exec(ctx, `
			UPDATE santa_sync_state
			SET client_rules_hash = $2, updated_at = now()
			WHERE host_id = $1
		`, hostID, clientRulesHash)
		return err
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			DELETE FROM santa_sync_targets
			WHERE host_id = $1 AND phase IN ('applied', 'pending')
		`, hostID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO santa_sync_targets (
				host_id,
				phase,
				position,
				rule_type,
				identifier,
				policy,
				cel_expression,
				custom_message,
				custom_url,
				payload_hash,
				updated_at
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
				payload_hash,
				now()
			FROM santa_sync_targets
			WHERE host_id = $1 AND phase = 'desired'
			ORDER BY position
		`, hostID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE santa_sync_state
			SET
				client_rules_hash = $2,
				pending_full_sync = false,
				last_rule_sync_success_at = now(),
				last_clean_sync_at = CASE WHEN $3 THEN now() ELSE last_clean_sync_at END,
				updated_at = now()
			WHERE host_id = $1
		`, hostID, clientRulesHash, pendingFullSync)
		return err
	})
}

func (s *Store) loadTargets(ctx context.Context, hostID int64, phase string) ([]Target, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT
			rule_type::text,
			identifier,
			policy::text,
			cel_expression,
			custom_message,
			custom_url,
			payload_hash
		FROM santa_sync_targets
		WHERE host_id = $1 AND phase = $2::santa_sync_target_phase
		ORDER BY position
	`, hostID, phase)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanTarget)
}

func insertTargets(ctx context.Context, tx pgx.Tx, hostID int64, phase string, targets []Target) error {
	for position, target := range targets {
		if _, err := tx.Exec(ctx, `
			INSERT INTO santa_sync_targets (
				host_id,
				phase,
				position,
				rule_type,
				identifier,
				policy,
				cel_expression,
				custom_message,
				custom_url,
				payload_hash
			)
			VALUES (
				$1,
				$2::santa_sync_target_phase,
				$3,
				$4::santa_rule_type,
				$5,
				$6::santa_policy,
				$7,
				$8,
				$9,
				$10
			)
		`,
			hostID,
			phase,
			position,
			target.RuleType,
			target.Identifier,
			target.Policy,
			target.CELExpression,
			target.CustomMessage,
			target.CustomURL,
			target.PayloadHash,
		); err != nil {
			return err
		}
	}
	return nil
}

func scanTarget(row pgx.CollectableRow) (Target, error) {
	var target Target
	err := row.Scan(
		&target.RuleType,
		&target.Identifier,
		&target.Policy,
		&target.CELExpression,
		&target.CustomMessage,
		&target.CustomURL,
		&target.PayloadHash,
	)
	return target, err
}
