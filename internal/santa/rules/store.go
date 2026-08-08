package rules

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/postgres"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

// Store persists Santa rule definitions and resolves host rule state.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) List(ctx context.Context, params RuleListParams) ([]Rule, int, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, 0, err
	}
	where, args := ruleListWhere(params)
	listQuery := postgres.ListQuery{
		SelectSQL: ruleSelectSQL(),
		WhereSQL:  where,
		Args:      args,
		OrderKeys: ruleOrderKeys(),
		DefaultOrder: []postgres.OrderExpr{
			{SQL: "configuration_id"},
			{SQL: "rule_type_sort"},
			{SQL: "identifier"},
			{SQL: "id"},
		},
		Params: params.ListParams,
	}
	rows, count, err := postgres.ListWithCount[ruleRow](ctx, s.pool, listQuery)
	if err != nil {
		return nil, 0, err
	}
	rules := make([]Rule, len(rows))
	ruleIDs := make([]int64, len(rows))
	for i, row := range rows {
		rules[i] = ruleFromRow(row)
		ruleIDs[i] = row.ID
	}
	if err := s.attachRuleTargets(ctx, rules, ruleIDs); err != nil {
		return nil, 0, err
	}
	return rules, count, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Rule, error) {
	if id <= 0 {
		return nil, fault.ErrNotFound
	}
	row, err := postgres.GetOne[ruleRow](ctx, s.pool, ruleSelectSQL()+"\nWHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	rule := ruleFromRow(row)
	rules := []Rule{rule}
	if err := s.attachRuleTargets(ctx, rules, []int64{rule.ID}); err != nil {
		return nil, err
	}
	return &rules[0], nil
}

func (s *Store) Create(ctx context.Context, params RuleMutation) (*Rule, error) {
	params.normalize()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	write := newRuleWrite(params)
	var ruleID int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := validateRuleTargetingLabels(ctx, tx, params); err != nil {
			return err
		}
		if err := validateBundleRuleTarget(ctx, tx, params); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO santa_rules (
				configuration_id,
				rule_type,
				identifier,
				name,
				description,
				policy,
				cel_expression,
				custom_message,
				custom_url
			) VALUES (
				@configuration_id,
				@rule_type::santa_rule_type,
				@identifier,
				@name,
				@description,
				@policy::santa_policy,
				@cel_expression,
				@custom_message,
				@custom_url
			)
			RETURNING id`, pgx.StructArgs(write)).Scan(&ruleID); err != nil {
			return postgres.MutationError(err)
		}
		return replaceRuleTargets(ctx, tx, ruleID, params.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, ruleID)
}

func (s *Store) Update(ctx context.Context, id int64, params RuleMutation) (*Rule, error) {
	params.normalize()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	write := newRuleWrite(params)
	write.ID = id
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := validateRuleTargetingLabels(ctx, tx, params); err != nil {
			return err
		}
		if err := validateBundleRuleTarget(ctx, tx, params); err != nil {
			return err
		}
		var updatedID int64
		if err := tx.QueryRow(ctx, `
			UPDATE santa_rules
			SET
				configuration_id = @configuration_id,
				rule_type = @rule_type::santa_rule_type,
				identifier = @identifier,
				name = @name,
				description = @description,
				policy = @policy::santa_policy,
				cel_expression = @cel_expression,
				custom_message = @custom_message,
				custom_url = @custom_url,
				updated_at = now()
			WHERE id = @id
			RETURNING id`, pgx.StructArgs(write)).Scan(&updatedID); err != nil {
			return postgres.MutationError(err)
		}
		return replaceRuleTargets(ctx, tx, id, params.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM santa_rules WHERE id = $1`, id)
	if err != nil {
		return postgres.DeleteConflict(err, "Santa rule is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return fault.ErrNotFound
	}
	return nil
}

// DeleteMany removes multiple Santa rules. Missing IDs are ignored so repeated bulk actions are idempotent.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	rows, err := s.pool.Query(
		ctx,
		`DELETE FROM santa_rules WHERE id = ANY($1::bigint[]) RETURNING id`,
		ids,
	)
	if err != nil {
		return 0, err
	}
	deletedIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return 0, err
	}
	return len(deletedIDs), nil
}

func (s *Store) ResolveRulesForHost(ctx context.Context, hostID, configurationID int64) ([]HostRule, error) {
	qrows, err := s.pool.Query(ctx, `
		SELECT
			rule_id,
			rule_type,
			identifier,
			name,
			description,
			policy,
			cel_expression,
			custom_message,
			custom_url,
			notification_app_name
		FROM santa_resolved_rules_for_host($1, $2)
		ORDER BY rule_type_sort, identifier, rule_id`, hostID, configurationID)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[hostRuleRow])
	if err != nil {
		return nil, err
	}
	return hostRulesFromRows(rows), nil
}

func (s *Store) ListRuleStatusesForHost(
	ctx context.Context,
	hostID int64,
	configurationID int64,
	params RuleStatusListParams,
) ([]RuleStatus, int, error) {
	params.ListParams = listing.Normalize(params.ListParams)
	if err := listing.Validate(params.ListParams); err != nil {
		return nil, 0, err
	}

	var hostExists bool
	if err := s.pool.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM hosts WHERE id = $1)`,
		hostID,
	).Scan(&hostExists); err != nil {
		return nil, 0, err
	}
	if !hostExists {
		return nil, 0, fault.ErrNotFound
	}
	if configurationID <= 0 {
		return []RuleStatus{}, 0, nil
	}
	hostRules, err := s.ResolveRulesForHost(ctx, hostID, configurationID)
	if err != nil {
		return nil, 0, err
	}
	hostRules, targets, err := canonicalHostRules(hostRules)
	if err != nil {
		return nil, 0, err
	}
	count := len(hostRules)
	start := int(params.ListParams.PageIndex * params.ListParams.PageSize)
	if start >= count {
		return []RuleStatus{}, count, nil
	}
	end := min(start+int(params.ListParams.PageSize), count)
	hostRules = hostRules[start:end]
	targets = targets[start:end]

	applied, err := s.appliedSyncTargetSet(ctx, hostID)
	if err != nil {
		return nil, 0, err
	}
	statuses := make([]RuleStatus, 0, len(hostRules))
	for i, rule := range hostRules {
		target := targets[i]
		appliedRule := applied[target]
		statuses = append(statuses, RuleStatus{
			HostRule: rule,
			Applied:  appliedRule,
		})
	}
	return statuses, count, nil
}

type hostRuleRow struct {
	RuleID              int64  `db:"rule_id"`
	RuleType            string `db:"rule_type"`
	Identifier          string `db:"identifier"`
	Name                string `db:"name"`
	Description         string `db:"description"`
	Policy              string `db:"policy"`
	CelExpression       string `db:"cel_expression"`
	CustomMessage       string `db:"custom_message"`
	CustomURL           string `db:"custom_url"`
	NotificationAppName string `db:"notification_app_name"`
}

func hostRulesFromRows(rows []hostRuleRow) []HostRule {
	rules := make([]HostRule, len(rows))
	for i, row := range rows {
		rules[i] = HostRule{
			RuleID:        row.RuleID,
			RuleType:      RuleType(row.RuleType),
			Identifier:    row.Identifier,
			Name:          row.Name,
			Description:   row.Description,
			Policy:        Policy(row.Policy),
			CELExpression: row.CelExpression,
			CustomMessage: row.CustomMessage,
			CustomURL:     row.CustomURL,
			AppName:       row.NotificationAppName,
		}
	}
	return rules
}

func (s *Store) appliedSyncTargetSet(ctx context.Context, hostID int64) (map[syncstate.Target]bool, error) {
	type appliedRow struct {
		RuleType            string `db:"rule_type"`
		Identifier          string `db:"identifier"`
		Policy              string `db:"policy"`
		CelExpression       string `db:"cel_expression"`
		CustomMessage       string `db:"custom_message"`
		CustomURL           string `db:"custom_url"`
		NotificationAppName string `db:"notification_app_name"`
	}
	qrows, err := s.pool.Query(ctx, `
		SELECT
			rule_type::text,
			identifier,
			policy::text,
			cel_expression,
			custom_message,
			custom_url,
			notification_app_name
		FROM santa_sync_targets
		WHERE host_id = $1 AND phase = 'applied'`, hostID)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[appliedRow])
	if err != nil {
		return nil, err
	}
	applied := make([]syncstate.Target, len(rows))
	for i, row := range rows {
		applied[i] = syncstate.Target{
			RuleType:      row.RuleType,
			Identifier:    row.Identifier,
			Policy:        row.Policy,
			CELExpression: row.CelExpression,
			CustomMessage: row.CustomMessage,
			CustomURL:     row.CustomURL,
			AppName:       row.NotificationAppName,
		}
	}
	return syncstate.TargetSet(applied), nil
}

func replaceRuleTargets(
	ctx context.Context,
	tx pgx.Tx,
	ruleID int64,
	targets RuleTargets,
) error {
	targets = normalizeRuleTargets(targets)
	rows := make([]ruleTargetWrite, 0, len(targets.Include)+len(targets.Exclude))
	for i, include := range targets.Include {
		rows = append(rows, ruleTargetWrite{
			RuleID:    ruleID,
			Direction: string(targeting.Include),
			Position:  int32(i),
			LabelID:   include.LabelID,
		})
	}
	for i, exclude := range targets.Exclude {
		rows = append(rows, ruleTargetWrite{
			RuleID:    ruleID,
			Direction: string(targeting.Exclude),
			Position:  int32(i),
			LabelID:   exclude.LabelID,
		})
	}
	if err := postgres.ReplaceChildren(
		ctx, tx,
		`DELETE FROM santa_rule_targets WHERE rule_id = $1`, []any{ruleID},
		`
			INSERT INTO santa_rule_targets (rule_id, direction, position, label_id)
			VALUES (
				@rule_id,
				@direction::target_direction,
				@position,
				@label_id
			)`, rows,
	); err != nil {
		return postgres.MutationError(err)
	}
	return nil
}

func (s *Store) attachRuleTargets(ctx context.Context, rules []Rule, ruleIDs []int64) error {
	if len(ruleIDs) == 0 {
		return nil
	}
	ruleIndexes := make(map[int64]int, len(rules))
	for i := range rules {
		ruleIndexes[rules[i].ID] = i
		rules[i].Targets = emptyRuleTargets()
	}

	type targetRow struct {
		RuleID    int64  `db:"rule_id"`
		Direction string `db:"direction"`
		LabelID   int64  `db:"label_id"`
	}
	qrows, err := s.pool.Query(ctx, `
		SELECT
			rule_id,
			direction::text AS direction,
			label_id
		FROM santa_rule_targets
		WHERE rule_id = ANY($1::bigint[])
		ORDER BY rule_id, direction, position`,
		ruleIDs,
	)
	if err != nil {
		return err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[targetRow])
	if err != nil {
		return err
	}
	for _, row := range rows {
		i, ok := ruleIndexes[row.RuleID]
		if !ok {
			continue
		}
		tgts := rules[i].Targets
		switch targeting.Direction(row.Direction) {
		case targeting.Include:
			tgts.Include = append(tgts.Include, targeting.LabelRef{LabelID: row.LabelID})
		case targeting.Exclude:
			tgts.Exclude = append(tgts.Exclude, targeting.LabelRef{LabelID: row.LabelID})
		}
		rules[i].Targets = tgts
	}
	return nil
}

func validateRuleTargetingLabels(ctx context.Context, tx pgx.Tx, params RuleMutation) error {
	if len(params.Targets.Exclude) == 0 {
		return nil
	}
	ids := targeting.LabelRefIDs(params.Targets.Exclude)
	rows, err := tx.Query(ctx, `
		SELECT id FROM labels
		WHERE id = ANY($1::bigint[]) AND label_type = 'builtin'`, ids)
	if err != nil {
		return err
	}
	builtinIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return err
	}
	if len(builtinIDs) > 0 {
		return fmt.Errorf("%w: builtin labels cannot be excluded from Santa rules", fault.ErrInvalidInput)
	}
	return nil
}

func validateBundleRuleTarget(ctx context.Context, tx pgx.Tx, params RuleMutation) error {
	if params.RuleType != RuleTypeBundle {
		return nil
	}
	var complete bool
	err := tx.QueryRow(ctx, `
		SELECT (uploaded_at IS NOT NULL)::boolean AS complete
		FROM santa_bundles
		WHERE sha256 = $1`, params.Identifier).Scan(&complete)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: bundle reference is not collected", fault.ErrInvalidInput)
	}
	if err != nil {
		return err
	}
	if !complete {
		return fmt.Errorf("%w: bundle reference is incomplete", fault.ErrInvalidInput)
	}
	return nil
}

func ruleListWhere(params RuleListParams) (string, []any) {
	var where postgres.WhereBuilder
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
		where.Add(`(
			identifier ILIKE ` + search + `
			OR name ILIKE ` + search + `
			OR description ILIKE ` + search + `
		)`)
	}
	if len(params.RuleTypes) > 0 {
		where.Add("rule_type::text = ANY(" + where.Arg(params.RuleTypes) + "::text[])")
	}
	if len(params.ConfigurationIDs) > 0 {
		where.Add("configuration_id = ANY(" + where.Arg(params.ConfigurationIDs) + "::bigint[])")
	}
	return where.Build()
}

func ruleOrderKeys() map[string]postgres.OrderExpr {
	return map[string]postgres.OrderExpr{
		"configuration_id": {SQL: "configuration_id"},
		"rule_type":        {SQL: "rule_type_sort"},
		"identifier":       {SQL: "identifier"},
		"name":             {SQL: "lower(name)"},
		"description":      {SQL: "lower(description)"},
		"policy":           {SQL: "policy::text"},
		"updated_at":       {SQL: "updated_at"},
	}
}

type ruleRow struct {
	ID              int64     `db:"id"`
	ConfigurationID int64     `db:"configuration_id"`
	RuleType        string    `db:"rule_type"`
	Identifier      string    `db:"identifier"`
	Name            string    `db:"name"`
	Description     string    `db:"description"`
	Policy          string    `db:"policy"`
	CELExpression   string    `db:"cel_expression"`
	CustomMessage   string    `db:"custom_message"`
	CustomURL       string    `db:"custom_url"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

func ruleFromRow(row ruleRow) Rule {
	return Rule{
		ID:              row.ID,
		ConfigurationID: row.ConfigurationID,
		RuleType:        RuleType(row.RuleType),
		Identifier:      row.Identifier,
		Name:            row.Name,
		Description:     row.Description,
		Policy:          Policy(row.Policy),
		CELExpression:   row.CELExpression,
		CustomMessage:   row.CustomMessage,
		CustomURL:       row.CustomURL,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

type ruleWrite struct {
	ID              int64  `db:"id"`
	ConfigurationID int64  `db:"configuration_id"`
	RuleType        string `db:"rule_type"`
	Identifier      string `db:"identifier"`
	Name            string `db:"name"`
	Description     string `db:"description"`
	Policy          string `db:"policy"`
	CELExpression   string `db:"cel_expression"`
	CustomMessage   string `db:"custom_message"`
	CustomURL       string `db:"custom_url"`
}

func newRuleWrite(params RuleMutation) ruleWrite {
	return ruleWrite{
		ConfigurationID: params.ConfigurationID,
		RuleType:        string(params.RuleType),
		Identifier:      params.Identifier,
		Name:            params.Name,
		Description:     params.Description,
		Policy:          string(params.Policy),
		CELExpression:   params.CELExpression,
		CustomMessage:   params.CustomMessage,
		CustomURL:       params.CustomURL,
	}
}

type ruleTargetWrite struct {
	RuleID    int64  `db:"rule_id"`
	Direction string `db:"direction"`
	Position  int32  `db:"position"`
	LabelID   int64  `db:"label_id"`
}

// SyncTargetsFromRules returns one wire target per identity and rejects conflicting decisions.
func SyncTargetsFromRules(rules []HostRule) ([]syncstate.Target, error) {
	_, targets, err := canonicalHostRules(rules)
	return targets, err
}

func canonicalHostRules(rules []HostRule) ([]HostRule, []syncstate.Target, error) {
	canonicalRules := make([]HostRule, 0, len(rules))
	targets := make([]syncstate.Target, 0, len(rules))
	seen := make(map[string]syncstate.Target, len(rules))
	for _, rule := range rules {
		target := syncTargetFromRule(rule)
		identity := target.RuleType + "\x00" + target.Identifier
		if existing, ok := seen[identity]; ok {
			if existing != target {
				return nil, nil, fmt.Errorf(
					"%w: conflicting Santa rules for %s %q",
					fault.ErrConflict,
					rule.RuleType,
					rule.Identifier,
				)
			}
			continue
		}
		seen[identity] = target
		canonicalRules = append(canonicalRules, rule)
		targets = append(targets, target)
	}
	return canonicalRules, targets, nil
}

func syncTargetFromRule(rule HostRule) syncstate.Target {
	target := syncstate.Target{
		RuleType:      string(rule.RuleType),
		Identifier:    rule.Identifier,
		Policy:        string(rule.Policy),
		CELExpression: rule.CELExpression,
		CustomMessage: rule.CustomMessage,
		CustomURL:     rule.CustomURL,
		AppName:       rule.AppName,
	}
	return target
}

func ruleSelectSQL() string {
	return `
		SELECT
			id,
			configuration_id,
			rule_type::text,
			identifier,
			name,
			description,
			policy::text,
			cel_expression,
			custom_message,
			custom_url,
			created_at,
			updated_at
		FROM (
			SELECT
				r.id,
				r.configuration_id,
				r.rule_type,
				r.identifier,
				r.name,
				r.description,
				r.policy,
				r.cel_expression,
				r.custom_message,
				r.custom_url,
				r.created_at,
				r.updated_at,
				santa_rule_type_sort(r.rule_type) AS rule_type_sort
			FROM santa_rules r
		) sorted_rules`
}
