package rules

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/payloadhash"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

// Store persists Santa rule definitions and resolves host rule state.
type Store struct {
	db *database.DB
}

var (
	sha256IdentifierRE    = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	cdhashIdentifierRE    = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
	signingIDIdentifierRE = regexp.MustCompile(`^(?:[A-Z0-9]{10}|platform):[a-zA-Z0-9.-]+$`)
	teamIDIdentifierRE    = regexp.MustCompile(`^[A-Z0-9]{10}$`)
)

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

func (s *Store) List(ctx context.Context, params RuleListParams) ([]Rule, int, error) {
	where, args := ruleListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL:    ruleSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    ruleOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{{SQL: "rule_type_sort"}, {SQL: "identifier"}, {SQL: "id"}},
		Params:       params.ListParams,
	}
	rows, count, err := dbutil.ListWithCount[ruleRow](ctx, s.db.Pool(), listQuery)
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
		return nil, dbutil.ErrNotFound
	}
	row, err := dbutil.GetOne[ruleRow](ctx, s.db.Pool(), ruleSelectSQL+"\nWHERE id = $1", id)
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
	if err := params.Validate(); err != nil {
		return nil, err
	}
	write := newRuleWrite(params)
	var ruleID int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := validateRuleTargetingLabels(ctx, tx, params); err != nil {
			return err
		}
		if err := validateRuleReference(ctx, tx, params); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, insertRuleSQL, pgx.StructArgs(write)).Scan(&ruleID); err != nil {
			return dbutil.MutationError(err)
		}
		return replaceRuleTargets(ctx, tx, ruleID, params.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, ruleID)
}

func (s *Store) Update(ctx context.Context, id int64, params RuleMutation) (*Rule, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	write := newRuleWrite(params)
	write.ID = id
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := validateRuleTargetingLabels(ctx, tx, params); err != nil {
			return err
		}
		if err := validateRuleReference(ctx, tx, params); err != nil {
			return err
		}
		var updatedID int64
		if err := tx.QueryRow(ctx, updateRuleSQL, pgx.StructArgs(write)).Scan(&updatedID); err != nil {
			return dbutil.MutationError(err)
		}
		return replaceRuleTargets(ctx, tx, id, params.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Pool().Exec(ctx, `DELETE FROM santa_rules WHERE id = $1`, id)
	if err != nil {
		return dbutil.DeleteConflict(err, "Santa rule is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

// DeleteMany removes multiple Santa rules. Missing IDs are ignored so repeated bulk actions are idempotent.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	rows, err := s.db.Pool().Query(
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

func (s *Store) ListRuleReferences(
	ctx context.Context,
	params RuleReferenceListParams,
) ([]RuleReferenceCandidate, error) {
	if params.RuleType != "" && !validRuleType(params.RuleType) {
		return nil, fmt.Errorf("%w: unknown rule_type", dbutil.ErrInvalidInput)
	}
	if params.Limit <= 0 || params.Limit > 50 {
		params.Limit = 20
	}

	type refRow struct {
		RuleType                      string `db:"rule_type"`
		Identifier                    string `db:"identifier"`
		DisplayName                   string `db:"display_name"`
		CertificateCommonName         string `db:"certificate_common_name"`
		CertificateOrganization       string `db:"certificate_organization"`
		CertificateOrganizationalUnit string `db:"certificate_organizational_unit"`
		FileName                      string `db:"file_name"`
		BundleIdentifier              string `db:"bundle_identifier"`
		Path                          string `db:"path"`
		Version                       string `db:"version"`
		BinaryCount                   int32  `db:"binary_count"`
		CollectedBinaryCount          int32  `db:"collected_binary_count"`
		RuleCount                     int32  `db:"rule_count"`
		Complete                      bool   `db:"complete"`
	}

	qrows, err := s.db.Pool().Query(ctx, listRuleReferencesSQL, params.Q, string(params.RuleType), params.Limit)
	if err != nil {
		return nil, err
	}
	refRows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[refRow])
	if err != nil {
		return nil, err
	}
	candidates := make([]RuleReferenceCandidate, len(refRows))
	for i, row := range refRows {
		candidates[i] = RuleReferenceCandidate{
			RuleType:                      RuleType(row.RuleType),
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
	return candidates, nil
}

func (s *Store) ResolveRulesForHost(ctx context.Context, hostID int64) ([]HostRule, error) {
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
	qrows, err := s.db.Pool().Query(ctx, listRulesForHostSQL, hostID)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[hostRuleRow])
	if err != nil {
		return nil, err
	}
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
	if params.PageIndex < 0 {
		params.PageIndex = 0
	}

	var count int
	if err := s.db.Pool().QueryRow(ctx, countRulesForHostSQL, hostID).Scan(&count); err != nil {
		return nil, 0, err
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
	qrows, err := s.db.Pool().Query(
		ctx, listRulesForHostPageSQL,
		hostID,
		params.PageSize,
		params.PageIndex*params.PageSize,
	)
	if err != nil {
		return nil, 0, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[hostRuleRow])
	if err != nil {
		return nil, 0, err
	}
	hostRules := make([]HostRule, len(rows))
	for i, row := range rows {
		hostRules[i] = HostRule{
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

	targets := SyncTargetsFromRules(hostRules)
	applied, err := s.appliedSyncTargetSet(ctx, hostID)
	if err != nil {
		return nil, 0, err
	}
	statuses := make([]RuleStatus, 0, len(hostRules))
	for i, rule := range hostRules {
		target := targets[i]
		appliedRule := applied[target.Key()]
		statuses = append(statuses, RuleStatus{
			HostRule: rule,
			Applied:  appliedRule,
		})
	}
	return statuses, count, nil
}

func (s *Store) appliedSyncTargetSet(ctx context.Context, hostID int64) (map[string]bool, error) {
	type appliedRow struct {
		RuleType            string `db:"rule_type"`
		Identifier          string `db:"identifier"`
		Policy              string `db:"policy"`
		CelExpression       string `db:"cel_expression"`
		CustomMessage       string `db:"custom_message"`
		CustomURL           string `db:"custom_url"`
		NotificationAppName string `db:"notification_app_name"`
		PayloadHash         string `db:"payload_hash"`
	}
	qrows, err := s.db.Pool().Query(ctx, listAppliedSyncTargetsSQL, hostID)
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
			PayloadHash:   row.PayloadHash,
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
	if err := targets.validate(); err != nil {
		return err
	}
	rows := make([]ruleTargetWrite, 0, len(targets.Include)+len(targets.Exclude))
	for i, include := range targets.Include {
		rows = append(rows, ruleTargetWrite{
			RuleID:        ruleID,
			Direction:     string(targeting.Include),
			Position:      int32(i),
			LabelID:       include.LabelID,
			Policy:        string(include.Policy),
			CELExpression: include.CELExpression,
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
	if err := dbutil.ReplaceChildren(
		ctx, tx,
		deleteRuleTargetsSQL, []any{ruleID},
		insertRuleTargetSQL, rows,
	); err != nil {
		return dbutil.MutationError(err)
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
		RuleID        int64  `db:"rule_id"`
		Direction     string `db:"direction"`
		LabelID       int64  `db:"label_id"`
		Policy        string `db:"policy"`
		CELExpression string `db:"cel_expression"`
	}
	qrows, err := s.db.Pool().Query(ctx, `
		SELECT
			rule_id,
			direction::text AS direction,
			label_id,
			COALESCE(policy::text, '') AS policy,
			COALESCE(cel_expression, '') AS cel_expression
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
			tgts.Include = append(tgts.Include, RuleInclude{
				Policy:        Policy(row.Policy),
				CELExpression: row.CELExpression,
				LabelID:       row.LabelID,
			})
		case targeting.Exclude:
			tgts.Exclude = append(tgts.Exclude, targeting.LabelRef{LabelID: row.LabelID})
		}
		rules[i].Targets = tgts
	}
	return nil
}

func (p RuleMutation) Validate() error {
	if !validRuleType(p.RuleType) {
		return fmt.Errorf("%w: rule_type is required", dbutil.ErrInvalidInput)
	}
	if p.Identifier == "" {
		return fmt.Errorf("%w: identifier is required", dbutil.ErrInvalidInput)
	}
	if p.Name == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	if err := validateRuleIdentifier(p.RuleType, p.Identifier); err != nil {
		return err
	}
	if err := p.Targets.validate(); err != nil {
		return err
	}
	return nil
}

func validRuleType(ruleType RuleType) bool {
	return slices.Contains(RuleTypeValues, ruleType)
}

func validPolicy(policy Policy) bool {
	return slices.Contains(PolicyValues, policy)
}

func validateRuleIdentifier(ruleType RuleType, identifier string) error {
	switch ruleType {
	case RuleTypeBinary:
		if !sha256IdentifierRE.MatchString(identifier) {
			return fmt.Errorf("%w: identifier must be a 64 character SHA-256 hex hash", dbutil.ErrInvalidInput)
		}
	case RuleTypeCertificate:
		if !sha256IdentifierRE.MatchString(identifier) {
			return fmt.Errorf(
				"%w: identifier must be a 64 character certificate SHA-256 hex fingerprint",
				dbutil.ErrInvalidInput,
			)
		}
	case RuleTypeBundle:
		if !sha256IdentifierRE.MatchString(identifier) {
			return fmt.Errorf("%w: identifier must be a 64 character bundle SHA-256 hex hash", dbutil.ErrInvalidInput)
		}
	case RuleTypeCDHash:
		if !cdhashIdentifierRE.MatchString(identifier) {
			return fmt.Errorf("%w: identifier must be a 40 character CDHash hex value", dbutil.ErrInvalidInput)
		}
	case RuleTypeSigningID:
		if !signingIDIdentifierRE.MatchString(identifier) {
			return fmt.Errorf(
				"%w: identifier must be TEAMID:bundle.identifier or platform:bundle.identifier",
				dbutil.ErrInvalidInput,
			)
		}
	case RuleTypeTeamID:
		if !teamIDIdentifierRE.MatchString(identifier) {
			return fmt.Errorf("%w: identifier must be a 10 character uppercase Team ID", dbutil.ErrInvalidInput)
		}
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
		return fmt.Errorf("%w: builtin labels cannot be excluded from Santa rules", dbutil.ErrInvalidInput)
	}
	return nil
}

func validateRuleReference(ctx context.Context, tx pgx.Tx, params RuleMutation) error {
	if params.RuleType != RuleTypeBundle {
		return nil
	}
	var complete bool
	err := tx.QueryRow(ctx, `
		SELECT (uploaded_at IS NOT NULL)::boolean AS complete
		FROM santa_bundles
		WHERE sha256 = $1`, params.Identifier).Scan(&complete)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: bundle reference is not collected", dbutil.ErrInvalidInput)
	}
	if err != nil {
		return err
	}
	if !complete {
		return fmt.Errorf("%w: bundle reference is incomplete", dbutil.ErrInvalidInput)
	}
	return nil
}

func ruleListWhere(params RuleListParams) (string, []any) {
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
	return where.Build()
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

type ruleRow struct {
	ID            int64     `db:"id"`
	RuleType      string    `db:"rule_type"`
	Identifier    string    `db:"identifier"`
	Name          string    `db:"name"`
	Description   string    `db:"description"`
	CustomMessage string    `db:"custom_message"`
	CustomURL     string    `db:"custom_url"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

func ruleFromRow(row ruleRow) Rule {
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

type ruleWrite struct {
	ID            int64  `db:"id"`
	RuleType      string `db:"rule_type"`
	Identifier    string `db:"identifier"`
	Name          string `db:"name"`
	Description   string `db:"description"`
	CustomMessage string `db:"custom_message"`
	CustomURL     string `db:"custom_url"`
}

func newRuleWrite(params RuleMutation) ruleWrite {
	return ruleWrite{
		RuleType:      string(params.RuleType),
		Identifier:    params.Identifier,
		Name:          params.Name,
		Description:   params.Description,
		CustomMessage: params.CustomMessage,
		CustomURL:     params.CustomURL,
	}
}

type ruleTargetWrite struct {
	RuleID        int64  `db:"rule_id"`
	Direction     string `db:"direction"`
	Position      int32  `db:"position"`
	LabelID       int64  `db:"label_id"`
	Policy        string `db:"policy"`
	CELExpression string `db:"cel_expression"`
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
	return payloadhash.Hash(
		target.RuleType,
		target.Identifier,
		target.Policy,
		target.CELExpression,
		target.CustomMessage,
		target.CustomURL,
		target.AppName,
	)
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

//nolint:unqueryvet // inner r.* is re-projected by the explicit outer SELECT; new columns are ignored
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

const insertRuleSQL = `
INSERT INTO santa_rules (
	rule_type,
	identifier,
	name,
	description,
	custom_message,
	custom_url
) VALUES (
	@rule_type::santa_rule_type,
	@identifier,
	@name,
	@description,
	@custom_message,
	@custom_url
)
RETURNING id`

const updateRuleSQL = `
UPDATE santa_rules
SET
	rule_type = @rule_type::santa_rule_type,
	identifier = @identifier,
	name = @name,
	description = @description,
	custom_message = @custom_message,
	custom_url = @custom_url,
	updated_at = now()
WHERE id = @id
RETURNING id`

const deleteRuleTargetsSQL = `DELETE FROM santa_rule_targets WHERE rule_id = $1`

const insertRuleTargetSQL = `
INSERT INTO santa_rule_targets (rule_id, direction, position, label_id, policy, cel_expression)
VALUES (
	@rule_id,
	@direction::target_direction,
	@position,
	@label_id,
	NULLIF(@policy, '')::santa_policy,
	NULLIF(@cel_expression, '')
)`

//nolint:unqueryvet // expanded_rules columns are declared explicitly; CASE expression in CTE is not SELECT *
const listRulesForHostSQL = `
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
		r.name,
		r.description,
		i.policy,
		COALESCE(i.cel_expression, '') AS cel_expression,
		r.custom_message,
		r.custom_url,
		i.position::bigint AS matched_include_id,
		CASE r.rule_type
			WHEN 'cdhash' THEN 1
			WHEN 'binary' THEN 2
			WHEN 'signingid' THEN 3
			WHEN 'certificate' THEN 4
			WHEN 'teamid' THEN 5
			WHEN 'bundle' THEN 6
			ELSE 7
		END AS rule_type_sort,
		row_number() OVER (PARTITION BY r.id ORDER BY i.position) AS include_rank
	FROM santa_rules r
	JOIN santa_rule_targets i ON i.rule_id = r.id AND i.direction = 'include'
	JOIN host_labels include_hl ON include_hl.label_id = i.label_id
	WHERE NOT EXISTS (
		SELECT 1
		FROM santa_rule_targets el
		JOIN host_labels hl ON hl.label_id = el.label_id
		WHERE el.rule_id = r.id
		  AND el.direction = 'exclude'
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
		name,
		description,
		policy,
		cel_expression,
		custom_message,
		custom_url,
		''::text AS notification_app_name,
		rule_type_sort
	FROM selected_includes
	WHERE rule_type <> 'bundle'
	UNION ALL
	SELECT
		si.rule_id,
		'binary'::santa_rule_type AS rule_type,
		e.sha256 AS identifier,
		si.name,
		si.description,
		si.policy,
		si.cel_expression,
		si.custom_message,
		si.custom_url,
		COALESCE(NULLIF(b.name, ''), NULLIF(b.bundle_id, ''), b.sha256) AS notification_app_name,
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
	name,
	description,
	policy::text,
	cel_expression,
	custom_message,
	custom_url,
	notification_app_name
FROM expanded_rules
ORDER BY rule_type_sort, identifier, rule_id`

const countRulesForHostSQL = `
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
		i.position,
		row_number() OVER (PARTITION BY r.id ORDER BY i.position) AS include_rank
	FROM santa_rules r
	JOIN santa_rule_targets i ON i.rule_id = r.id AND i.direction = 'include'
	JOIN host_labels include_hl ON include_hl.label_id = i.label_id
	WHERE NOT EXISTS (
		SELECT 1
		FROM santa_rule_targets el
		JOIN host_labels hl ON hl.label_id = el.label_id
		WHERE el.rule_id = r.id
		  AND el.direction = 'exclude'
	)
),
selected_includes AS (
	SELECT *
	FROM matching_includes
	WHERE include_rank = 1
),
expanded_rules AS (
	SELECT rule_id FROM selected_includes WHERE rule_type <> 'bundle'
	UNION ALL
	SELECT si.rule_id
	FROM selected_includes si
	JOIN santa_bundles b ON b.sha256 = si.identifier AND b.uploaded_at IS NOT NULL
	JOIN santa_bundle_executables be ON be.bundle_id = b.id
	JOIN santa_executables e ON e.id = be.executable_id
	WHERE si.rule_type = 'bundle'
	  AND e.sha256 <> ''
)
SELECT count(*)::integer FROM expanded_rules`

//nolint:unqueryvet // expanded_rules columns are declared explicitly; CASE expression in CTE is not SELECT *
const listRulesForHostPageSQL = `
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
		r.name,
		r.description,
		i.policy,
		COALESCE(i.cel_expression, '') AS cel_expression,
		r.custom_message,
		r.custom_url,
		i.position::bigint AS matched_include_id,
		CASE r.rule_type
			WHEN 'cdhash' THEN 1
			WHEN 'binary' THEN 2
			WHEN 'signingid' THEN 3
			WHEN 'certificate' THEN 4
			WHEN 'teamid' THEN 5
			WHEN 'bundle' THEN 6
			ELSE 7
		END AS rule_type_sort,
		row_number() OVER (PARTITION BY r.id ORDER BY i.position) AS include_rank
	FROM santa_rules r
	JOIN santa_rule_targets i ON i.rule_id = r.id AND i.direction = 'include'
	JOIN host_labels include_hl ON include_hl.label_id = i.label_id
	WHERE NOT EXISTS (
		SELECT 1
		FROM santa_rule_targets el
		JOIN host_labels hl ON hl.label_id = el.label_id
		WHERE el.rule_id = r.id
		  AND el.direction = 'exclude'
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
		name,
		description,
		policy,
		cel_expression,
		custom_message,
		custom_url,
		''::text AS notification_app_name,
		rule_type_sort
	FROM selected_includes
	WHERE rule_type <> 'bundle'
	UNION ALL
	SELECT
		si.rule_id,
		'binary'::santa_rule_type AS rule_type,
		e.sha256 AS identifier,
		si.name,
		si.description,
		si.policy,
		si.cel_expression,
		si.custom_message,
		si.custom_url,
		COALESCE(NULLIF(b.name, ''), NULLIF(b.bundle_id, ''), b.sha256) AS notification_app_name,
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
	name,
	description,
	policy::text,
	cel_expression,
	custom_message,
	custom_url,
	notification_app_name
FROM expanded_rules
ORDER BY rule_type_sort, identifier, rule_id
LIMIT $2 OFFSET $3`

const listAppliedSyncTargetsSQL = `
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
WHERE host_id = $1 AND phase = 'applied'`

const listRuleReferencesSQL = `
WITH candidate_sources AS (
	SELECT
		'binary'::text AS rule_type,
		e.sha256 AS identifier,
		NULLIF(e.file_bundle_name, '') AS display_name,
		NULL::text AS certificate_common_name,
		NULL::text AS certificate_organization,
		NULL::text AS certificate_organizational_unit,
		NULLIF(e.file_name, '') AS file_name,
		NULLIF(e.file_bundle_id, '') AS bundle_identifier,
		NULLIF(e.file_bundle_path, '') AS path,
		COALESCE(NULLIF(e.file_bundle_version_string, ''), NULLIF(e.file_bundle_version, '')) AS version,
		0::integer AS binary_count,
		0::integer AS collected_binary_count,
		true AS complete
	FROM santa_executables e
	WHERE e.sha256 <> ''
	UNION ALL
	SELECT
		'binary'::text,
		p.executable_sha256,
		COALESCE(NULLIF(st.display_name, ''), NULLIF(st.name, '')),
		NULL::text,
		NULL::text,
		NULL::text,
		NULL::text,
		NULLIF(s.bundle_identifier, ''),
		COALESCE(NULLIF(p.executable_path, ''), NULLIF(p.installed_path, '')),
		NULLIF(s.version, ''),
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
		NULL::text,
		NULLIF(c.common_name, ''),
		NULLIF(c.organization, ''),
		NULLIF(c.organizational_unit, ''),
		NULL::text,
		NULL::text,
		NULL::text,
		NULL::text,
		0::integer,
		0::integer,
		true
	FROM santa_certificates c
	WHERE c.sha256 <> ''
	UNION ALL
	SELECT
		'teamid'::text,
		e.team_id,
		NULL::text,
		NULL::text,
		NULLIF(c.organization, ''),
		NULLIF(c.organizational_unit, ''),
		NULLIF(e.file_name, ''),
		NULLIF(e.file_bundle_id, ''),
		NULLIF(e.file_bundle_path, ''),
		COALESCE(NULLIF(e.file_bundle_version_string, ''), NULLIF(e.file_bundle_version, '')),
		0::integer,
		0::integer,
		true
	FROM santa_executables e
	LEFT JOIN santa_executable_signing_chains esc ON esc.executable_id = e.id
	LEFT JOIN santa_signing_chain_entries sce ON sce.signing_chain_id = esc.signing_chain_id
	LEFT JOIN santa_certificates c ON c.id = sce.certificate_id AND c.organizational_unit = e.team_id
	WHERE e.team_id <> ''
	UNION ALL
	SELECT
		'teamid'::text,
		p.team_identifier,
		NULL::text,
		NULL::text,
		NULL::text,
		NULL::text,
		NULL::text,
		NULLIF(s.bundle_identifier, ''),
		COALESCE(NULLIF(p.executable_path, ''), NULLIF(p.installed_path, '')),
		NULLIF(s.version, ''),
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
		NULLIF(e.file_bundle_name, ''),
		NULL::text,
		NULL::text,
		NULL::text,
		NULLIF(e.file_name, ''),
		NULLIF(e.file_bundle_id, ''),
		NULLIF(e.file_bundle_path, ''),
		COALESCE(NULLIF(e.file_bundle_version_string, ''), NULLIF(e.file_bundle_version, '')),
		0::integer,
		0::integer,
		true
	FROM santa_executables e
	WHERE e.signing_id <> ''
	UNION ALL
	SELECT
		'signingid'::text,
		p.team_identifier || ':' || s.bundle_identifier,
		COALESCE(NULLIF(st.display_name, ''), NULLIF(st.name, '')),
		NULL::text,
		NULL::text,
		NULL::text,
		NULL::text,
		NULLIF(s.bundle_identifier, ''),
		COALESCE(NULLIF(p.executable_path, ''), NULLIF(p.installed_path, '')),
		NULLIF(s.version, ''),
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
		NULLIF(e.file_bundle_name, ''),
		NULL::text,
		NULL::text,
		NULL::text,
		NULLIF(e.file_name, ''),
		NULLIF(e.file_bundle_id, ''),
		NULLIF(e.file_bundle_path, ''),
		COALESCE(NULLIF(e.file_bundle_version_string, ''), NULLIF(e.file_bundle_version, '')),
		0::integer,
		0::integer,
		true
	FROM santa_executables e
	WHERE e.cdhash <> ''
	UNION ALL
	SELECT
		'cdhash'::text,
		p.cdhash_sha256,
		COALESCE(NULLIF(st.display_name, ''), NULLIF(st.name, '')),
		NULL::text,
		NULL::text,
		NULL::text,
		NULL::text,
		NULLIF(s.bundle_identifier, ''),
		COALESCE(NULLIF(p.executable_path, ''), NULLIF(p.installed_path, '')),
		NULLIF(s.version, ''),
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
		NULLIF(b.name, ''),
		NULL::text,
		NULL::text,
		NULL::text,
		NULL::text,
		NULLIF(b.bundle_id, ''),
		NULLIF(b.path, ''),
		COALESCE(NULLIF(b.version_string, ''), NULLIF(b.version, '')),
		b.binary_count,
		COUNT(be.executable_id)::integer,
		b.uploaded_at IS NOT NULL
	FROM santa_bundles b
	LEFT JOIN santa_bundle_executables be ON be.bundle_id = b.id
	WHERE b.sha256 <> ''
	GROUP BY b.id
),
candidates AS (
	SELECT
		rule_type,
		identifier,
		COALESCE(
			CASE WHEN COUNT(DISTINCT NULLIF(display_name, '')) = 1 THEN MIN(NULLIF(display_name, '')) END,
			''
		)::text AS display_name,
		COALESCE(
			CASE
				WHEN COUNT(DISTINCT NULLIF(certificate_common_name, '')) = 1
				THEN MIN(NULLIF(certificate_common_name, ''))
			END,
			''
		)::text AS certificate_common_name,
		COALESCE(
			CASE
				WHEN COUNT(DISTINCT NULLIF(certificate_organization, '')) = 1
				THEN MIN(NULLIF(certificate_organization, ''))
			END,
			''
		)::text AS certificate_organization,
		COALESCE(
			CASE
				WHEN COUNT(DISTINCT NULLIF(certificate_organizational_unit, '')) = 1
				THEN MIN(NULLIF(certificate_organizational_unit, ''))
			END,
			''
		)::text AS certificate_organizational_unit,
		COALESCE(
			CASE WHEN COUNT(DISTINCT NULLIF(file_name, '')) = 1 THEN MIN(NULLIF(file_name, '')) END,
			''
		)::text AS file_name,
		COALESCE(
			CASE
				WHEN COUNT(DISTINCT NULLIF(bundle_identifier, '')) = 1
				THEN MIN(NULLIF(bundle_identifier, ''))
			END,
			''
		)::text AS bundle_identifier,
		COALESCE(
			CASE WHEN COUNT(DISTINCT NULLIF(path, '')) = 1 THEN MIN(NULLIF(path, '')) END,
			''
		)::text AS path,
		COALESCE(
			CASE WHEN COUNT(DISTINCT NULLIF(version, '')) = 1 THEN MIN(NULLIF(version, '')) END,
			''
		)::text AS version,
		max(binary_count)::integer AS binary_count,
		max(collected_binary_count)::integer AS collected_binary_count,
		bool_or(complete) AS complete,
		COALESCE(
			string_agg(
				DISTINCT concat_ws(
					' ',
					NULLIF(display_name, ''),
					NULLIF(certificate_common_name, ''),
					NULLIF(certificate_organization, ''),
					NULLIF(certificate_organizational_unit, ''),
					NULLIF(file_name, ''),
					NULLIF(bundle_identifier, ''),
					NULLIF(path, ''),
					NULLIF(version, '')
				),
				' '
			),
			''
		)::text AS search_text
	FROM candidate_sources
	WHERE identifier <> ''
	GROUP BY rule_type, identifier
)
SELECT
	t.rule_type,
	t.identifier,
	t.display_name,
	t.certificate_common_name,
	t.certificate_organization,
	t.certificate_organizational_unit,
	t.file_name,
	t.bundle_identifier,
	t.path,
	t.version,
	t.binary_count,
	t.collected_binary_count,
	COUNT(r.id)::integer AS rule_count,
	t.complete
FROM candidates t
LEFT JOIN santa_rules r
	ON r.rule_type::text = t.rule_type AND r.identifier = t.identifier
WHERE
	($1::text = ''
		OR t.identifier ILIKE '%' || $1::text || '%'
		OR t.display_name ILIKE '%' || $1::text || '%'
		OR t.certificate_common_name ILIKE '%' || $1::text || '%'
		OR t.certificate_organization ILIKE '%' || $1::text || '%'
		OR t.certificate_organizational_unit ILIKE '%' || $1::text || '%'
		OR t.file_name ILIKE '%' || $1::text || '%'
		OR t.bundle_identifier ILIKE '%' || $1::text || '%'
		OR t.path ILIKE '%' || $1::text || '%'
		OR t.version ILIKE '%' || $1::text || '%'
		OR t.search_text ILIKE '%' || $1::text || '%')
	AND ($2::text = '' OR t.rule_type = $2::text)
GROUP BY
	t.rule_type,
	t.identifier,
	t.display_name,
	t.certificate_common_name,
	t.certificate_organization,
	t.certificate_organizational_unit,
	t.file_name,
	t.bundle_identifier,
	t.path,
	t.version,
	t.binary_count,
	t.collected_binary_count,
	t.complete
ORDER BY
	CASE t.rule_type
		WHEN 'bundle' THEN 1
		WHEN 'signingid' THEN 2
		WHEN 'teamid' THEN 3
		WHEN 'certificate' THEN 4
		WHEN 'binary' THEN 5
		WHEN 'cdhash' THEN 6
		ELSE 7
	END,
	lower(COALESCE(NULLIF(t.display_name, ''), NULLIF(t.certificate_common_name, ''), t.identifier)),
	t.identifier
LIMIT $3`
