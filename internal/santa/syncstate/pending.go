package syncstate

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const (
	phaseDesired = "desired"
	phaseApplied = "applied"
)

type Target struct {
	RuleType      string `json:"rule_type"`
	Identifier    string `json:"identifier"`
	Policy        string `json:"policy"`
	CELExpression string `json:"cel_expression,omitempty"`
	CustomMessage string `json:"custom_message,omitempty"`
	CustomURL     string `json:"custom_url,omitempty"`
	AppName       string `json:"notification_app_name,omitempty"`
	PayloadHash   string `json:"payload_hash"`
}

type PayloadRule struct {
	RuleType      string `json:"rule_type"`
	Identifier    string `json:"identifier"`
	Policy        string `json:"policy,omitempty"`
	CELExpression string `json:"cel_expression,omitempty"`
	CustomMessage string `json:"custom_message,omitempty"`
	CustomURL     string `json:"custom_url,omitempty"`
	AppName       string `json:"notification_app_name,omitempty"`
	PayloadHash   string `json:"payload_hash,omitempty"`
	Removed       bool   `json:"removed,omitempty"`
}

type PayloadRulePage struct {
	Rules  []PayloadRule
	Cursor string
}

type pageCursor struct {
	Offset int `json:"offset"`
}

func (target Target) key() string {
	return target.RuleType + "\x00" + target.Identifier + "\x00" + target.PayloadHash
}

func (target Target) identityKey() string {
	return target.RuleType + "\x00" + target.Identifier
}

func TargetSet(targets []Target) map[string]bool {
	out := make(map[string]bool, len(targets))
	for _, target := range targets {
		out[target.key()] = true
	}
	return out
}

func (s *Store) PreparePending(
	ctx context.Context,
	hostID int64,
	clientRulesHash string,
	desired []Target,
	reported RuleCounts,
	requestCleanSync bool,
) (SyncType, error) {
	desired = sortedTargets(desired)

	var hadState bool
	var pendingFullSync bool
	var applied []Target
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var err error
		hadState, err = syncStateExists(ctx, tx, hostID)
		if err != nil {
			return err
		}
		applied, err = loadTargets(ctx, tx, hostID, phaseApplied)
		if err != nil {
			return err
		}
		pendingFullSync = requestCleanSync ||
			!hadState ||
			(targetsEqual(desired, applied) && countTargets(desired) != reported)

		payload := incrementalPayload(desired, applied)
		if pendingFullSync {
			payload = fullSyncPayload(desired)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO santa_sync_state (
				host_id,
				client_rules_hash,
				pending_full_sync,
				pending_payload_rule_count,
				pending_preflight_at,
				desired_binary_rule_count,
				desired_certificate_rule_count,
				desired_teamid_rule_count,
				desired_signingid_rule_count,
				desired_cdhash_rule_count,
				binary_rule_count,
				certificate_rule_count,
				teamid_rule_count,
				signingid_rule_count,
				cdhash_rule_count,
				last_rule_sync_attempt_at,
				last_reported_counts_match_at,
				updated_at
			)
			VALUES (
				$1,
				$2,
				$3,
				$4,
				now(),
				$5,
				$6,
				$7,
				$8,
				$9,
				$10,
				$11,
				$12,
				$13,
				$14,
				now(),
				CASE WHEN $15 THEN now() ELSE NULL END,
				now()
			)
			ON CONFLICT (host_id) DO UPDATE SET
				client_rules_hash = EXCLUDED.client_rules_hash,
				pending_full_sync = EXCLUDED.pending_full_sync,
				pending_payload_rule_count = EXCLUDED.pending_payload_rule_count,
				pending_preflight_at = EXCLUDED.pending_preflight_at,
				desired_binary_rule_count = EXCLUDED.desired_binary_rule_count,
				desired_certificate_rule_count = EXCLUDED.desired_certificate_rule_count,
				desired_teamid_rule_count = EXCLUDED.desired_teamid_rule_count,
				desired_signingid_rule_count = EXCLUDED.desired_signingid_rule_count,
				desired_cdhash_rule_count = EXCLUDED.desired_cdhash_rule_count,
				binary_rule_count = EXCLUDED.binary_rule_count,
				certificate_rule_count = EXCLUDED.certificate_rule_count,
				teamid_rule_count = EXCLUDED.teamid_rule_count,
				signingid_rule_count = EXCLUDED.signingid_rule_count,
				cdhash_rule_count = EXCLUDED.cdhash_rule_count,
				last_rule_sync_attempt_at = EXCLUDED.last_rule_sync_attempt_at,
				last_reported_counts_match_at = CASE
					WHEN $15 THEN EXCLUDED.last_reported_counts_match_at
					ELSE santa_sync_state.last_reported_counts_match_at
				END,
				updated_at = now()
		`,
			hostID,
			clientRulesHash,
			pendingFullSync,
			len(payload),
			countTargets(desired).Binary,
			countTargets(desired).Certificate,
			countTargets(desired).TeamID,
			countTargets(desired).SigningID,
			countTargets(desired).CDHash,
			reported.Binary,
			reported.Certificate,
			reported.TeamID,
			reported.SigningID,
			reported.CDHash,
			countTargets(desired) == reported,
		); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM santa_sync_targets
			WHERE host_id = $1 AND phase = 'desired'
		`, hostID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM santa_sync_pending_rules WHERE host_id = $1`, hostID); err != nil {
			return err
		}
		if err := insertTargets(ctx, tx, hostID, phaseDesired, desired); err != nil {
			return err
		}
		return insertPayloadRules(ctx, tx, hostID, payload)
	})
	if err != nil {
		return "", err
	}
	if pendingFullSync {
		return SyncTypeClean, nil
	}
	return SyncTypeNormal, nil
}

func (s *Store) LoadPendingPayloadPage(
	ctx context.Context,
	hostID int64,
	cursor string,
	limit int,
) (PayloadRulePage, error) {
	if limit <= 0 {
		return PayloadRulePage{}, dbutil.ErrInvalidInput
	}
	offset, err := decodeCursor(cursor)
	if err != nil {
		return PayloadRulePage{}, err
	}

	rows, err := s.db.Pool().Query(ctx, `
		SELECT
			rule_type::text,
			identifier,
			COALESCE(policy::text, ''),
			cel_expression,
			custom_message,
			custom_url,
			notification_app_name,
			payload_hash,
			removed
		FROM santa_sync_pending_rules
		WHERE host_id = $1
		ORDER BY position
		LIMIT $2 OFFSET $3
	`, hostID, limit+1, offset)
	if err != nil {
		return PayloadRulePage{}, err
	}
	defer rows.Close()

	rules, err := pgx.CollectRows(rows, scanPayloadRule)
	if err != nil {
		return PayloadRulePage{}, err
	}
	nextCursor := ""
	if len(rules) > limit {
		rules = rules[:limit]
		nextCursor, err = encodeCursor(offset + limit)
		if err != nil {
			return PayloadRulePage{}, err
		}
	}
	return PayloadRulePage{Rules: rules, Cursor: nextCursor}, nil
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
		SELECT pending_payload_rule_count, pending_full_sync
		FROM santa_sync_state
		WHERE host_id = $1
	`, hostID).Scan(&pendingCount, &pendingFullSync)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if rulesProcessed != pendingCount {
		_, err = s.db.Pool().Exec(ctx, `
			UPDATE santa_sync_state
			SET
				client_rules_hash = $2,
				rules_received = $3,
				rules_processed = $4,
				last_rule_sync_attempt_at = now(),
				updated_at = now()
			WHERE host_id = $1
		`, hostID, clientRulesHash, rulesReceived, rulesProcessed)
		return err
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			DELETE FROM santa_sync_targets
			WHERE host_id = $1 AND phase = 'applied'
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
				notification_app_name,
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
				notification_app_name,
				payload_hash,
				now()
			FROM santa_sync_targets
			WHERE host_id = $1 AND phase = 'desired'
			ORDER BY position
		`, hostID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM santa_sync_pending_rules WHERE host_id = $1`, hostID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE santa_sync_state
			SET
				client_rules_hash = $2,
				rules_received = $3,
				rules_processed = $4,
				pending_full_sync = false,
				pending_payload_rule_count = 0,
				pending_preflight_at = NULL,
				last_rule_sync_attempt_at = now(),
				last_rule_sync_success_at = now(),
				last_clean_sync_at = CASE WHEN $5 THEN now() ELSE last_clean_sync_at END,
				updated_at = now()
			WHERE host_id = $1
		`, hostID, clientRulesHash, rulesReceived, rulesProcessed, pendingFullSync)
		return err
	})
}

func syncStateExists(ctx context.Context, tx pgx.Tx, hostID int64) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM santa_sync_state WHERE host_id = $1)`, hostID).
		Scan(&exists)
	return exists, err
}

func loadTargets(ctx context.Context, tx pgx.Tx, hostID int64, phase string) ([]Target, error) {
	rows, err := tx.Query(ctx, `
		SELECT
			rule_type::text,
			identifier,
			policy::text,
			cel_expression,
			custom_message,
			custom_url,
			notification_app_name,
			payload_hash
		FROM santa_sync_targets
		WHERE host_id = $1 AND phase = $2::santa_sync_target_phase
		ORDER BY position
	`, hostID, phase)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	targets, err := pgx.CollectRows(rows, scanTarget)
	if err != nil {
		return nil, err
	}
	return sortedTargets(targets), nil
}

func insertTargets(ctx context.Context, tx pgx.Tx, hostID int64, phase string, targets []Target) error {
	for position, target := range sortedTargets(targets) {
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
				notification_app_name,
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
				$10,
				$11
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
			target.AppName,
			target.PayloadHash,
		); err != nil {
			return err
		}
	}
	return nil
}

func insertPayloadRules(ctx context.Context, tx pgx.Tx, hostID int64, rules []PayloadRule) error {
	for position, rule := range sortedPayloadRules(rules) {
		var policy *string
		if !rule.Removed {
			policy = &rule.Policy
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO santa_sync_pending_rules (
				host_id,
				position,
				rule_type,
				identifier,
				policy,
					cel_expression,
					custom_message,
					custom_url,
					notification_app_name,
					payload_hash,
					removed
				)
			VALUES (
				$1,
				$2,
				$3::santa_rule_type,
				$4,
				$5::santa_policy,
				$6,
				$7,
				$8,
				$9,
				$10,
				$11
			)
			`,
			hostID,
			position,
			rule.RuleType,
			rule.Identifier,
			policy,
			rule.CELExpression,
			rule.CustomMessage,
			rule.CustomURL,
			rule.AppName,
			rule.PayloadHash,
			rule.Removed,
		); err != nil {
			return err
		}
	}
	return nil
}

func incrementalPayload(desired []Target, applied []Target) []PayloadRule {
	appliedByIdentity := make(map[string]Target, len(applied))
	for _, target := range applied {
		appliedByIdentity[target.identityKey()] = target
	}

	currentIdentities := make(map[string]struct{}, len(desired))
	payload := make([]PayloadRule, 0, len(desired)+len(applied))
	for _, target := range desired {
		currentIdentities[target.identityKey()] = struct{}{}
		appliedTarget, ok := appliedByIdentity[target.identityKey()]
		if !ok || appliedTarget.PayloadHash != target.PayloadHash {
			payload = append(payload, payloadRuleFromTarget(target))
		}
	}

	for _, target := range applied {
		if _, ok := currentIdentities[target.identityKey()]; ok {
			continue
		}
		payload = append(payload, PayloadRule{
			RuleType:   target.RuleType,
			Identifier: target.Identifier,
			Removed:    true,
		})
	}

	return sortedPayloadRules(payload)
}

func fullSyncPayload(targets []Target) []PayloadRule {
	payload := make([]PayloadRule, 0, len(targets))
	for _, target := range targets {
		payload = append(payload, payloadRuleFromTarget(target))
	}
	return sortedPayloadRules(payload)
}

func payloadRuleFromTarget(target Target) PayloadRule {
	return PayloadRule{
		RuleType:      target.RuleType,
		Identifier:    target.Identifier,
		Policy:        target.Policy,
		CELExpression: target.CELExpression,
		CustomMessage: target.CustomMessage,
		CustomURL:     target.CustomURL,
		AppName:       target.AppName,
		PayloadHash:   target.PayloadHash,
	}
}

func targetsEqual(a []Target, b []Target) bool {
	return slices.EqualFunc(sortedTargets(a), sortedTargets(b), func(left Target, right Target) bool {
		return left == right
	})
}

func sortedTargets(targets []Target) []Target {
	out := slices.Clone(targets)
	slices.SortFunc(out, func(left Target, right Target) int {
		if n := cmp.Compare(ruleTypeSort(left.RuleType), ruleTypeSort(right.RuleType)); n != 0 {
			return n
		}
		if n := cmp.Compare(left.Identifier, right.Identifier); n != 0 {
			return n
		}
		return cmp.Compare(left.PayloadHash, right.PayloadHash)
	})
	return out
}

func sortedPayloadRules(rules []PayloadRule) []PayloadRule {
	out := slices.Clone(rules)
	slices.SortFunc(out, func(left PayloadRule, right PayloadRule) int {
		if n := cmp.Compare(ruleTypeSort(left.RuleType), ruleTypeSort(right.RuleType)); n != 0 {
			return n
		}
		if n := cmp.Compare(left.Identifier, right.Identifier); n != 0 {
			return n
		}
		if n := cmp.Compare(left.PayloadHash, right.PayloadHash); n != 0 {
			return n
		}
		switch {
		case left.Removed == right.Removed:
			return 0
		case left.Removed:
			return 1
		default:
			return -1
		}
	})
	return out
}

func ruleTypeSort(ruleType string) int {
	switch ruleType {
	case "cdhash":
		return 1
	case "binary":
		return 2
	case "signingid":
		return 3
	case "certificate":
		return 4
	case "teamid":
		return 5
	default:
		return 6
	}
}

func countTargets(targets []Target) RuleCounts {
	var counts RuleCounts
	for _, target := range targets {
		switch target.RuleType {
		case "binary":
			counts.Binary++
		case "certificate":
			counts.Certificate++
		case "teamid":
			counts.TeamID++
		case "signingid":
			counts.SigningID++
		case "cdhash":
			counts.CDHash++
		}
	}
	return counts
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
		&target.AppName,
		&target.PayloadHash,
	)
	return target, err
}

func scanPayloadRule(row pgx.CollectableRow) (PayloadRule, error) {
	var rule PayloadRule
	err := row.Scan(
		&rule.RuleType,
		&rule.Identifier,
		&rule.Policy,
		&rule.CELExpression,
		&rule.CustomMessage,
		&rule.CustomURL,
		&rule.AppName,
		&rule.PayloadHash,
		&rule.Removed,
	)
	return rule, err
}

func decodeCursor(cursor string) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, dbutil.ErrInvalidInput
	}
	var decoded pageCursor
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return 0, dbutil.ErrInvalidInput
	}
	if decoded.Offset < 0 {
		return 0, dbutil.ErrInvalidInput
	}
	return decoded.Offset, nil
}

func encodeCursor(offset int) (string, error) {
	payload, err := json.Marshal(pageCursor{Offset: offset})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}
