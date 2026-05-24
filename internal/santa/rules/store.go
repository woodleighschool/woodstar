package rules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	santaids "github.com/woodleighschool/woodstar/internal/santa/ids"
	santasync "github.com/woodleighschool/woodstar/internal/santa/sync"
)

// Store persists Santa rule definitions and resolves effective rule state.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

var validRuleTypes = map[RuleType]struct{}{
	RuleTypeBinary:      {},
	RuleTypeCertificate: {},
	RuleTypeTeamID:      {},
	RuleTypeSigningID:   {},
	RuleTypeCDHash:      {},
}

var validPolicies = map[Policy]struct{}{
	PolicyAllowlist:         {},
	PolicyAllowlistCompiler: {},
	PolicyBlocklist:         {},
	PolicySilentBlocklist:   {},
	PolicyCEL:               {},
}

func (s *Store) ListRules(ctx context.Context, params RuleListParams) ([]Rule, int, error) {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	params.RuleType = RuleType(strings.TrimSpace(string(params.RuleType)))
	where, args, err := ruleListWhere(params)
	if err != nil {
		return nil, 0, err
	}

	var count int
	if err := s.db.Pool().QueryRow(ctx, "SELECT count(*) FROM santa_rules "+where, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := ruleListSQL(params, where, args)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	rules := []Rule{}
	ruleIDs := []int64{}
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, 0, err
		}
		rules = append(rules, rule)
		ruleIDs = append(ruleIDs, rule.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := s.attachRuleChildren(ctx, rules, ruleIDs); err != nil {
		return nil, 0, err
	}
	return rules, count, nil
}

func (s *Store) GetRuleByID(ctx context.Context, id int64) (*Rule, error) {
	rule, err := s.getRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	rules := []Rule{*rule}
	if err := s.attachRuleChildren(ctx, rules, []int64{rule.ID}); err != nil {
		return nil, err
	}
	return &rules[0], nil
}

func (s *Store) getRuleByID(ctx context.Context, id int64) (*Rule, error) {
	row, err := s.q.GetSantaRuleByID(ctx, sqlc.GetSantaRuleByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rule := ruleFromSQLC(row)
	return &rule, nil
}

func (s *Store) CreateRule(ctx context.Context, params RuleMutation) (*Rule, error) {
	cleaned, err := cleanRuleMutation(params)
	if err != nil {
		return nil, err
	}

	var ruleID int64
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row, err := s.q.WithTx(tx).CreateSantaRule(ctx, sqlc.CreateSantaRuleParams{
			RuleType:      sqlc.SantaRuleType(cleaned.RuleType),
			Identifier:    cleaned.Identifier,
			Name:          cleaned.Name,
			CustomMessage: cleaned.CustomMessage,
			CustomURL:     cleaned.CustomURL,
		})
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		ruleID = row.ID
		return replaceRuleChildren(ctx, tx, ruleID, cleaned.Includes, cleaned.ExcludeLabelIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.GetRuleByID(ctx, ruleID)
}

func (s *Store) UpdateRule(ctx context.Context, id int64, params RuleMutation) (*Rule, error) {
	cleaned, err := cleanRuleMutation(params)
	if err != nil {
		return nil, err
	}

	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row, err := s.q.WithTx(tx).UpdateSantaRule(ctx, sqlc.UpdateSantaRuleParams{
			RuleType:      sqlc.SantaRuleType(cleaned.RuleType),
			Identifier:    cleaned.Identifier,
			Name:          cleaned.Name,
			CustomMessage: cleaned.CustomMessage,
			CustomURL:     cleaned.CustomURL,
			ID:            id,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return dbutil.ErrNotFound
		}
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		return replaceRuleChildren(ctx, tx, row.ID, cleaned.Includes, cleaned.ExcludeLabelIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.GetRuleByID(ctx, id)
}

func (s *Store) DeleteRule(ctx context.Context, id int64) error {
	_, err := s.q.DeleteSantaRule(ctx, sqlc.DeleteSantaRuleParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

// DeleteMany removes multiple Santa rules. Missing IDs are ignored so repeated bulk actions are idempotent.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	deletedIDs, err := s.q.DeleteSantaRules(ctx, sqlc.DeleteSantaRulesParams{Ids: ids})
	if err != nil {
		return 0, err
	}
	return len(deletedIDs), nil
}

func (s *Store) ReorderRuleIncludes(ctx context.Context, ruleID int64, orderedIncludeIDs []int64) error {
	if ruleID <= 0 {
		return dbutil.ErrNotFound
	}
	ids, err := santaids.ParsePositive(orderedIncludeIDs, "ordered_include_ids")
	if err != nil {
		return err
	}

	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM santa_rules
				WHERE id = $1
			)
		`, ruleID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return dbutil.ErrNotFound
		}

		rows, err := tx.Query(ctx, `
			SELECT id
			FROM santa_rule_includes
			WHERE rule_id = $1
			ORDER BY position, id
		`, ruleID)
		if err != nil {
			return err
		}
		currentIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil {
			return err
		}
		if !santaids.SameSet(ids, currentIDs) {
			return fmt.Errorf("%w: ordered_include_ids must exactly match existing include IDs", dbutil.ErrInvalidInput)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE santa_rule_includes i
			SET position = -ordered.position
			FROM unnest($1::bigint[]) WITH ORDINALITY AS ordered(id, position)
			WHERE i.id = ordered.id AND i.rule_id = $2
		`, ids, ruleID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE santa_rule_includes
			SET position = -position - 1
			WHERE rule_id = $1
		`, ruleID)
		return err
	})
}

func (s *Store) ResolveRulesForHost(ctx context.Context, hostID int64) ([]EffectiveRule, error) {
	rows, err := s.db.Pool().Query(ctx, effectiveRulesForHostSQL+`
		ORDER BY rule_type_sort, identifier, rule_id
	`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, scanEffectiveRule)
}

func (s *Store) ListEffectiveRulesForHost(
	ctx context.Context,
	hostID int64,
	params EffectiveRuleListParams,
) ([]EffectiveRuleStatus, int, error) {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	if params.PageSize <= 0 {
		params.PageSize = 100
	}

	var count int
	if err := s.db.Pool().QueryRow(ctx, "SELECT count(*) FROM ("+effectiveRulesForHostSQL+") effective_rules", hostID).
		Scan(&count); err != nil {
		return nil, 0, err
	}
	offset := params.PageIndex * params.PageSize
	rows, err := s.db.Pool().Query(ctx, effectiveRulesForHostSQL+`
		ORDER BY rule_type_sort, identifier, rule_id
		LIMIT $2 OFFSET $3
	`, hostID, params.PageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	rules, err := pgx.CollectRows(rows, scanEffectiveRule)
	if err != nil {
		return nil, 0, err
	}

	targets := SyncTargetsFromRules(rules)
	applied, err := s.appliedSyncTargetSet(ctx, hostID)
	if err != nil {
		return nil, 0, err
	}

	statuses := make([]EffectiveRuleStatus, 0, len(rules))
	for i, rule := range rules {
		target := targets[i]
		appliedRule := applied[syncTargetKey(target)]
		statuses = append(statuses, EffectiveRuleStatus{
			EffectiveRule: rule,
			Applied:       appliedRule,
			PayloadHash:   target.PayloadHash,
		})
	}

	return statuses, count, nil
}

func (s *Store) appliedSyncTargetSet(ctx context.Context, hostID int64) (map[string]bool, error) {
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
		WHERE host_id = $1 AND phase = 'applied'
	`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applied, err := pgx.CollectRows(rows, scanSyncTarget)
	if err != nil {
		return nil, err
	}
	return santasync.TargetSet(applied), nil
}

func replaceRuleChildren(
	ctx context.Context,
	tx pgx.Tx,
	ruleID int64,
	includes []RuleIncludeWrite,
	excludeLabelIDs []int64,
) error {
	if _, err := tx.Exec(ctx, `DELETE FROM santa_rule_exclude_labels WHERE rule_id = $1`, ruleID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM santa_rule_includes WHERE rule_id = $1`, ruleID); err != nil {
		return err
	}
	includeIDs := []int64{}
	if len(includes) > 0 {
		policies := make([]string, len(includes))
		celExpressions := make([]string, len(includes))
		for i, include := range includes {
			policies[i] = string(include.Policy)
			celExpressions[i] = include.CELExpression
		}
		rows, err := tx.Query(ctx, `
			WITH input AS (
				SELECT
					policy,
					cel_expression,
					position
				FROM unnest($2::text[], $3::text[]) WITH ORDINALITY AS input(
					policy,
					cel_expression,
					position
				)
			)
			INSERT INTO santa_rule_includes (rule_id, position, policy, cel_expression)
			SELECT
				$1,
				position - 1,
				policy::santa_policy,
				NULLIF(cel_expression, '')
			FROM input
			ORDER BY position
			RETURNING id
		`, ruleID, policies, celExpressions)
		if err != nil {
			return err
		}
		includeIDs, err = pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil {
			return err
		}
	}
	includeLabelIncludeIDs := []int64{}
	includeLabelLabelIDs := []int64{}
	for position, include := range includes {
		for _, labelID := range include.LabelIDs {
			includeLabelIncludeIDs = append(includeLabelIncludeIDs, includeIDs[position])
			includeLabelLabelIDs = append(includeLabelLabelIDs, labelID)
		}
	}
	if len(includeLabelIncludeIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO santa_rule_include_labels (include_id, label_id)
			SELECT include_id, label_id
			FROM unnest($1::bigint[], $2::bigint[]) AS input(include_id, label_id)
		`, includeLabelIncludeIDs, includeLabelLabelIDs); err != nil {
			return err
		}
	}
	if len(excludeLabelIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO santa_rule_exclude_labels (rule_id, label_id)
		SELECT $1, label_id
		FROM unnest($2::bigint[]) AS label_id
	`, ruleID, excludeLabelIDs)
	return err
}

func (s *Store) attachRuleChildren(ctx context.Context, rules []Rule, ruleIDs []int64) error {
	if len(ruleIDs) == 0 {
		return nil
	}
	ruleIndexes := make(map[int64]int, len(rules))
	for i := range rules {
		ruleIndexes[rules[i].ID] = i
	}

	includes, err := s.loadRuleIncludes(ctx, ruleIDs)
	if err != nil {
		return err
	}
	for ruleID, values := range includes {
		if i, ok := ruleIndexes[ruleID]; ok {
			rules[i].Includes = values
		}
	}

	excludes, err := s.loadRuleExcludeLabels(ctx, ruleIDs)
	if err != nil {
		return err
	}
	for ruleID, values := range excludes {
		if i, ok := ruleIndexes[ruleID]; ok {
			rules[i].ExcludeLabelIDs = values
		}
	}
	return nil
}

func (s *Store) loadRuleIncludes(ctx context.Context, ruleIDs []int64) (map[int64][]RuleInclude, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT
			i.rule_id,
			i.id,
			i.position,
			i.policy::text,
			COALESCE(i.cel_expression, ''),
			COALESCE(array_agg(il.label_id ORDER BY il.label_id) FILTER (WHERE il.label_id IS NOT NULL), ARRAY[]::bigint[])
		FROM santa_rule_includes i
		LEFT JOIN santa_rule_include_labels il ON il.include_id = i.id
		WHERE i.rule_id = ANY($1)
		GROUP BY i.rule_id, i.id
		ORDER BY i.rule_id, i.position, i.id
	`, ruleIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64][]RuleInclude{}
	for rows.Next() {
		var ruleID int64
		var include RuleInclude
		if err := rows.Scan(
			&ruleID,
			&include.ID,
			&include.Position,
			&include.Policy,
			&include.CELExpression,
			&include.LabelIDs,
		); err != nil {
			return nil, err
		}
		out[ruleID] = append(out[ruleID], include)
	}
	return out, rows.Err()
}

func (s *Store) loadRuleExcludeLabels(ctx context.Context, ruleIDs []int64) (map[int64][]int64, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT rule_id, label_id
		FROM santa_rule_exclude_labels
		WHERE rule_id = ANY($1)
		ORDER BY rule_id, label_id
	`, ruleIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64][]int64{}
	for rows.Next() {
		var ruleID int64
		var labelID int64
		if err := rows.Scan(&ruleID, &labelID); err != nil {
			return nil, err
		}
		out[ruleID] = append(out[ruleID], labelID)
	}
	return out, rows.Err()
}

func cleanRuleMutation(params RuleMutation) (RuleMutation, error) {
	params.RuleType = RuleType(strings.TrimSpace(string(params.RuleType)))
	params.Identifier = strings.TrimSpace(params.Identifier)
	params.Name = strings.TrimSpace(params.Name)
	params.CustomMessage = strings.TrimSpace(params.CustomMessage)
	params.CustomURL = strings.TrimSpace(params.CustomURL)
	if !validRuleType(params.RuleType) {
		return RuleMutation{}, fmt.Errorf("%w: unknown rule type", dbutil.ErrInvalidInput)
	}
	if params.Identifier == "" {
		return RuleMutation{}, fmt.Errorf("%w: identifier is required", dbutil.ErrInvalidInput)
	}
	includes, err := cleanRuleIncludes(params.Includes)
	if err != nil {
		return RuleMutation{}, err
	}
	excludeLabelIDs, err := santaids.CleanLabelIDs(params.ExcludeLabelIDs, "exclude_label_ids")
	if err != nil {
		return RuleMutation{}, err
	}
	params.Includes = includes
	params.ExcludeLabelIDs = excludeLabelIDs
	return params, nil
}

func cleanRuleIncludes(includes []RuleIncludeWrite) ([]RuleIncludeWrite, error) {
	cleaned := make([]RuleIncludeWrite, 0, len(includes))
	for _, include := range includes {
		include.Policy = Policy(strings.TrimSpace(string(include.Policy)))
		include.CELExpression = strings.TrimSpace(include.CELExpression)
		if !validPolicy(include.Policy) {
			return nil, fmt.Errorf("%w: unknown policy", dbutil.ErrInvalidInput)
		}
		if include.Policy == PolicyCEL && include.CELExpression == "" {
			return nil, fmt.Errorf("%w: cel_expression is required for cel policy", dbutil.ErrInvalidInput)
		}
		if include.Policy != PolicyCEL && include.CELExpression != "" {
			return nil, fmt.Errorf("%w: cel_expression is only valid for cel policy", dbutil.ErrInvalidInput)
		}
		labelIDs, err := santaids.CleanLabelIDs(include.LabelIDs, "label_ids")
		if err != nil {
			return nil, err
		}
		if len(labelIDs) == 0 {
			return nil, fmt.Errorf("%w: include label_ids must not be empty", dbutil.ErrInvalidInput)
		}
		include.LabelIDs = labelIDs
		cleaned = append(cleaned, include)
	}
	return cleaned, nil
}

func validRuleType(ruleType RuleType) bool {
	_, ok := validRuleTypes[ruleType]
	return ok
}

func validPolicy(policy Policy) bool {
	_, ok := validPolicies[policy]
	return ok
}

func ruleListWhere(params RuleListParams) (string, []any, error) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add("(identifier ILIKE " + search + " OR name ILIKE " + search + ")")
	}
	if params.RuleType != "" {
		if !validRuleType(params.RuleType) {
			return "", nil, fmt.Errorf("%w: unknown rule type", dbutil.ErrInvalidInput)
		}
		where.Add("rule_type = " + where.Arg(params.RuleType))
	}
	whereSQL, args := where.Build()
	return whereSQL, args, nil
}

func ruleListSQL(params RuleListParams, where string, args []any) (string, []any, error) {
	return dbutil.ListQuery{
		SelectSQL:    ruleSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    ruleOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{{SQL: "rule_type_sort"}, {SQL: "identifier"}, {SQL: "id"}},
		Params:       params.ListParams,
	}.Build()
}

func ruleOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"rule_type":  {SQL: "rule_type_sort"},
		"identifier": {SQL: "identifier"},
		"name":       {SQL: "lower(name)"},
		"updated_at": {SQL: "updated_at"},
	}
}

func scanRule(row pgx.Row) (Rule, error) {
	var rule Rule
	err := row.Scan(
		&rule.ID,
		&rule.RuleType,
		&rule.Identifier,
		&rule.Name,
		&rule.CustomMessage,
		&rule.CustomURL,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	return rule, err
}

func ruleFromSQLC(row sqlc.SantaRule) Rule {
	return Rule{
		ID:            row.ID,
		RuleType:      RuleType(row.RuleType),
		Identifier:    row.Identifier,
		Name:          row.Name,
		CustomMessage: row.CustomMessage,
		CustomURL:     row.CustomURL,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func scanEffectiveRule(row pgx.CollectableRow) (EffectiveRule, error) {
	var rule EffectiveRule
	var ignoredSort int
	err := row.Scan(
		&rule.RuleID,
		&rule.RuleType,
		&rule.Identifier,
		&rule.Policy,
		&rule.CELExpression,
		&rule.CustomMessage,
		&rule.CustomURL,
		&rule.MatchedIncludeID,
		&ignoredSort,
	)
	return rule, err
}

// SyncTargetsFromRules returns Santa sync payload targets for effective rules.
func SyncTargetsFromRules(rules []EffectiveRule) []santasync.Target {
	targets := make([]santasync.Target, 0, len(rules))
	for _, rule := range rules {
		target := santasync.Target{
			RuleType:      string(rule.RuleType),
			Identifier:    rule.Identifier,
			Policy:        string(rule.Policy),
			CELExpression: rule.CELExpression,
			CustomMessage: rule.CustomMessage,
			CustomURL:     rule.CustomURL,
		}
		target.PayloadHash = syncTargetPayloadHash(target)
		targets = append(targets, target)
	}
	return targets
}

func syncTargetPayloadHash(target santasync.Target) string {
	var payload strings.Builder
	payload.WriteString("v1")
	writeHashField(&payload, target.RuleType)
	writeHashField(&payload, target.Identifier)
	writeHashField(&payload, target.Policy)
	writeHashField(&payload, target.CELExpression)
	writeHashField(&payload, target.CustomMessage)
	writeHashField(&payload, target.CustomURL)
	sum := sha256.Sum256([]byte(payload.String()))
	return hex.EncodeToString(sum[:])
}

func writeHashField(payload *strings.Builder, value string) {
	payload.WriteByte(0)
	payload.WriteString(strconv.Itoa(len(value)))
	payload.WriteByte(':')
	payload.WriteString(value)
}

func syncTargetKey(target santasync.Target) string {
	return target.RuleType + "\x00" + target.Identifier + "\x00" + target.PayloadHash
}

func scanSyncTarget(row pgx.CollectableRow) (santasync.Target, error) {
	var target santasync.Target
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

const ruleTypeSortSQL = `CASE r.rule_type
	WHEN 'cdhash' THEN 1
	WHEN 'binary' THEN 2
	WHEN 'signingid' THEN 3
	WHEN 'certificate' THEN 4
	WHEN 'teamid' THEN 5
	ELSE 6
END`

const ruleSelectSQL = `
SELECT
	id,
	rule_type::text,
	identifier,
	name,
	custom_message,
	custom_url,
	created_at,
	updated_at
FROM (
	SELECT
		r.*,
		` + ruleTypeSortSQL + ` AS rule_type_sort
	FROM santa_rules r
) sorted_rules`

const effectiveRulesForHostSQL = `
WITH host_labels AS (
	SELECT label_id
	FROM label_membership
	WHERE host_id = $1
),
matching_includes AS (
	SELECT
		r.id AS rule_id,
		r.rule_type,
		r.identifier,
		i.policy,
		COALESCE(i.cel_expression, '') AS cel_expression,
		r.custom_message,
		r.custom_url,
		i.id AS matched_include_id,
		` + ruleTypeSortSQL + ` AS rule_type_sort,
		row_number() OVER (PARTITION BY r.id ORDER BY i.position, i.id) AS include_rank
	FROM santa_rules r
	JOIN santa_rule_includes i ON i.rule_id = r.id
	WHERE EXISTS (
		SELECT 1
		FROM santa_rule_include_labels il
		JOIN host_labels hl ON hl.label_id = il.label_id
		WHERE il.include_id = i.id
	)
	AND NOT EXISTS (
		SELECT 1
		FROM santa_rule_exclude_labels el
		JOIN host_labels hl ON hl.label_id = el.label_id
		WHERE el.rule_id = r.id
	)
)
SELECT
	rule_id,
	rule_type::text,
	identifier,
	policy::text,
	cel_expression,
	custom_message,
	custom_url,
	matched_include_id,
	rule_type_sort
FROM matching_includes
WHERE include_rank = 1`
