package osquery

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/osquery/catalog"
	"github.com/woodleighschool/woodstar/internal/osquery/ingest"
	"github.com/woodleighschool/woodstar/internal/osquery/livequery"
)

// queryKind tags our osquery work.
type queryKind string

const (
	kindDetail queryKind = "detail"
	kindLabel  queryKind = "label"
	kindCheck  queryKind = "check"
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

func detailQueryName(suffix string) string {
	return queryName(kindDetail, suffix)
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
	case kindDetail, kindLabel, kindCheck, kindLive:
		return kind, suffix, true
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

// dispatchWriteResults runs a single pass over req.Queries, routing each
// result to its kind handler, then finalizes detail and label state.
func (s *AgentService) dispatchWriteResults(
	ctx context.Context,
	host *hosts.Host,
	req DistributedWriteRequest,
) error {
	details := newDetailDispatchPass()
	labels := &labelDispatchPass{}

	for name, rows := range req.Queries {
		kind, suffix, ok := parseQueryName(name)
		if !ok {
			continue
		}
		status, hasStatus := req.Statuses[name]
		message := req.Messages[name]

		var err error
		switch kind { //nolint:exhaustive // parseQueryName already narrowed to the four dispatched kinds
		case kindDetail:
			err = s.handleDetailResult(ctx, host.ID, suffix, rows, status, hasStatus, message, details)
		case kindLabel:
			s.handleLabelResult(ctx, host.ID, suffix, rows, status, hasStatus, message, labels)
		case kindCheck:
			err = s.handleCheckResult(ctx, host.ID, suffix, rows, status, hasStatus, message)
		case kindLive:
			err = s.handleLiveResult(host, suffix, rows, status, hasStatus, message)
		}
		if err != nil {
			return fmt.Errorf("ingest %s: %w", name, err)
		}
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
		s.deps.Logger.WarnContext(
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
	pass.results[suffix] = detailResult{rows: rows, status: status, hasStatus: hasStatus}

	query, ok := pass.registry[suffix]
	if !ok {
		return nil
	}
	if !distributedStatusOK(status, hasStatus) {
		if !query.Optional {
			pass.allSucceeded = false
		}
		s.deps.Logger.WarnContext(
			ctx,
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
	if err := s.clearMissingOrFailedMunkiDetails(ctx, host.ID, pass); err != nil {
		return err
	}
	if !pass.allSucceeded || !sawEveryRequiredDetailQuery(pass) {
		return nil
	}
	if err := s.deps.InventoryProjector.MarkFresh(ctx, host.ID); err != nil {
		return err
	}
	s.deps.Logger.DebugContext(
		ctx,
		"osquery detail inventory refreshed", "operation", "inventory_refresh",
		"host_id", host.ID,
		"query_count", len(pass.results),
	)
	return nil
}

func (s *AgentService) clearMissingOrFailedMunkiDetails(
	ctx context.Context,
	hostID int64,
	pass *detailDispatchPass,
) error {
	if !sawRequiredDetailQuery(pass) {
		return nil
	}
	for _, name := range []string{catalog.QueryMunkiInfo, catalog.QueryMunkiInstalls} {
		query, ok := pass.registry[name]
		if !ok {
			continue
		}
		result, ok := pass.results[name]
		if ok && distributedStatusOK(result.status, result.hasStatus) {
			continue
		}
		if err := s.deps.InventoryProjector.IngestDetail(ctx, query, name, hostID, nil); err != nil {
			return fmt.Errorf("clear stale %s detail: %w", name, err)
		}
	}
	return nil
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

func sawRequiredDetailQuery(pass *detailDispatchPass) bool {
	for name, query := range pass.registry {
		if query.Optional {
			continue
		}
		if _, ok := pass.results[name]; ok {
			return true
		}
	}
	return false
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

func (s *AgentService) handleCheckResult(
	ctx context.Context,
	hostID int64,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	hasStatus bool,
	message string,
) error {
	checkID, ok := parsePositiveSuffix(suffix)
	if !ok {
		return nil
	}
	matched, ok := rowPresenceResult(status, hasStatus, rows)
	var passes *bool
	if ok {
		passes = &matched
	} else {
		s.deps.Logger.WarnContext(
			ctx,
			"osquery check query failed", "operation", "check_evaluation",
			"host_id", hostID,
			"check_id", checkID,
			"message", message,
		)
	}
	return s.deps.CheckStore.UpsertMembership(ctx, checkID, hostID, passes)
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
	resultStatus := livequery.StatusSuccess
	var data json.RawMessage
	if distributedStatusOK(status, hasStatus) {
		encoded, err := json.Marshal(rows)
		if err != nil {
			return fmt.Errorf("marshal live query rows: %w", err)
		}
		data = encoded
	} else {
		resultStatus = livequery.StatusError
	}
	s.deps.LiveQueries.RecordResult(livequery.Result{
		QueryID:  queryID,
		HostID:   host.ID,
		HostName: host.DisplayName,
		Status:   resultStatus,
		Data:     data,
		Error:    message,
	})
	return nil
}
