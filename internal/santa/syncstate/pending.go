package syncstate

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
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
	clientRulesHash string,
	desired []Target,
	reported RuleCounts,
	requestCleanSync bool,
) (SyncType, error) {
	desired = sortedTargets(desired)

	var pendingFullSync bool
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		applied, err := loadPriorState(ctx, q, hostID)
		if err != nil {
			return err
		}
		pendingFullSync = applied.requiresFullSync(desired, reported, requestCleanSync)

		payload := incrementalPayload(desired, applied.targets)
		if pendingFullSync {
			payload = fullSyncPayload(desired)
		}

		if err := upsertPreflight(ctx, q, preflightParams{
			hostID:          hostID,
			clientRulesHash: clientRulesHash,
			pendingFullSync: pendingFullSync,
			payload:         payload,
			desired:         desired,
			reported:        reported,
		}); err != nil {
			return err
		}
		return rewritePendingState(ctx, q, hostID, desired, payload)
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

func loadPriorState(ctx context.Context, q *sqlc.Queries, hostID int64) (priorState, error) {
	exists, err := syncStateExists(ctx, q, hostID)
	if err != nil {
		return priorState{}, err
	}
	targets, err := loadTargets(ctx, q, hostID, phaseApplied)
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
	clientRulesHash string
	pendingFullSync bool
	payload         []PayloadRule
	desired         []Target
	reported        RuleCounts
}

func upsertPreflight(ctx context.Context, q *sqlc.Queries, p preflightParams) error {
	desiredCounts := countTargets(p.desired)
	return q.UpsertSantaSyncPreflight(ctx, sqlc.UpsertSantaSyncPreflightParams{
		HostID:                      p.hostID,
		ClientRulesHash:             p.clientRulesHash,
		PendingFullSync:             p.pendingFullSync,
		PendingPayloadRuleCount:     int32(len(p.payload)),
		DesiredBinaryRuleCount:      desiredCounts.Binary,
		DesiredCertificateRuleCount: desiredCounts.Certificate,
		DesiredTeamidRuleCount:      desiredCounts.TeamID,
		DesiredSigningidRuleCount:   desiredCounts.SigningID,
		DesiredCdhashRuleCount:      desiredCounts.CDHash,
		DesiredCompilerRuleCount:    desiredCounts.Compiler,
		BinaryRuleCount:             p.reported.Binary,
		CertificateRuleCount:        p.reported.Certificate,
		TeamidRuleCount:             p.reported.TeamID,
		SigningidRuleCount:          p.reported.SigningID,
		CdhashRuleCount:             p.reported.CDHash,
		CompilerRuleCount:           p.reported.Compiler,
		TransitiveRuleCount:         p.reported.Transitive,
		CountsMatch:                 desiredCounts.MatchesReported(p.reported),
	})
}

func rewritePendingState(
	ctx context.Context,
	q *sqlc.Queries,
	hostID int64,
	desired []Target,
	payload []PayloadRule,
) error {
	if err := q.DeleteSantaSyncTargetsByPhase(ctx, sqlc.DeleteSantaSyncTargetsByPhaseParams{
		HostID: hostID,
		Phase:  sqlc.SantaSyncTargetPhase(phaseDesired),
	}); err != nil {
		return err
	}
	if err := q.DeleteSantaSyncPendingRules(
		ctx,
		sqlc.DeleteSantaSyncPendingRulesParams{HostID: hostID},
	); err != nil {
		return err
	}
	if err := insertTargets(ctx, q, hostID, phaseDesired, desired); err != nil {
		return err
	}
	return insertPayloadRules(ctx, q, hostID, payload)
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

	rows, err := s.q.ListSantaPendingPayloadPage(ctx, sqlc.ListSantaPendingPayloadPageParams{
		HostID:      hostID,
		LimitCount:  limit + 1,
		OffsetCount: offset,
	})
	if err != nil {
		return PayloadRulePage{}, err
	}

	rules := make([]PayloadRule, len(rows))
	for i, row := range rows {
		rules[i] = payloadRuleFromSQLC(row)
	}
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
	clientRulesHash string,
	rulesReceived int32,
	rulesProcessed int32,
) error {
	var pendingCount int32
	var pendingFullSync bool
	row, err := s.q.GetSantaPendingState(ctx, sqlc.GetSantaPendingStateParams{HostID: hostID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	pendingCount = row.PendingPayloadRuleCount
	pendingFullSync = row.PendingFullSync
	if rulesProcessed != pendingCount {
		return s.q.MarkSantaSyncAttempt(ctx, sqlc.MarkSantaSyncAttemptParams{
			HostID:          hostID,
			ClientRulesHash: clientRulesHash,
			RulesReceived:   rulesReceived,
			RulesProcessed:  rulesProcessed,
		})
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.DeleteSantaSyncTargetsByPhase(ctx, sqlc.DeleteSantaSyncTargetsByPhaseParams{
			HostID: hostID,
			Phase:  sqlc.SantaSyncTargetPhase(phaseApplied),
		}); err != nil {
			return err
		}
		if err := q.PromoteSantaDesiredSyncTargets(
			ctx,
			sqlc.PromoteSantaDesiredSyncTargetsParams{PromoteHostID: hostID},
		); err != nil {
			return err
		}
		if err := q.DeleteSantaSyncPendingRules(
			ctx,
			sqlc.DeleteSantaSyncPendingRulesParams{HostID: hostID},
		); err != nil {
			return err
		}
		return q.CompleteSantaSync(ctx, sqlc.CompleteSantaSyncParams{
			HostID:          hostID,
			ClientRulesHash: clientRulesHash,
			RulesReceived:   rulesReceived,
			RulesProcessed:  rulesProcessed,
			PendingFullSync: pendingFullSync,
		})
	})
}

func syncStateExists(ctx context.Context, q *sqlc.Queries, hostID int64) (bool, error) {
	return q.SantaSyncStateExists(ctx, sqlc.SantaSyncStateExistsParams{HostID: hostID})
}

func loadTargets(ctx context.Context, q *sqlc.Queries, hostID int64, phase string) ([]Target, error) {
	rows, err := q.ListSantaSyncTargets(ctx, sqlc.ListSantaSyncTargetsParams{
		HostID: hostID,
		Phase:  sqlc.SantaSyncTargetPhase(phase),
	})
	if err != nil {
		return nil, err
	}
	targets := make([]Target, len(rows))
	for i, row := range rows {
		targets[i] = targetFromSQLC(row)
	}
	return sortedTargets(targets), nil
}

func insertTargets(ctx context.Context, q *sqlc.Queries, hostID int64, phase string, targets []Target) error {
	for position, target := range sortedTargets(targets) {
		if err := q.InsertSantaSyncTarget(ctx, sqlc.InsertSantaSyncTargetParams{
			HostID:              hostID,
			Phase:               sqlc.SantaSyncTargetPhase(phase),
			Position:            int32(position),
			RuleType:            sqlc.SantaRuleType(target.RuleType),
			Identifier:          target.Identifier,
			Policy:              sqlc.SantaPolicy(target.Policy),
			CelExpression:       target.CELExpression,
			CustomMessage:       target.CustomMessage,
			CustomURL:           target.CustomURL,
			NotificationAppName: target.AppName,
			PayloadHash:         target.PayloadHash,
		}); err != nil {
			return err
		}
	}
	return nil
}

func insertPayloadRules(ctx context.Context, q *sqlc.Queries, hostID int64, rules []PayloadRule) error {
	for position, rule := range sortedPayloadRules(rules) {
		var policy *sqlc.SantaPolicy
		if !rule.Removed {
			value := sqlc.SantaPolicy(rule.Policy)
			policy = &value
		}
		if err := q.InsertSantaSyncPendingRule(ctx, sqlc.InsertSantaSyncPendingRuleParams{
			HostID:              hostID,
			Position:            int32(position),
			RuleType:            sqlc.SantaRuleType(rule.RuleType),
			Identifier:          rule.Identifier,
			Policy:              policy,
			CelExpression:       rule.CELExpression,
			CustomMessage:       rule.CustomMessage,
			CustomURL:           rule.CustomURL,
			NotificationAppName: rule.AppName,
			PayloadHash:         rule.PayloadHash,
			Removed:             rule.Removed,
		}); err != nil {
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

func targetFromSQLC(row sqlc.ListSantaSyncTargetsRow) Target {
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

func payloadRuleFromSQLC(row sqlc.ListSantaPendingPayloadPageRow) PayloadRule {
	return PayloadRule{
		RuleType:      row.RuleType,
		Identifier:    row.Identifier,
		Policy:        row.Policy,
		CELExpression: row.CelExpression,
		CustomMessage: row.CustomMessage,
		CustomURL:     row.CustomURL,
		AppName:       row.NotificationAppName,
		PayloadHash:   row.PayloadHash,
		Removed:       row.Removed,
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
