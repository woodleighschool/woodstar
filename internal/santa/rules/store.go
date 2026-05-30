package rules

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

// Store persists Santa rule definitions and resolves host rule state.
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
	if params.TargetType != "" && !validRuleType(params.TargetType) {
		return nil, fmt.Errorf("%w: unknown target_type", dbutil.ErrInvalidInput)
	}
	if params.Limit <= 0 || params.Limit > 50 {
		params.Limit = 20
	}
	rows, err := s.q.ListSantaRuleTargets(ctx, sqlc.ListSantaRuleTargetsParams{
		Q:          params.Q,
		TargetType: string(params.TargetType),
		LimitCount: int32(params.Limit),
	})
	if err != nil {
		return nil, err
	}
	targets := make([]RuleTarget, len(rows))
	for i, row := range rows {
		targets[i] = ruleTargetFromSQLC(row)
	}
	return targets, nil
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
			Description:   params.Description,
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
			Description:   params.Description,
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
		q := s.q.WithTx(tx)
		exists, err := q.SantaRuleExists(ctx, sqlc.SantaRuleExistsParams{ID: ruleID})
		if err != nil {
			return err
		}
		if !exists {
			return dbutil.ErrNotFound
		}

		currentIDs, err := q.ListSantaRuleIncludeIDs(ctx, sqlc.ListSantaRuleIncludeIDsParams{RuleID: ruleID})
		if err != nil {
			return err
		}
		if !dbutil.SameInt64Set(orderedIncludeIDs, currentIDs) {
			return fmt.Errorf("%w: ordered_include_ids must exactly match existing include IDs", dbutil.ErrInvalidInput)
		}
		if err := q.SetSantaRuleIncludePositions(ctx, sqlc.SetSantaRuleIncludePositionsParams{
			RuleID:     ruleID,
			OrderedIds: orderedIncludeIDs,
		}); err != nil {
			return err
		}
		return q.NormalizeSantaRuleIncludePositions(
			ctx,
			sqlc.NormalizeSantaRuleIncludePositionsParams{RuleID: ruleID},
		)
	})
}

func (s *Store) ResolveRulesForHost(ctx context.Context, hostID int64) ([]HostRule, error) {
	rows, err := s.q.ListSantaRulesForHost(ctx, sqlc.ListSantaRulesForHostParams{HostID: hostID})
	if err != nil {
		return nil, err
	}
	rules := make([]HostRule, len(rows))
	for i, row := range rows {
		rules[i] = hostRuleFromSQLC(row)
	}
	return rules, nil
}

func (s *Store) ListRuleStatusesForHost(
	ctx context.Context,
	hostID int64,
	params RuleStatusListParams,
) ([]RuleStatus, int, error) {
	if params.PageSize <= 0 {
		params.PageSize = 100
	}

	count, err := s.q.CountSantaRulesForHost(
		ctx,
		sqlc.CountSantaRulesForHostParams{HostID: hostID},
	)
	if err != nil {
		return nil, 0, err
	}
	offset := params.PageIndex * params.PageSize
	rows, err := s.q.ListSantaRulesForHostPage(
		ctx,
		sqlc.ListSantaRulesForHostPageParams{
			HostID:      hostID,
			LimitCount:  int32(params.PageSize),
			OffsetCount: int32(offset),
		},
	)
	if err != nil {
		return nil, 0, err
	}
	rules := make([]HostRule, len(rows))
	for i, row := range rows {
		rules[i] = hostRuleFromPageSQLC(row)
	}

	targets := SyncTargetsFromRules(rules)
	applied, err := s.appliedSyncTargetSet(ctx, hostID)
	if err != nil {
		return nil, 0, err
	}

	statuses := make([]RuleStatus, 0, len(rules))
	for i, rule := range rules {
		target := targets[i]
		appliedRule := applied[syncTargetKey(target)]
		statuses = append(statuses, RuleStatus{
			HostRule:    rule,
			Applied:     appliedRule,
			PayloadHash: target.PayloadHash,
		})
	}

	return statuses, int(count), nil
}

func (s *Store) appliedSyncTargetSet(ctx context.Context, hostID int64) (map[string]bool, error) {
	rows, err := s.q.ListAppliedSantaSyncTargets(ctx, sqlc.ListAppliedSantaSyncTargetsParams{HostID: hostID})
	if err != nil {
		return nil, err
	}
	applied := make([]syncstate.Target, len(rows))
	for i, row := range rows {
		applied[i] = syncTargetFromSQLC(row)
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
	q := sqlc.New(tx)
	if err := q.DeleteSantaRuleExcludeLabels(ctx, sqlc.DeleteSantaRuleExcludeLabelsParams{RuleID: ruleID}); err != nil {
		return err
	}
	if err := q.DeleteSantaRuleIncludes(ctx, sqlc.DeleteSantaRuleIncludesParams{RuleID: ruleID}); err != nil {
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
		if err := q.InsertSantaRuleIncludes(ctx, sqlc.InsertSantaRuleIncludesParams{
			RuleID:         ruleID,
			Policies:       policies,
			CelExpressions: celExpressions,
			LabelIds:       labelIDs,
		}); err != nil {
			return err
		}
	}
	if len(excludeLabelIDs) == 0 {
		return nil
	}
	return q.InsertSantaRuleExcludeLabels(ctx, sqlc.InsertSantaRuleExcludeLabelsParams{
		RuleID:   ruleID,
		LabelIds: excludeLabelIDs,
	})
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
	rows, err := s.q.ListSantaRuleIncludes(ctx, sqlc.ListSantaRuleIncludesParams{RuleIds: ruleIDs})
	if err != nil {
		return nil, err
	}

	out := map[int64][]RuleInclude{}
	for _, row := range rows {
		out[row.RuleID] = append(out[row.RuleID], RuleInclude{
			ID:            row.ID,
			Position:      row.Position,
			Policy:        Policy(row.Policy),
			CELExpression: row.CelExpression,
			LabelID:       row.LabelID,
		})
	}
	return out, nil
}

func (s *Store) loadRuleExcludeLabels(ctx context.Context, ruleIDs []int64) (map[int64][]int64, error) {
	rows, err := s.q.ListSantaRuleExcludeLabels(ctx, sqlc.ListSantaRuleExcludeLabelsParams{RuleIds: ruleIDs})
	if err != nil {
		return nil, err
	}

	out := map[int64][]int64{}
	for _, row := range rows {
		out[row.RuleID] = append(out[row.RuleID], row.LabelID)
	}
	return out, nil
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
	builtinExcludeIDs, err := sqlc.New(tx).ListBuiltinLabelIDs(
		ctx,
		sqlc.ListBuiltinLabelIDsParams{LabelIds: params.ExcludeLabelIDs},
	)
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
	complete, err := sqlc.New(tx).IsSantaBundleComplete(
		ctx,
		sqlc.IsSantaBundleCompleteParams{Sha256: params.Identifier},
	)
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
		where.Add(`(
			identifier ILIKE ` + search + `
			OR name ILIKE ` + search + `
			OR description ILIKE ` + search + `
		)`)
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
		"rule_type":   {SQL: "rule_type_sort"},
		"identifier":  {SQL: "identifier"},
		"name":        {SQL: "lower(name)"},
		"description": {SQL: "lower(description)"},
		"updated_at":  {SQL: "updated_at"},
	}
}

func scanRule(row pgx.Row) (Rule, error) {
	var rule Rule
	err := row.Scan(
		&rule.ID,
		&rule.RuleType,
		&rule.Identifier,
		&rule.Name,
		&rule.Description,
		&rule.CustomMessage,
		&rule.CustomURL,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	return rule, err
}

func ruleTargetFromSQLC(row sqlc.ListSantaRuleTargetsRow) RuleTarget {
	return RuleTarget{
		TargetType:                    RuleType(row.TargetType),
		Identifier:                    row.Identifier,
		DisplayName:                   row.DisplayName,
		CertificateCommonName:         row.CertificateCommonName,
		CertificateOrganization:       row.CertificateOrganization,
		CertificateOrganizationalUnit: row.CertificateOrganizationalUnit,
		FileName:                      row.FileName,
		BundleIdentifier:              row.BundleIdentifier,
		Path:                          row.Path,
		Version:                       row.Version,
		BinaryCount:                   row.BinaryCount,
		CollectedBinaryCount:          row.CollectedBinaryCount,
		RuleCount:                     row.RuleCount,
		Complete:                      row.Complete,
	}
}

func ruleFromSQLC(row sqlc.SantaRule) Rule {
	return Rule{
		ID:            row.ID,
		RuleType:      RuleType(row.RuleType),
		Identifier:    row.Identifier,
		Name:          row.Name,
		Description:   row.Description,
		CustomMessage: row.CustomMessage,
		CustomURL:     row.CustomURL,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func hostRuleFromSQLC(row sqlc.ListSantaRulesForHostRow) HostRule {
	return HostRule{
		RuleID:           row.RuleID,
		RuleType:         RuleType(row.RuleType),
		Identifier:       row.Identifier,
		Name:             row.Name,
		Description:      row.Description,
		Policy:           Policy(row.Policy),
		CELExpression:    row.CelExpression,
		CustomMessage:    row.CustomMessage,
		CustomURL:        row.CustomURL,
		AppName:          row.NotificationAppName,
		MatchedIncludeID: row.MatchedIncludeID,
	}
}

func hostRuleFromPageSQLC(row sqlc.ListSantaRulesForHostPageRow) HostRule {
	return HostRule{
		RuleID:           row.RuleID,
		RuleType:         RuleType(row.RuleType),
		Identifier:       row.Identifier,
		Name:             row.Name,
		Description:      row.Description,
		Policy:           Policy(row.Policy),
		CELExpression:    row.CelExpression,
		CustomMessage:    row.CustomMessage,
		CustomURL:        row.CustomURL,
		AppName:          row.NotificationAppName,
		MatchedIncludeID: row.MatchedIncludeID,
	}
}

// SyncTargetsFromRules returns Santa sync payload targets for host rules.
func SyncTargetsFromRules(rules []HostRule) []syncstate.Target {
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

func syncTargetFromSQLC(row sqlc.ListAppliedSantaSyncTargetsRow) syncstate.Target {
	return syncstate.Target{
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
	description,
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
