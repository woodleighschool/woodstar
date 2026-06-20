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
	Offset int32 `json:"offset"`
}

func (target Target) Key() string {
	return target.RuleType + "\x00" + target.Identifier + "\x00" + target.PayloadHash
}

func (target Target) identityKey() string {
	return target.RuleType + "\x00" + target.Identifier
}

func TargetSet(targets []Target) map[string]bool {
	out := make(map[string]bool, len(targets))
	for _, target := range targets {
		out[target.Key()] = true
	}
	return out
}

func (s *Store) PreparePending(
	ctx context.Context,
	hostID int64,
	desired []Target,
	reported RuleCounts,
	requestCleanSync bool,
) (SyncType, error) {
	desired = sortedTargets(desired)

	var pendingFullSync bool
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		applied, err := loadPriorState(ctx, tx, hostID)
		if err != nil {
			return err
		}
		pendingFullSync = applied.requiresFullSync(desired, reported, requestCleanSync)

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
	exists  bool
	targets []Target
}

func loadPriorState(ctx context.Context, tx pgx.Tx, hostID int64) (priorState, error) {
	exists, err := syncStateExists(ctx, tx, hostID)
	if err != nil {
		return priorState{}, err
	}
	targets, err := loadTargets(ctx, tx, hostID, phaseApplied)
	if err != nil {
		return priorState{}, err
	}
	return priorState{exists: exists, targets: targets}, nil
}

func (p priorState) requiresFullSync(desired []Target, reported RuleCounts, requestCleanSync bool) bool {
	return requestCleanSync ||
		!p.exists ||
		(targetsEqual(desired, p.targets) && !countTargets(desired).MatchesReported(reported))
}

type preflightParams struct {
	hostID          int64
	pendingFullSync bool
	payload         []PayloadRule
	desired         []Target
	reported        RuleCounts
}

const upsertPreflightSQL = `
INSERT INTO santa_sync_state (
    host_id,
    pending_full_sync,
    pending_payload_rule_count,
    pending_preflight_at,
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
    $1, $2, $3, now(),
    $4, $5, $6, $7, $8, $9,
    $10, $11, $12, $13, $14, $15, $16,
    now(),
    CASE WHEN $17::boolean THEN now() ELSE NULL END,
    now()
)
ON CONFLICT (host_id) DO UPDATE SET
    pending_full_sync = EXCLUDED.pending_full_sync,
    pending_payload_rule_count = EXCLUDED.pending_payload_rule_count,
    pending_preflight_at = EXCLUDED.pending_preflight_at,
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
        WHEN $17::boolean THEN EXCLUDED.last_reported_counts_match_at
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

type santaPendingStateRow struct {
	PendingPayloadRuleCount int32
	PendingFullSync         bool
}

func (s *Store) LoadPendingPayloadPage(
	ctx context.Context,
	hostID int64,
	cursor string,
	limit int32,
) (PayloadRulePage, error) {
	if limit <= 0 {
		return PayloadRulePage{}, dbutil.ErrInvalidInput
	}
	limitRows := int(limit)
	offset, err := decodeCursor(cursor)
	if err != nil {
		return PayloadRulePage{}, err
	}

	var state santaPendingStateRow
	err = s.db.Pool().QueryRow(ctx,
		`SELECT pending_payload_rule_count, pending_full_sync FROM santa_sync_state WHERE host_id = $1`,
		hostID,
	).Scan(&state.PendingPayloadRuleCount, &state.PendingFullSync)
	if errors.Is(err, pgx.ErrNoRows) {
		return PayloadRulePage{}, nil
	}
	if err != nil {
		return PayloadRulePage{}, err
	}
	if state.PendingPayloadRuleCount == 0 {
		return PayloadRulePage{}, nil
	}
	desired, err := loadTargets(ctx, s.db.Pool(), hostID, phaseDesired)
	if err != nil {
		return PayloadRulePage{}, err
	}
	payload := fullSyncPayload(desired)
	if !state.PendingFullSync {
		applied, err := loadTargets(ctx, s.db.Pool(), hostID, phaseApplied)
		if err != nil {
			return PayloadRulePage{}, err
		}
		payload = normalSyncPayload(desired, applied)
	}

	start := int(offset)
	if start >= len(payload) {
		return PayloadRulePage{}, nil
	}
	end := min(start+limitRows+1, len(payload))
	rules := payload[start:end]
	nextCursor := ""
	if len(rules) > limitRows {
		rules = rules[:limitRows]
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
	rulesReceived int32,
	rulesProcessed int32,
) error {
	var state santaPendingStateRow
	err := s.db.Pool().QueryRow(ctx,
		`SELECT pending_payload_rule_count, pending_full_sync FROM santa_sync_state WHERE host_id = $1`,
		hostID,
	).Scan(&state.PendingPayloadRuleCount, &state.PendingFullSync)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if rulesProcessed != state.PendingPayloadRuleCount {
		_, err = s.db.Pool().Exec(ctx, `
UPDATE santa_sync_state
SET
    rules_received = $2,
    rules_processed = $3,
    last_rule_sync_attempt_at = now(),
    updated_at = now()
WHERE host_id = $1`,
			hostID, rulesReceived, rulesProcessed,
		)
		return err
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
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
		_, err := tx.Exec(ctx, `
UPDATE santa_sync_state
SET
    rules_received = $2,
    rules_processed = $3,
    pending_full_sync = false,
    pending_payload_rule_count = 0,
    pending_preflight_at = NULL,
    last_rule_sync_attempt_at = now(),
    last_rule_sync_success_at = now(),
    last_clean_sync_at = CASE WHEN $4::boolean THEN now() ELSE last_clean_sync_at END,
    updated_at = now()
WHERE host_id = $1`,
			hostID, rulesReceived, rulesProcessed, state.PendingFullSync,
		)
		return err
	})
}

func syncStateExists(ctx context.Context, tx pgx.Tx, hostID int64) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM santa_sync_state WHERE host_id = $1)`, hostID).Scan(&exists)
	return exists, err
}

type targetRow struct {
	RuleType            string `db:"rule_type"`
	Identifier          string `db:"identifier"`
	Policy              string `db:"policy"`
	CelExpression       string `db:"cel_expression"`
	CustomMessage       string `db:"custom_message"`
	CustomURL           string `db:"custom_url"`
	NotificationAppName string `db:"notification_app_name"`
	PayloadHash         string `db:"payload_hash"`
}

const listTargetsSQL = `
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
ORDER BY position`

func loadTargets(ctx context.Context, q dbutil.Queryer, hostID int64, phase string) ([]Target, error) {
	qrows, err := q.Query(ctx, listTargetsSQL, hostID, phase)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[targetRow])
	if err != nil {
		return nil, err
	}
	targets := make([]Target, len(rows))
	for i, row := range rows {
		targets[i] = targetFromRow(row)
	}
	return sortedTargets(targets), nil
}

func insertTargets(ctx context.Context, tx pgx.Tx, hostID int64, phase string, targets []Target) error {
	for position, target := range sortedTargets(targets) {
		if _, err := tx.Exec(ctx, `
INSERT INTO santa_sync_targets (
    host_id, phase, position, rule_type, identifier, policy,
    cel_expression, custom_message, custom_url, notification_app_name, payload_hash
)
VALUES (
    $1, $2::santa_sync_target_phase, $3, $4::santa_rule_type, $5, $6::santa_policy,
    $7, $8, $9, $10, $11
)`,
			hostID, phase, int32(position),
			target.RuleType, target.Identifier, target.Policy,
			target.CELExpression, target.CustomMessage, target.CustomURL,
			target.AppName, target.PayloadHash,
		); err != nil {
			return err
		}
	}
	return nil
}

func normalSyncPayload(desired []Target, applied []Target) []PayloadRule {
	currentIdentities := make(map[string]struct{}, len(desired))
	payload := make([]PayloadRule, 0, len(desired)+len(applied))
	for _, target := range desired {
		currentIdentities[target.identityKey()] = struct{}{}
		payload = append(payload, payloadRuleFromTarget(target))
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
			if target.Policy == "allowlist_compiler" {
				counts.Compiler++
			}
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

func targetFromRow(row targetRow) Target {
	return Target{
		RuleType:      row.RuleType,
		Identifier:    row.Identifier,
		Policy:        row.Policy,
		CELExpression: row.CelExpression,
		CustomMessage: row.CustomMessage,
		CustomURL:     row.CustomURL,
		AppName:       row.NotificationAppName,
		PayloadHash:   row.PayloadHash,
	}
}

func decodeCursor(cursor string) (int32, error) {
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

func encodeCursor(offset int32) (string, error) {
	payload, err := json.Marshal(pageCursor{Offset: offset})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}
