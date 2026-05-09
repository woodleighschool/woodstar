package osquery

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/woodleighschool/woodstar/internal/models"
	queryinfra "github.com/woodleighschool/woodstar/internal/queries"
)

// queryKind tags each kind of query Woodstar emits to osquery agents.
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

// parseQueryName splits a Woodstar query name into kind and suffix.
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
	case kindDetail, kindLabel, kindCheck, kindLive, kindReport:
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

// statusOK reports whether an osquery status payload represents success.
func statusOK(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return number == 0
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text == "" || text == "0"
	}
	return false
}

// dispatchPass accumulates per-kind state during one DistributedWrite call.
// Detail and label kinds aggregate cross-row state finalized after the loop;
// check and campaign are pure per-row writes.
type dispatchPass struct {
	registry           map[string]DetailQuery
	detailRowsBySuffix map[string][]map[string]string
	detailAllSucceeded bool

	labelResults []labelQueryResult
	labelIDs     []int64
}

type labelQueryResult struct {
	labelID int64
	matched bool
}

// dispatchWriteResults runs a single pass over req.Queries, routing each
// result to its kind handler, then finalizes detail and label state.
func (s *Service) dispatchWriteResults(
	ctx context.Context,
	host *models.Host,
	req DistributedWriteRequest,
) error {
	pass := &dispatchPass{
		registry:           DetailQueries(),
		detailRowsBySuffix: make(map[string][]map[string]string),
		detailAllSucceeded: true,
	}

	for name, rows := range req.Queries {
		kind, suffix, ok := parseQueryName(name)
		if !ok {
			continue
		}
		status := req.Statuses[name]
		message := req.Messages[name]

		var err error
		switch kind {
		case kindDetail:
			err = s.handleDetailResult(ctx, host.ID, suffix, rows, status, message, pass)
		case kindLabel:
			s.handleLabelResult(ctx, host.ID, suffix, rows, status, message, pass)
		case kindCheck:
			err = s.handleCheckResult(ctx, host.ID, suffix, rows, status, message)
		case kindLive:
			err = s.handleLiveResult(ctx, host, suffix, rows, status, message)
		case kindReport:
			// reports flow via /log, not /distributed/write — silently ignore.
		}
		if err != nil {
			return fmt.Errorf("ingest %s: %w", name, err)
		}
	}

	if err := s.finalizeDetailPass(ctx, host, req, pass); err != nil {
		return err
	}
	return s.finalizeLabelPass(ctx, host, pass)
}

// ----- detail -----

func (s *Service) handleDetailResult(
	ctx context.Context,
	hostID int64,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	message string,
	pass *dispatchPass,
) error {
	// software_macos depends on cross-query enrichment — keep the rows for
	// finalize, but do not run its registry ingest now.
	pass.detailRowsBySuffix[suffix] = rows

	query, ok := pass.registry[suffix]
	if !ok {
		return nil
	}
	if !statusOK(status) {
		if !query.Optional {
			pass.detailAllSucceeded = false
		}
		s.logger.WarnContext(
			ctx,
			"osquery detail query failed", "operation", "distributed_write",
			"host_id", hostID,
			"query", queryName(kindDetail, suffix),
			"optional", query.Optional,
			"message", message,
		)
		return nil
	}
	if suffix == querySoftwareMacOS {
		return nil
	}
	return query.Ingest(ctx, s, hostID, rows)
}

func (s *Service) finalizeDetailPass(
	ctx context.Context,
	host *models.Host,
	req DistributedWriteRequest,
	pass *dispatchPass,
) error {
	if rows, ok := pass.detailRowsBySuffix[querySoftwareMacOS]; ok &&
		statusOK(req.Statuses[queryName(kindDetail, querySoftwareMacOS)]) {
		if err := ingestSoftwareMacOSWithEnrichment(ctx, s, host.ID, rows, pass.detailRowsBySuffix); err != nil {
			return fmt.Errorf("ingest %s: %w", querySoftwareMacOS, err)
		}
	}
	if !pass.detailAllSucceeded || !sawEveryRequiredDetailQuery(req, pass.registry) {
		return nil
	}
	if err := s.hosts.MarkDetailFresh(ctx, host.ID, detailQueryHash()); err != nil {
		return err
	}
	s.logger.DebugContext(
		ctx,
		"osquery detail inventory refreshed", "operation", "inventory_refresh",
		"host_id", host.ID,
		"query_count", len(req.Queries),
	)
	return nil
}

func sawEveryRequiredDetailQuery(req DistributedWriteRequest, registry map[string]DetailQuery) bool {
	for name, query := range registry {
		if query.Optional {
			continue
		}
		emitted := queryName(kindDetail, name)
		if _, ok := req.Queries[emitted]; !ok || !statusOK(req.Statuses[emitted]) {
			return false
		}
	}
	return true
}

// ----- check -----

func (s *Service) handleCheckResult(
	ctx context.Context,
	hostID int64,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	message string,
) error {
	if s.checks == nil {
		return nil
	}
	checkID, ok := parsePositiveSuffix(suffix)
	if !ok {
		return nil
	}
	var passes *bool
	if statusOK(status) {
		value := len(rows) > 0
		passes = &value
	} else {
		s.logger.WarnContext(
			ctx,
			"osquery check query failed", "operation", "check_evaluation",
			"host_id", hostID,
			"check_id", checkID,
			"message", message,
		)
	}
	return s.checks.UpsertMembership(ctx, checkID, hostID, passes)
}

// ----- live query -----

func (s *Service) handleLiveResult(
	_ context.Context,
	host *models.Host,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	message string,
) error {
	if s.live == nil {
		return nil
	}
	queryID, ok := parsePositiveSuffix(suffix)
	if !ok {
		return nil
	}
	resultStatus := queryinfra.LiveStatusSuccess
	var data json.RawMessage
	if statusOK(status) {
		encoded, err := json.Marshal(rows)
		if err != nil {
			return fmt.Errorf("marshal live query rows: %w", err)
		}
		data = encoded
	} else {
		resultStatus = queryinfra.LiveStatusError
	}
	s.live.RecordResult(queryID, host.ID, host.DisplayName, resultStatus, data, message)
	return nil
}
