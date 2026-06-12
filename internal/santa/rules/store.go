package rules

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

// Store persists Santa rule definitions and resolves host rule state.
type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

var (
	sha256IdentifierRE    = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	cdhashIdentifierRE    = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
	signingIDIdentifierRE = regexp.MustCompile(`^(?:[A-Z0-9]{10}|platform):[a-zA-Z0-9.-]+$`)
	teamIDIdentifierRE    = regexp.MustCompile(`^[A-Z0-9]{10}$`)
)

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

func (s *Store) ListRules(ctx context.Context, params RuleListParams) ([]Rule, int, error) {
	where, args := ruleListWhere(params)
	listQuery := ruleListQuery(params, where, args)

	var count int
	countSQL, countArgs := listQuery.BuildCount()
	if err := s.db.Pool().QueryRow(ctx, countSQL, countArgs...).Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := listQuery.Build()
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
	if err := s.attachRuleTargets(ctx, rules, ruleIDs); err != nil {
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
	if err := s.attachRuleTargets(ctx, rules, []int64{rule.ID}); err != nil {
		return nil, err
	}
	return &rules[0], nil
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
	rows, err := s.q.ListSantaRuleReferences(ctx, sqlc.ListSantaRuleReferencesParams{
		Q:          params.Q,
		RuleType:   string(params.RuleType),
		LimitCount: int32(params.Limit),
	})
	if err != nil {
		return nil, err
	}
	candidates := make([]RuleReferenceCandidate, len(rows))
	for i, row := range rows {
		candidates[i] = ruleReferenceCandidateFromSQLC(row)
	}
	return candidates, nil
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
		if err := validateRuleTargetingLabels(ctx, tx, params); err != nil {
			return err
		}
		if err := validateRuleReference(ctx, tx, params); err != nil {
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
			return err
		}
		ruleID = row.ID
		return replaceRuleTargets(ctx, tx, ruleID, params.Targets)
	})
	if err != nil {
		return nil, mapRuleMutationError(err)
	}
	return s.GetRuleByID(ctx, ruleID)
}

func (s *Store) UpdateRule(ctx context.Context, id int64, params RuleMutation) (*Rule, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := validateRuleTargetingLabels(ctx, tx, params); err != nil {
			return err
		}
		if err := validateRuleReference(ctx, tx, params); err != nil {
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
			return err
		}
		return replaceRuleTargets(ctx, tx, row.ID, params.Targets)
	})
	if err != nil {
		return nil, mapRuleMutationError(err)
	}
	return s.GetRuleByID(ctx, id)
}

func mapRuleMutationError(err error) error {
	switch database.SQLState(err) {
	case pgerrcode.ForeignKeyViolation:
		return dbutil.ErrNotFound
	case pgerrcode.UniqueViolation:
		return dbutil.ErrAlreadyExists
	case pgerrcode.InvalidTextRepresentation,
		pgerrcode.NotNullViolation,
		pgerrcode.CheckViolation:
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return err
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

	q := sqlc.New(tx)
	if err := q.DeleteSantaRuleExcludeLabels(ctx, sqlc.DeleteSantaRuleExcludeLabelsParams{RuleID: ruleID}); err != nil {
		return err
	}
	if err := q.DeleteSantaRuleIncludes(ctx, sqlc.DeleteSantaRuleIncludesParams{RuleID: ruleID}); err != nil {
		return err
	}
	if len(targets.Include) > 0 {
		policies := make([]string, len(targets.Include))
		celExpressions := make([]string, len(targets.Include))
		labelIDs := make([]int64, len(targets.Include))
		for i, include := range targets.Include {
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
	if len(targets.Exclude) == 0 {
		return nil
	}
	return q.InsertSantaRuleExcludeLabels(ctx, sqlc.InsertSantaRuleExcludeLabelsParams{
		RuleID:   ruleID,
		LabelIds: labelRefIDs(targets.Exclude),
	})
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

	includes, err := s.loadRuleIncludes(ctx, ruleIDs)
	if err != nil {
		return err
	}
	for ruleID, values := range includes {
		if i, ok := ruleIndexes[ruleID]; ok {
			targets := rules[i].Targets
			targets.Include = values
			rules[i].Targets = targets
		}
	}

	excludes, err := s.loadRuleExcludes(ctx, ruleIDs)
	if err != nil {
		return err
	}
	for ruleID, values := range excludes {
		if i, ok := ruleIndexes[ruleID]; ok {
			targets := rules[i].Targets
			targets.Exclude = values
			rules[i].Targets = targets
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
			Policy:        Policy(row.Policy),
			CELExpression: row.CelExpression,
			LabelID:       row.LabelID,
		})
	}
	return out, nil
}

func (s *Store) loadRuleExcludes(ctx context.Context, ruleIDs []int64) (map[int64][]targeting.LabelRef, error) {
	rows, err := s.q.ListSantaRuleExcludeLabels(ctx, sqlc.ListSantaRuleExcludeLabelsParams{RuleIds: ruleIDs})
	if err != nil {
		return nil, err
	}

	out := map[int64][]targeting.LabelRef{}
	for _, row := range rows {
		out[row.RuleID] = append(out[row.RuleID], targeting.LabelRef{LabelID: row.LabelID})
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
	builtinExcludeIDs, err := sqlc.New(tx).ListBuiltinLabelIDs(
		ctx,
		sqlc.ListBuiltinLabelIDsParams{LabelIds: labelRefIDs(params.Targets.Exclude)},
	)
	if err != nil {
		return err
	}
	if len(builtinExcludeIDs) > 0 {
		return fmt.Errorf("%w: builtin labels cannot be excluded from Santa rules", dbutil.ErrInvalidInput)
	}
	return nil
}

func validateRuleReference(ctx context.Context, tx pgx.Tx, params RuleMutation) error {
	if params.RuleType != RuleTypeBundle {
		return nil
	}
	complete, err := sqlc.New(tx).IsSantaBundleComplete(
		ctx,
		sqlc.IsSantaBundleCompleteParams{Sha256: params.Identifier},
	)
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

func ruleListQuery(params RuleListParams, where string, args []any) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL:    ruleSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    ruleOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{{SQL: "rule_type_sort"}, {SQL: "identifier"}, {SQL: "id"}},
		Params:       params.ListParams,
	}
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

func ruleReferenceCandidateFromSQLC(row sqlc.ListSantaRuleReferencesRow) RuleReferenceCandidate {
	return RuleReferenceCandidate{
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
