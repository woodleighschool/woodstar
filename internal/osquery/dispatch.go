package osquery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/munki"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
	"github.com/woodleighschool/woodstar/internal/osquery/ingest"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
	"github.com/woodleighschool/woodstar/internal/osquery/policies"
)

// queryKind tags our osquery work.
type queryKind string

const (
	kindDetail queryKind = "detail"
	kindLabel  queryKind = "label"
	kindPolicy queryKind = "policy"
	kindLive   queryKind = "live"
	kindReport queryKind = "report"
)

const namePrefix = "woodstar_"

func queryName(kind queryKind, suffix string) string {
	return namePrefix + string(kind) + "_query_" + suffix
}

func queryNameID(kind queryKind, id int64) string {
	return queryName(kind, strconv.FormatInt(id, 10))
}

// queryNameForSQL binds persisted results to the SQL text that produced them.
// A result for replaced SQL therefore cannot be mistaken for current state.
func queryNameForSQL(kind queryKind, id int64, sql string) string {
	return queryName(kind, strconv.FormatInt(id, 10)+"_"+queryHash(sql))
}

func queryNameForEvaluation(evaluation policies.Evaluation) string {
	return queryName(
		kindPolicy,
		strconv.FormatInt(evaluation.PolicyID, 10)+"_"+
			queryHash(evaluation.Query)+"_"+
			strconv.FormatInt(evaluation.Revision, 10)+"_"+
			strconv.FormatInt(evaluation.Sequence, 10),
	)
}

func queryHash(sql string) string {
	sum := sha256.Sum256([]byte(sql))
	return hex.EncodeToString(sum[:])
}

// parseQueryName splits our query name into kind and suffix.
func parseQueryName(name string) (queryKind, string, bool) {
	raw, ok := strings.CutPrefix(name, namePrefix)
	if !ok {
		return "", "", false
	}
	kindRaw, suffix, ok := strings.Cut(raw, "_query_")
	if !ok || suffix == "" {
		return "", "", false
	}
	kind := queryKind(kindRaw)
	switch kind {
	case kindDetail, kindLabel, kindPolicy, kindLive:
		return kind, suffix, true
	case kindReport:
		return "", "", false
	default:
		return "", "", false
	}
}

func parsePositiveSuffix(suffix string) (int64, bool) {
	id, err := strconv.ParseInt(suffix, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func parseQueryIdentity(suffix string) (int64, string, bool) {
	idRaw, hash, ok := strings.Cut(suffix, "_")
	if !ok || len(hash) != sha256.Size*2 {
		return 0, "", false
	}
	id, ok := parsePositiveSuffix(idRaw)
	if !ok {
		return 0, "", false
	}
	decoded, err := hex.DecodeString(hash)
	if err != nil || len(decoded) != sha256.Size {
		return 0, "", false
	}
	return id, hash, true
}

func parsePolicyEvaluationIdentity(suffix string) (int64, string, int64, int64, bool) {
	separator := strings.LastIndexByte(suffix, '_')
	if separator <= 0 || separator == len(suffix)-1 {
		return 0, "", 0, 0, false
	}
	revisionIdentity := suffix[:separator]
	sequenceRaw := suffix[separator+1:]
	revisionSeparator := strings.LastIndexByte(revisionIdentity, '_')
	if revisionSeparator <= 0 || revisionSeparator == len(revisionIdentity)-1 {
		return 0, "", 0, 0, false
	}
	identity := revisionIdentity[:revisionSeparator]
	revisionRaw := revisionIdentity[revisionSeparator+1:]
	policyID, hash, ok := parseQueryIdentity(identity)
	if !ok {
		return 0, "", 0, 0, false
	}
	revision, ok := parsePositiveSuffix(revisionRaw)
	if !ok {
		return 0, "", 0, 0, false
	}
	sequence, ok := parsePositiveSuffix(sequenceRaw)
	if !ok {
		return 0, "", 0, 0, false
	}
	return policyID, hash, revision, sequence, true
}

// detailDispatchPass accumulates detail-query state during one DistributedWrite call.
type detailDispatchPass struct {
	registry     map[string]catalog.DetailQuery
	results      map[string]detailResult
	allSucceeded bool
}

type detailResult struct {
	rows      []map[string]string
	status    json.RawMessage
	hasStatus bool
}

func newDetailDispatchPass() *detailDispatchPass {
	return &detailDispatchPass{
		registry:     catalog.DetailQueries(),
		results:      make(map[string]detailResult),
		allSucceeded: true,
	}
}

type labelDispatchPass struct {
	results []ingest.LabelResult
}

type policyDispatchPass struct {
	results []policies.EvaluationResult
}

// dispatchWriteResults runs a single pass over req.Queries, routing each
// result to its kind handler, then finalizes policy, detail, and label state.
func (s *AgentService) dispatchWriteResults(
	ctx context.Context,
	host *hosts.Host,
	req DistributedWriteRequest,
) error {
	details := newDetailDispatchPass()
	labels := &labelDispatchPass{}
	policyResults := &policyDispatchPass{}

	for name, rows := range req.Queries {
		kind, suffix, ok := parseQueryName(name)
		if !ok {
			continue
		}
		status, hasStatus := req.Statuses[name]
		message := req.Messages[name]

		var err error
		switch kind { //nolint:exhaustive // parseQueryName excludes report queries.
		case kindDetail:
			err = s.handleDetailResult(ctx, host.ID, suffix, rows, status, hasStatus, message, details)
		case kindLabel:
			s.handleLabelResult(ctx, host.ID, suffix, rows, status, hasStatus, message, labels)
		case kindPolicy:
			s.handlePolicyResult(ctx, host.ID, suffix, rows, status, hasStatus, message, policyResults)
		case kindLive:
			err = s.handleLiveResult(ctx, host, suffix, rows, status, hasStatus, message)
		}
		if err != nil {
			return fmt.Errorf("ingest %s: %w", name, err)
		}
	}

	if err := s.finalizePolicyPass(ctx, host.ID, policyResults); err != nil {
		return err
	}
	if err := s.finalizeDetailPass(ctx, host, details); err != nil {
		return err
	}
	return s.finalizeLabelPass(ctx, host, labels)
}

func (s *AgentService) handleLabelResult(
	ctx context.Context,
	hostID int64,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	hasStatus bool,
	message string,
	pass *labelDispatchPass,
) {
	labelID, ok := parsePositiveSuffix(suffix)
	if !ok {
		return
	}
	matched, ok := rowPresenceResult(status, hasStatus, rows)
	if !ok {
		s.deps.Logger.DebugContext(
			ctx,
			"osquery label query failed", "operation", "label_evaluation",
			"host_id", hostID,
			"label_id", labelID,
			"query", queryNameID(kindLabel, labelID),
			"message", message,
		)
		return
	}
	pass.results = append(pass.results, ingest.LabelResult{LabelID: labelID, Matched: matched})
}

func (s *AgentService) finalizeLabelPass(
	ctx context.Context,
	host *hosts.Host,
	pass *labelDispatchPass,
) error {
	return s.deps.LabelEvaluator.Finalize(ctx, host, pass.results)
}

func (s *AgentService) handleDetailResult(
	ctx context.Context,
	hostID int64,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	hasStatus bool,
	message string,
	pass *detailDispatchPass,
) error {
	pass.results[suffix] = detailResult{
		rows:      rows,
		status:    status,
		hasStatus: hasStatus,
	}

	query, ok := pass.registry[suffix]
	if !ok {
		return nil
	}
	if !distributedStatusOK(status, hasStatus) {
		if !query.Optional {
			pass.allSucceeded = false
		}
		level := slog.LevelWarn
		if query.Optional {
			level = slog.LevelDebug
		}
		s.deps.Logger.Log(
			ctx,
			level,
			"osquery detail query failed", "operation", "distributed_write",
			"host_id", hostID,
			"query", queryName(kindDetail, suffix),
			"optional", query.Optional,
			"message", message,
		)
		return nil
	}
	if query.Deferred() {
		return nil
	}
	return s.deps.InventoryProjector.IngestDetail(ctx, query, suffix, hostID, rows)
}

func (s *AgentService) finalizeDetailPass(
	ctx context.Context,
	host *hosts.Host,
	pass *detailDispatchPass,
) error {
	if softwareRows, ok := successfulSoftwareRows(pass); ok {
		if err := s.deps.InventoryProjector.IngestSoftware(ctx, host.ID, softwareRows); err != nil {
			return fmt.Errorf("ingest software inventory: %w", err)
		}
	}
	if pass.allSucceeded && sawEveryRequiredDetailQuery(pass) {
		if err := s.deps.InventoryProjector.MarkFresh(ctx, host.ID); err != nil {
			return err
		}
		s.deps.Logger.DebugContext(
			ctx,
			"osquery detail inventory refreshed", "operation", "inventory_refresh",
			"host_id", host.ID,
			"query_count", len(pass.results),
		)
	}
	return s.finalizeMunkiCollection(ctx, host.ID, pass)
}

func (s *AgentService) finalizeMunkiCollection(
	ctx context.Context,
	hostID int64,
	pass *detailDispatchPass,
) error {
	info, hasInfo := pass.results[catalog.QueryMunkiInfo]
	installs, hasInstalls := pass.results[catalog.QueryMunkiInstalls]
	if !hasInfo && !hasInstalls && !sawKnownDetailResult(pass) {
		return nil
	}
	if hasInfo != hasInstalls {
		s.deps.Logger.WarnContext(
			ctx,
			"incomplete Munki detail collection", "operation", "munki_collection",
			"host_id", hostID,
			"info_present", hasInfo,
			"installs_present", hasInstalls,
		)
	}
	if s.deps.MunkiCollector == nil {
		return fmt.Errorf("munki collector is not configured")
	}
	return s.deps.MunkiCollector.IngestCollection(ctx, hostID, munki.Collection{
		Info:     munkiQueryResult(hasInfo, info),
		Installs: munkiQueryResult(hasInstalls, installs),
	})
}

func sawKnownDetailResult(pass *detailDispatchPass) bool {
	for name := range pass.results {
		if _, ok := pass.registry[name]; ok {
			return true
		}
	}
	return false
}

func munkiQueryResult(present bool, result detailResult) munki.QueryResult {
	queryResult := munki.QueryResult{Present: present, Rows: result.rows}
	if !present {
		return queryResult
	}
	if distributedStatusOK(result.status, result.hasStatus) {
		queryResult.Successful = true
	}
	return queryResult
}

func successfulSoftwareRows(
	pass *detailDispatchPass,
) (map[string][]map[string]string, bool) {
	rowsBySuffix := make(map[string][]map[string]string)
	baseSucceeded := false
	for suffix, query := range pass.registry {
		if query.Ingest != catalog.IngestSoftwareBase && query.Ingest != catalog.IngestSoftwareEnrichment {
			continue
		}
		result, ok := pass.results[suffix]
		if !ok || !distributedStatusOK(result.status, result.hasStatus) {
			continue
		}
		rowsBySuffix[suffix] = result.rows
		if query.Ingest == catalog.IngestSoftwareBase {
			baseSucceeded = true
		}
	}
	return rowsBySuffix, baseSucceeded
}

func sawEveryRequiredDetailQuery(pass *detailDispatchPass) bool {
	for name, query := range pass.registry {
		if query.Optional {
			continue
		}
		result, ok := pass.results[name]
		if !ok || !distributedStatusOK(result.status, result.hasStatus) {
			return false
		}
	}
	return true
}

func (s *AgentService) handlePolicyResult(
	ctx context.Context,
	hostID int64,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	hasStatus bool,
	message string,
	pass *policyDispatchPass,
) {
	policyID, queryHash, revision, sequence, ok := parsePolicyEvaluationIdentity(suffix)
	if !ok {
		return
	}
	matched, ok := rowPresenceResult(status, hasStatus, rows)
	result := policies.EvaluationResult{
		PolicyID:  policyID,
		QueryHash: queryHash,
		Revision:  revision,
		Sequence:  sequence,
		Status:    policies.PolicyStatusError,
		Error:     message,
	}
	if ok {
		if matched {
			result.Status = policies.PolicyStatusPass
		} else {
			result.Status = policies.PolicyStatusFail
		}
		result.Error = ""
	} else {
		s.deps.Logger.DebugContext(
			ctx,
			"osquery policy query failed", "operation", "policy_evaluation",
			"host_id", hostID,
			"policy_id", policyID,
			"message", message,
		)
	}
	pass.results = append(pass.results, result)
}

func (s *AgentService) finalizePolicyPass(
	ctx context.Context,
	hostID int64,
	pass *policyDispatchPass,
) error {
	return s.deps.PolicyStore.RecordEvaluations(ctx, hostID, pass.results)
}

func rowPresenceResult(status json.RawMessage, hasStatus bool, rows []map[string]string) (bool, bool) {
	if !distributedStatusOK(status, hasStatus) {
		return false, false
	}
	return len(rows) > 0, true
}

func distributedStatusOK(raw json.RawMessage, hasStatus bool) bool {
	if !hasStatus {
		return false
	}
	var number int
	if err := json.Unmarshal(raw, &number); err != nil {
		return false
	}
	return number == 0
}

func (s *AgentService) handleLiveResult(
	ctx context.Context,
	host *hosts.Host,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	hasStatus bool,
	message string,
) error {
	queryID, ok := parsePositiveSuffix(suffix)
	if !ok {
		return nil
	}
	resultStatus := livequery.StatusCollected
	if !distributedStatusOK(status, hasStatus) {
		resultStatus = livequery.StatusError
	}
	return s.deps.LiveQueries.RecordResult(ctx, livequery.Result{
		QueryID:  queryID,
		HostID:   host.ID,
		HostName: host.DisplayName,
		Status:   resultStatus,
		Rows:     rows,
		Error:    message,
	})
}
