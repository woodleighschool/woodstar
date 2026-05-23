package santa

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

type RuleType string

const (
	RuleTypeBinary      RuleType = "binary"
	RuleTypeCertificate RuleType = "certificate"
	RuleTypeTeamID      RuleType = "teamid"
	RuleTypeSigningID   RuleType = "signingid"
	RuleTypeCDHash      RuleType = "cdhash"
)

type Policy string

const (
	PolicyAllowlist         Policy = "allowlist"
	PolicyAllowlistCompiler Policy = "allowlist_compiler"
	PolicyBlocklist         Policy = "blocklist"
	PolicySilentBlocklist   Policy = "silent_blocklist"
	PolicyCEL               Policy = "cel"
)

type RuleListParams struct {
	dbutil.ListParams

	RuleType RuleType
}

type RuleCreate struct {
	RuleType        RuleType           `json:"rule_type"`
	Identifier      string             `json:"identifier"`
	Name            string             `json:"name,omitempty"`
	CustomMessage   string             `json:"custom_message,omitempty"`
	CustomURL       string             `json:"custom_url,omitempty"`
	Includes        []RuleIncludeWrite `json:"includes,omitempty"`
	ExcludeLabelIDs []int64            `json:"exclude_label_ids,omitempty"`
}

type RuleUpdate struct {
	Name            string             `json:"name,omitempty"`
	CustomMessage   string             `json:"custom_message,omitempty"`
	CustomURL       string             `json:"custom_url,omitempty"`
	Includes        []RuleIncludeWrite `json:"includes,omitempty"`
	ExcludeLabelIDs []int64            `json:"exclude_label_ids,omitempty"`
}

type RuleIncludeWrite struct {
	Policy        Policy  `json:"policy"`
	CELExpression string  `json:"cel_expression,omitempty"`
	LabelIDs      []int64 `json:"label_ids"`
}

type Rule struct {
	ID              int64         `json:"id"`
	RuleType        RuleType      `json:"rule_type"`
	Identifier      string        `json:"identifier"`
	Name            string        `json:"name"`
	CustomMessage   string        `json:"custom_message"`
	CustomURL       string        `json:"custom_url"`
	Includes        []RuleInclude `json:"includes"`
	ExcludeLabelIDs []int64       `json:"exclude_label_ids"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type RuleInclude struct {
	ID            int64   `json:"id"`
	Position      int     `json:"position"`
	Policy        Policy  `json:"policy"`
	CELExpression string  `json:"cel_expression,omitempty"`
	LabelIDs      []int64 `json:"label_ids"`
}

type EffectiveRule struct {
	RuleID           int64    `json:"rule_id"`
	RuleType         RuleType `json:"rule_type"`
	Identifier       string   `json:"identifier"`
	Policy           Policy   `json:"policy"`
	CELExpression    string   `json:"cel_expression,omitempty"`
	CustomMessage    string   `json:"custom_message,omitempty"`
	CustomURL        string   `json:"custom_url,omitempty"`
	MatchedIncludeID int64    `json:"matched_include_id"`
}

type EffectiveRuleStatus struct {
	EffectiveRule
	Applied     bool   `json:"applied"`
	Pending     bool   `json:"pending"`
	PayloadHash string `json:"payload_hash"`
}

type EffectiveRuleListParams struct {
	dbutil.ListParams
}

func (s *Store) ListRules(ctx context.Context, params RuleListParams) ([]Rule, int, error) {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	params.RuleType = cleanRuleType(params.RuleType)
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
	var rule Rule
	err := s.db.Pool().QueryRow(ctx, ruleSelectSQL+" WHERE id = $1", id).Scan(
		&rule.ID,
		&rule.RuleType,
		&rule.Identifier,
		&rule.Name,
		&rule.CustomMessage,
		&rule.CustomURL,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *Store) CreateRule(ctx context.Context, params RuleCreate) (*Rule, error) {
	cleaned, err := cleanRuleCreate(params)
	if err != nil {
		return nil, err
	}

	var ruleID int64
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO santa_rules (rule_type, identifier, name, custom_message, custom_url)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, cleaned.RuleType, cleaned.Identifier, cleaned.Name, cleaned.CustomMessage, cleaned.CustomURL).Scan(&ruleID)
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		return replaceRuleChildren(ctx, tx, ruleID, cleaned.Includes, cleaned.ExcludeLabelIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.GetRuleByID(ctx, ruleID)
}

func (s *Store) UpdateRule(ctx context.Context, id int64, params RuleUpdate) (*Rule, error) {
	cleaned, err := cleanRuleUpdate(params)
	if err != nil {
		return nil, err
	}

	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var ruleID int64
		err := tx.QueryRow(ctx, `
			UPDATE santa_rules
			SET name = $1, custom_message = $2, custom_url = $3, updated_at = now()
			WHERE id = $4
			RETURNING id
		`, cleaned.Name, cleaned.CustomMessage, cleaned.CustomURL, id).Scan(&ruleID)
		if errors.Is(err, pgx.ErrNoRows) {
			return dbutil.ErrNotFound
		}
		if err != nil {
			return err
		}
		return replaceRuleChildren(ctx, tx, ruleID, cleaned.Includes, cleaned.ExcludeLabelIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.GetRuleByID(ctx, id)
}

func (s *Store) DeleteRule(ctx context.Context, id int64) error {
	var deletedID int64
	err := s.db.Pool().QueryRow(ctx, `
		DELETE FROM santa_rules
		WHERE id = $1
		RETURNING id
	`, id).Scan(&deletedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

func (s *Store) ReorderRuleIncludes(ctx context.Context, ruleID int64, orderedIncludeIDs []int64) error {
	if ruleID <= 0 {
		return dbutil.ErrNotFound
	}
	ids, err := parsePositiveIDs(orderedIncludeIDs, "ordered_include_ids")
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
		if !sameIDSet(ids, currentIDs) {
			return fmt.Errorf("%w: ordered_include_ids must exactly match existing include IDs", dbutil.ErrInvalidInput)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE santa_rule_includes
			SET position = position + 100000
			WHERE rule_id = $1
		`, ruleID); err != nil {
			return err
		}
		for position, id := range ids {
			if _, err := tx.Exec(ctx, `
				UPDATE santa_rule_includes
				SET position = $1
				WHERE id = $2 AND rule_id = $3
			`, position, id, ruleID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ResolveRulesForHost(ctx context.Context, hostID int64) ([]EffectiveRule, error) {
	hostLabels, err := s.hostLabelSet(ctx, hostID)
	if err != nil {
		return nil, err
	}
	rules, _, err := s.ListRules(ctx, RuleListParams{
		ListParams: dbutil.ListParams{PerPage: 10000},
	})
	if err != nil {
		return nil, err
	}

	effective := []EffectiveRule{}
	for _, rule := range rules {
		if hasAnyLabel(hostLabels, rule.ExcludeLabelIDs) {
			continue
		}
		for _, include := range rule.Includes {
			if !hasAnyLabel(hostLabels, include.LabelIDs) {
				continue
			}
			effective = append(effective, EffectiveRule{
				RuleID:           rule.ID,
				RuleType:         rule.RuleType,
				Identifier:       rule.Identifier,
				Policy:           include.Policy,
				CELExpression:    include.CELExpression,
				CustomMessage:    rule.CustomMessage,
				CustomURL:        rule.CustomURL,
				MatchedIncludeID: include.ID,
			})
			break
		}
	}
	return effective, nil
}

func (s *Store) ListEffectiveRulesForHost(
	ctx context.Context,
	hostID int64,
	params EffectiveRuleListParams,
) ([]EffectiveRuleStatus, int, error) {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	if params.PerPage <= 0 {
		params.PerPage = 100
	}

	rules, err := s.ResolveRulesForHost(ctx, hostID)
	if err != nil {
		return nil, 0, err
	}
	targets := syncTargetsFromRules(rules)
	applied, err := s.appliedSyncTargetSet(ctx, hostID)
	if err != nil {
		return nil, 0, err
	}

	rows := make([]EffectiveRuleStatus, 0, len(rules))
	for i, rule := range rules {
		target := targets[i]
		fingerprint := syncTargetFingerprint{
			RuleType:    target.RuleType,
			Identifier:  target.Identifier,
			PayloadHash: target.PayloadHash,
		}
		appliedRule := applied[fingerprint.key()]
		rows = append(rows, EffectiveRuleStatus{
			EffectiveRule: rule,
			Applied:       appliedRule,
			Pending:       !appliedRule,
			PayloadHash:   target.PayloadHash,
		})
	}

	count := len(rows)
	start := (params.Page - 1) * params.PerPage
	if start >= count {
		return []EffectiveRuleStatus{}, count, nil
	}
	end := min(start+params.PerPage, count)
	return rows[start:end], count, nil
}

func (s *Store) appliedSyncTargetSet(ctx context.Context, hostID int64) (map[string]bool, error) {
	var appliedPayload []byte
	err := s.db.Pool().QueryRow(ctx, `
		SELECT COALESCE(applied_targets, '[]'::jsonb)
		FROM santa_sync_state
		WHERE host_id = $1
	`, hostID).Scan(&appliedPayload)
	if errors.Is(err, pgx.ErrNoRows) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}
	applied, err := decodeSyncTargets(appliedPayload)
	if err != nil {
		return nil, err
	}
	return syncTargetSet(applied), nil
}

func (s *Store) hostLabelSet(ctx context.Context, hostID int64) (map[int64]bool, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT label_id
		FROM label_membership
		WHERE host_id = $1
	`, hostID)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, err
	}
	out := make(map[int64]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
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
	for position, include := range includes {
		var includeID int64
		celExpression := dbutil.CleanStringPtr(&include.CELExpression)
		err := tx.QueryRow(ctx, `
			INSERT INTO santa_rule_includes (rule_id, position, policy, cel_expression)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, ruleID, position, include.Policy, celExpression).Scan(&includeID)
		if err != nil {
			return err
		}
		for _, labelID := range include.LabelIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO santa_rule_include_labels (include_id, label_id)
				VALUES ($1, $2)
			`, includeID, labelID); err != nil {
				return err
			}
		}
	}
	for _, labelID := range excludeLabelIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO santa_rule_exclude_labels (rule_id, label_id)
			VALUES ($1, $2)
		`, ruleID, labelID); err != nil {
			return err
		}
	}
	return nil
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

func cleanRuleCreate(params RuleCreate) (RuleCreate, error) {
	params.RuleType = cleanRuleType(params.RuleType)
	params.Identifier = strings.TrimSpace(params.Identifier)
	params.Name = strings.TrimSpace(params.Name)
	params.CustomMessage = strings.TrimSpace(params.CustomMessage)
	params.CustomURL = strings.TrimSpace(params.CustomURL)
	if !validRuleType(params.RuleType) {
		return RuleCreate{}, fmt.Errorf("%w: unknown rule type", dbutil.ErrInvalidInput)
	}
	if params.Identifier == "" {
		return RuleCreate{}, fmt.Errorf("%w: identifier is required", dbutil.ErrInvalidInput)
	}
	includes, err := cleanRuleIncludes(params.Includes)
	if err != nil {
		return RuleCreate{}, err
	}
	excludeLabelIDs, err := cleanLabelIDs(params.ExcludeLabelIDs, "exclude_label_ids")
	if err != nil {
		return RuleCreate{}, err
	}
	params.Includes = includes
	params.ExcludeLabelIDs = excludeLabelIDs
	return params, nil
}

func cleanRuleUpdate(params RuleUpdate) (RuleUpdate, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.CustomMessage = strings.TrimSpace(params.CustomMessage)
	params.CustomURL = strings.TrimSpace(params.CustomURL)
	includes, err := cleanRuleIncludes(params.Includes)
	if err != nil {
		return RuleUpdate{}, err
	}
	excludeLabelIDs, err := cleanLabelIDs(params.ExcludeLabelIDs, "exclude_label_ids")
	if err != nil {
		return RuleUpdate{}, err
	}
	params.Includes = includes
	params.ExcludeLabelIDs = excludeLabelIDs
	return params, nil
}

func cleanRuleIncludes(includes []RuleIncludeWrite) ([]RuleIncludeWrite, error) {
	cleaned := make([]RuleIncludeWrite, 0, len(includes))
	for _, include := range includes {
		include.Policy = cleanPolicy(include.Policy)
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
		labelIDs, err := cleanLabelIDs(include.LabelIDs, "label_ids")
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

func cleanLabelIDs(ids []int64, name string) ([]int64, error) {
	ids, err := parsePositiveIDs(ids, name)
	if err != nil {
		return nil, err
	}
	slices.Sort(ids)
	return slices.Compact(ids), nil
}

func parsePositiveIDs(ids []int64, name string) ([]int64, error) {
	out := make([]int64, len(ids))
	for i, id := range ids {
		if id <= 0 {
			return nil, fmt.Errorf("%w: %s includes a non-positive ID", dbutil.ErrInvalidInput, name)
		}
		out[i] = id
	}
	return out, nil
}

func sameIDSet(a []int64, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	a = slices.Clone(a)
	b = slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func cleanRuleType(ruleType RuleType) RuleType {
	return RuleType(strings.TrimSpace(string(ruleType)))
}

func cleanPolicy(policy Policy) Policy {
	return Policy(strings.TrimSpace(string(policy)))
}

func validRuleType(ruleType RuleType) bool {
	switch ruleType {
	case RuleTypeBinary, RuleTypeCertificate, RuleTypeTeamID, RuleTypeSigningID, RuleTypeCDHash:
		return true
	default:
		return false
	}
}

func validPolicy(policy Policy) bool {
	switch policy {
	case PolicyAllowlist, PolicyAllowlistCompiler, PolicyBlocklist, PolicySilentBlocklist, PolicyCEL:
		return true
	default:
		return false
	}
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

func hasAnyLabel(hostLabels map[int64]bool, labelIDs []int64) bool {
	for _, labelID := range labelIDs {
		if hostLabels[labelID] {
			return true
		}
	}
	return false
}

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
		CASE r.rule_type
			WHEN 'cdhash' THEN 1
			WHEN 'binary' THEN 2
			WHEN 'signingid' THEN 3
			WHEN 'certificate' THEN 4
			WHEN 'teamid' THEN 5
			ELSE 6
		END AS rule_type_sort
	FROM santa_rules r
) sorted_rules`
