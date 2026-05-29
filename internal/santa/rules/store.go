package rules

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

// Store persists Santa rule definitions and resolves effective rule state.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

func (s *Store) ListRules(ctx context.Context, params RuleListParams) ([]Rule, int, error) {
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

func (s *Store) ListRuleTargets(ctx context.Context, params RuleTargetListParams) ([]RuleTarget, error) {
	params.Q = strings.TrimSpace(params.Q)
	if params.TargetType != "" && !validRuleType(params.TargetType) {
		return nil, fmt.Errorf("%w: unknown target_type", dbutil.ErrInvalidInput)
	}
	if params.Limit <= 0 || params.Limit > 50 {
		params.Limit = 20
	}
	rows, err := s.db.Pool().Query(ctx, ruleTargetSearchSQL, params.Q, string(params.TargetType), params.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, scanRuleTarget)
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
	params = params.Clean()
	if err := params.Validate(); err != nil {
		return nil, err
	}

	var ruleID int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := validateRuleTargetLabels(ctx, tx, params); err != nil {
			return err
		}
		if err := validateRuleTarget(ctx, tx, params); err != nil {
			return err
		}
		row, err := s.q.WithTx(tx).CreateSantaRule(ctx, sqlc.CreateSantaRuleParams{
			RuleType:      sqlc.SantaRuleType(params.RuleType),
			Identifier:    params.Identifier,
			Name:          params.Name,
			CustomMessage: params.CustomMessage,
			CustomURL:     params.CustomURL,
		})
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		ruleID = row.ID
		return replaceRuleChildren(ctx, tx, ruleID, params.Includes, params.ExcludeLabelIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.GetRuleByID(ctx, ruleID)
}

func (s *Store) UpdateRule(ctx context.Context, id int64, params RuleMutation) (*Rule, error) {
	params = params.Clean()
	if err := params.Validate(); err != nil {
		return nil, err
	}

	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := validateRuleTargetLabels(ctx, tx, params); err != nil {
			return err
		}
		if err := validateRuleTarget(ctx, tx, params); err != nil {
			return err
		}
		row, err := s.q.WithTx(tx).UpdateSantaRule(ctx, sqlc.UpdateSantaRuleParams{
			RuleType:      sqlc.SantaRuleType(params.RuleType),
			Identifier:    params.Identifier,
			Name:          params.Name,
			CustomMessage: params.CustomMessage,
			CustomURL:     params.CustomURL,
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
		return replaceRuleChildren(ctx, tx, row.ID, params.Includes, params.ExcludeLabelIDs)
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
		if !dbutil.SameInt64Set(orderedIncludeIDs, currentIDs) {
			return fmt.Errorf("%w: ordered_include_ids must exactly match existing include IDs", dbutil.ErrInvalidInput)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE santa_rule_includes i
			SET position = -ordered.position
			FROM unnest($1::bigint[]) WITH ORDINALITY AS ordered(id, position)
			WHERE i.id = ordered.id AND i.rule_id = $2
		`, orderedIncludeIDs, ruleID); err != nil {
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
			notification_app_name,
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
	return syncstate.TargetSet(applied), nil
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
	if len(includes) > 0 {
		policies := make([]string, len(includes))
		celExpressions := make([]string, len(includes))
		labelIDs := make([]int64, len(includes))
		for i, include := range includes {
			policies[i] = string(include.Policy)
			celExpressions[i] = include.CELExpression
			labelIDs[i] = include.LabelID
		}
		if _, err := tx.Exec(ctx, `
			WITH input AS (
				SELECT
					policy,
					cel_expression,
					label_id,
					position
				FROM unnest($2::text[], $3::text[], $4::bigint[]) WITH ORDINALITY AS input(
					policy,
					cel_expression,
					label_id,
					position
				)
			)
			INSERT INTO santa_rule_includes (rule_id, position, policy, cel_expression, label_id)
			SELECT
				$1,
				position - 1,
				policy::santa_policy,
				NULLIF(cel_expression, ''),
				label_id
			FROM input
			ORDER BY position
		`, ruleID, policies, celExpressions, labelIDs); err != nil {
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
			i.label_id
		FROM santa_rule_includes i
		WHERE i.rule_id = ANY($1)
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
			&include.LabelID,
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

func (p RuleMutation) Validate() error {
	if !validRuleType(p.RuleType) {
		return fmt.Errorf("%w: rule_type is required", dbutil.ErrInvalidInput)
	}
	if p.Identifier == "" {
		return fmt.Errorf("%w: identifier is required", dbutil.ErrInvalidInput)
	}
	labelIDs := make(map[int64]struct{}, len(p.Includes)+len(p.ExcludeLabelIDs))
	for _, include := range p.Includes {
		if !validPolicy(include.Policy) {
			return fmt.Errorf("%w: include policy is required", dbutil.ErrInvalidInput)
		}
		if include.Policy == PolicyCEL && include.CELExpression == "" {
			return fmt.Errorf("%w: cel_expression is required for cel policy", dbutil.ErrInvalidInput)
		}
		if include.Policy != PolicyCEL && include.CELExpression != "" {
			return fmt.Errorf("%w: cel_expression is only valid for cel policy", dbutil.ErrInvalidInput)
		}
		if include.LabelID == 0 {
			return fmt.Errorf("%w: include label_id is required", dbutil.ErrInvalidInput)
		}
		if _, ok := labelIDs[include.LabelID]; ok {
			return fmt.Errorf("%w: label_id is already assigned to this rule", dbutil.ErrInvalidInput)
		}
		labelIDs[include.LabelID] = struct{}{}
	}
	for _, labelID := range p.ExcludeLabelIDs {
		if _, ok := labelIDs[labelID]; ok {
			return fmt.Errorf("%w: label_id is already assigned to this rule", dbutil.ErrInvalidInput)
		}
		labelIDs[labelID] = struct{}{}
	}
	return nil
}

func (p RuleMutation) Clean() RuleMutation {
	p.Identifier = strings.TrimSpace(p.Identifier)
	p.Name = strings.TrimSpace(p.Name)
	p.CustomMessage = strings.TrimSpace(p.CustomMessage)
	p.CustomURL = strings.TrimSpace(p.CustomURL)
	for i := range p.Includes {
		p.Includes[i].CELExpression = strings.TrimSpace(p.Includes[i].CELExpression)
	}
	return p
}

func validRuleType(ruleType RuleType) bool {
	return slices.Contains(RuleTypeValues, ruleType)
}

func validPolicy(policy Policy) bool {
	return slices.Contains(PolicyValues, policy)
}

func validateRuleTargetLabels(ctx context.Context, tx pgx.Tx, params RuleMutation) error {
	if len(params.ExcludeLabelIDs) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, `
		SELECT id
		FROM labels
		WHERE id = ANY($1::bigint[]) AND label_type = $2
	`, params.ExcludeLabelIDs, labels.LabelTypeBuiltin)
	if err != nil {
		return err
	}
	builtinExcludeIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return err
	}
	if len(builtinExcludeIDs) > 0 {
		return fmt.Errorf("%w: builtin labels cannot be excluded from Santa rules", dbutil.ErrInvalidInput)
	}
	return nil
}

func validateRuleTarget(ctx context.Context, tx pgx.Tx, params RuleMutation) error {
	if params.RuleType != RuleTypeBundle {
		return nil
	}
	var complete bool
	err := tx.QueryRow(ctx, `
		SELECT uploaded_at IS NOT NULL
		FROM santa_bundles
		WHERE sha256 = $1
	`, params.Identifier).Scan(&complete)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: bundle target is not collected", dbutil.ErrInvalidInput)
	}
	if err != nil {
		return err
	}
	if !complete {
		return fmt.Errorf("%w: bundle target is incomplete", dbutil.ErrInvalidInput)
	}
	return nil
}

func ruleListWhere(params RuleListParams) (string, []any, error) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add("(identifier ILIKE " + search + " OR name ILIKE " + search + ")")
	}
	if params.RuleType != "" {
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

func scanRuleTarget(row pgx.CollectableRow) (RuleTarget, error) {
	var target RuleTarget
	err := row.Scan(
		&target.TargetType,
		&target.Identifier,
		&target.Name,
		&target.Detail,
		&target.BundleID,
		&target.Version,
		&target.BinaryCount,
		&target.CollectedBinaryCount,
		&target.RuleCount,
		&target.Complete,
	)
	return target, err
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
		&rule.AppName,
		&rule.MatchedIncludeID,
		&ignoredSort,
	)
	return rule, err
}

// SyncTargetsFromRules returns Santa sync payload targets for effective rules.
func SyncTargetsFromRules(rules []EffectiveRule) []syncstate.Target {
	targets := make([]syncstate.Target, 0, len(rules))
	for _, rule := range rules {
		target := syncstate.Target{
			RuleType:      string(rule.RuleType),
			Identifier:    rule.Identifier,
			Policy:        string(rule.Policy),
			CELExpression: rule.CELExpression,
			CustomMessage: rule.CustomMessage,
			CustomURL:     rule.CustomURL,
			AppName:       rule.AppName,
		}
		target.PayloadHash = syncTargetPayloadHash(target)
		targets = append(targets, target)
	}
	return targets
}

func syncTargetPayloadHash(target syncstate.Target) string {
	return syncstate.PayloadHash(
		target.RuleType,
		target.Identifier,
		target.Policy,
		target.CELExpression,
		target.CustomMessage,
		target.CustomURL,
		target.AppName,
	)
}

func syncTargetKey(target syncstate.Target) string {
	return target.RuleType + "\x00" + target.Identifier + "\x00" + target.PayloadHash
}

func scanSyncTarget(row pgx.CollectableRow) (syncstate.Target, error) {
	var target syncstate.Target
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

const ruleTypeSortSQL = `CASE r.rule_type
	WHEN 'cdhash' THEN 1
	WHEN 'binary' THEN 2
	WHEN 'signingid' THEN 3
	WHEN 'certificate' THEN 4
	WHEN 'teamid' THEN 5
	WHEN 'bundle' THEN 6
	ELSE 7
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
	JOIN host_labels include_hl ON include_hl.label_id = i.label_id
	WHERE NOT EXISTS (
		SELECT 1
		FROM santa_rule_exclude_labels el
		JOIN host_labels hl ON hl.label_id = el.label_id
		WHERE el.rule_id = r.id
	)
),
selected_includes AS (
	SELECT *
	FROM matching_includes
	WHERE include_rank = 1
),
expanded_rules AS (
	SELECT
		rule_id,
		rule_type,
		identifier,
		policy,
		cel_expression,
		custom_message,
		custom_url,
		''::text AS notification_app_name,
		matched_include_id,
		rule_type_sort
	FROM selected_includes
	WHERE rule_type <> 'bundle'
	UNION ALL
	SELECT
		si.rule_id,
		'binary'::santa_rule_type AS rule_type,
		e.sha256 AS identifier,
		si.policy,
		si.cel_expression,
		si.custom_message,
		si.custom_url,
		COALESCE(NULLIF(b.name, ''), NULLIF(b.bundle_id, ''), b.sha256) AS notification_app_name,
		si.matched_include_id,
		2 AS rule_type_sort
	FROM selected_includes si
	JOIN santa_bundles b ON b.sha256 = si.identifier AND b.uploaded_at IS NOT NULL
	JOIN santa_bundle_executables be ON be.bundle_id = b.id
	JOIN santa_executables e ON e.id = be.executable_id
	WHERE si.rule_type = 'bundle'
		AND e.sha256 <> ''
)
SELECT
	rule_id,
	rule_type::text,
	identifier,
	policy::text,
	cel_expression,
	custom_message,
	custom_url,
	notification_app_name,
	matched_include_id,
	rule_type_sort
FROM expanded_rules`

const ruleTargetSearchSQL = `
WITH candidate_sources AS (
	SELECT
		'binary'::text AS target_type,
		e.sha256 AS identifier,
		COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, ''), e.sha256) AS name,
		NULLIF(e.file_name, '') AS detail,
		e.file_bundle_id AS bundle_id,
		COALESCE(NULLIF(e.file_bundle_version_string, ''), e.file_bundle_version) AS version,
		0::integer AS binary_count,
		0::integer AS collected_binary_count,
		true AS complete
	FROM santa_executables e
	WHERE e.sha256 <> ''
	UNION ALL
	SELECT
		'binary'::text,
		p.executable_sha256,
		COALESCE(NULLIF(st.display_name, ''), st.name, p.executable_sha256),
		COALESCE(NULLIF(p.executable_path, ''), p.installed_path),
		s.bundle_identifier,
		s.version,
		0::integer,
		0::integer,
		true
	FROM host_software_installed_paths p
	JOIN software s ON s.id = p.software_id
	JOIN software_titles st ON st.id = s.title_id
	WHERE p.executable_sha256 IS NOT NULL AND p.executable_sha256 <> ''
	UNION ALL
	SELECT
		'certificate'::text,
		c.sha256,
		COALESCE(NULLIF(c.common_name, ''), c.sha256),
		c.organizational_unit,
		''::text,
		''::text,
		0::integer,
		0::integer,
		true
	FROM santa_certificates c
	WHERE c.sha256 <> ''
	UNION ALL
	SELECT
		'teamid'::text,
		e.team_id,
		e.team_id,
		COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, '')),
		e.file_bundle_id,
		COALESCE(NULLIF(e.file_bundle_version_string, ''), e.file_bundle_version),
		0::integer,
		0::integer,
		true
	FROM santa_executables e
	WHERE e.team_id <> ''
	UNION ALL
	SELECT
		'teamid'::text,
		p.team_identifier,
		p.team_identifier,
		COALESCE(NULLIF(st.display_name, ''), st.name),
		s.bundle_identifier,
		s.version,
		0::integer,
		0::integer,
		true
	FROM host_software_installed_paths p
	JOIN software s ON s.id = p.software_id
	JOIN software_titles st ON st.id = s.title_id
	WHERE p.team_identifier <> ''
	UNION ALL
	SELECT
		'signingid'::text,
		e.signing_id,
		e.signing_id,
		COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, '')),
		e.file_bundle_id,
		COALESCE(NULLIF(e.file_bundle_version_string, ''), e.file_bundle_version),
		0::integer,
		0::integer,
		true
	FROM santa_executables e
	WHERE e.signing_id <> ''
	UNION ALL
	SELECT
		'signingid'::text,
		p.team_identifier || ':' || s.bundle_identifier,
		p.team_identifier || ':' || s.bundle_identifier,
		COALESCE(NULLIF(st.display_name, ''), st.name),
		s.bundle_identifier,
		s.version,
		0::integer,
		0::integer,
		true
	FROM host_software_installed_paths p
	JOIN software s ON s.id = p.software_id
	JOIN software_titles st ON st.id = s.title_id
	WHERE p.team_identifier <> '' AND s.bundle_identifier <> ''
	UNION ALL
	SELECT
		'cdhash'::text,
		e.cdhash,
		e.cdhash,
		COALESCE(NULLIF(e.file_bundle_name, ''), NULLIF(e.file_name, '')),
		e.file_bundle_id,
		COALESCE(NULLIF(e.file_bundle_version_string, ''), e.file_bundle_version),
		0::integer,
		0::integer,
		true
	FROM santa_executables e
	WHERE e.cdhash <> ''
	UNION ALL
	SELECT
		'cdhash'::text,
		p.cdhash_sha256,
		p.cdhash_sha256,
		COALESCE(NULLIF(st.display_name, ''), st.name),
		s.bundle_identifier,
		s.version,
		0::integer,
		0::integer,
		true
	FROM host_software_installed_paths p
	JOIN software s ON s.id = p.software_id
	JOIN software_titles st ON st.id = s.title_id
	WHERE p.cdhash_sha256 IS NOT NULL AND p.cdhash_sha256 <> ''
	UNION ALL
	SELECT
		'bundle'::text,
		b.sha256,
		COALESCE(NULLIF(b.name, ''), NULLIF(b.bundle_id, ''), b.sha256),
		b.path,
		b.bundle_id,
		COALESCE(NULLIF(b.version_string, ''), b.version),
		b.binary_count,
		COUNT(be.executable_id)::integer,
		b.uploaded_at IS NOT NULL
	FROM santa_bundles b
	LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
	WHERE b.sha256 <> ''
	GROUP BY b.id
),
targets AS (
	SELECT
		target_type,
		identifier,
		COALESCE(NULLIF(max(name), ''), identifier) AS name,
		COALESCE(max(detail), '') AS detail,
		COALESCE(max(bundle_id), '') AS bundle_id,
		COALESCE(max(version), '') AS version,
		max(binary_count)::integer AS binary_count,
		max(collected_binary_count)::integer AS collected_binary_count,
		bool_or(complete) AS complete
	FROM candidate_sources
	WHERE identifier <> ''
	GROUP BY target_type, identifier
)
SELECT
	t.target_type,
	t.identifier,
	t.name,
	t.detail,
	t.bundle_id,
	t.version,
	t.binary_count,
	t.collected_binary_count,
	COUNT(r.id)::integer AS rule_count,
	t.complete
FROM targets t
LEFT JOIN santa_rules r
	ON r.rule_type::text = t.target_type AND r.identifier = t.identifier
WHERE
	($1 = '' OR t.identifier ILIKE '%' || $1 || '%' OR t.name ILIKE '%' || $1 || '%'
		OR t.detail ILIKE '%' || $1 || '%' OR t.bundle_id ILIKE '%' || $1 || '%')
	AND ($2 = '' OR t.target_type = $2)
GROUP BY
	t.target_type,
	t.identifier,
	t.name,
	t.detail,
	t.bundle_id,
	t.version,
	t.binary_count,
	t.collected_binary_count,
	t.complete
ORDER BY
	CASE t.target_type
		WHEN 'bundle' THEN 1
		WHEN 'signingid' THEN 2
		WHEN 'teamid' THEN 3
		WHEN 'certificate' THEN 4
		WHEN 'binary' THEN 5
		WHEN 'cdhash' THEN 6
		ELSE 7
	END,
	lower(t.name),
	t.identifier
LIMIT $3`
