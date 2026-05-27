package osquery

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

// detailDispatchPass accumulates detail-query state during one DistributedWrite call.
type detailDispatchPass struct {
	registry     map[string]catalog.DetailQuery
	results      map[string]detailResult
	allSucceeded bool
}

type detailResult struct {
	rows   []map[string]string
	status json.RawMessage
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
func (s *Service) dispatchWriteResults(
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
		status := req.Statuses[name]
		message := req.Messages[name]

		var err error
		switch kind { //nolint:exhaustive // parseQueryName narrows to the four handled kinds.
		case kindDetail:
			err = handleDetailResult(
				ctx,
				s.logger,
				s.inventoryProjector,
				host.ID,
				suffix,
				rows,
				status,
				message,
				details,
			)
		case kindLabel:
			handleLabelResult(ctx, s.logger, host.ID, suffix, rows, status, message, labels)
		case kindCheck:
			err = handleCheckResult(ctx, s.logger, s.checkStore, host.ID, suffix, rows, status, message)
		case kindLive:
			err = handleLiveResult(s.liveQueries, host, suffix, rows, status, message)
		}
		if err != nil {
			return fmt.Errorf("ingest %s: %w", name, err)
		}
	}

	if err := finalizeDetailPass(ctx, s.logger, s.inventoryProjector, host, details); err != nil {
		return err
	}
	return finalizeLabelPass(ctx, s.labelEvaluator, host, labels)
}

func handleLabelResult(
	ctx context.Context,
	logger *slog.Logger,
	hostID int64,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	message string,
	pass *labelDispatchPass,
) {
	labelID, ok := parsePositiveSuffix(suffix)
	if !ok {
		return
	}
	if !statusOK(status) {
		logger.WarnContext(
			ctx,
			"osquery label query failed", "operation", "label_evaluation",
			"host_id", hostID,
			"label_id", labelID,
			"query", queryNameID(kindLabel, labelID),
			"message", message,
		)
		return
	}
	pass.results = append(pass.results, ingest.LabelResult{LabelID: labelID, Matched: len(rows) > 0})
}

func finalizeLabelPass(
	ctx context.Context,
	labelEvaluator labelEvaluator,
	host *hosts.Host,
	pass *labelDispatchPass,
) error {
	return labelEvaluator.Finalize(ctx, host, pass.results)
}

func handleDetailResult(
	ctx context.Context,
	logger *slog.Logger,
	inventoryProjector inventoryProjector,
	hostID int64,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	message string,
	pass *detailDispatchPass,
) error {
	pass.results[suffix] = detailResult{rows: rows, status: status}

	query, ok := pass.registry[suffix]
	if !ok {
		return nil
	}
	if !statusOK(status) {
		if !query.Optional {
			pass.allSucceeded = false
		}
		logger.WarnContext(
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
	return inventoryProjector.IngestDetail(ctx, query, suffix, hostID, rows)
}

func finalizeDetailPass(
	ctx context.Context,
	logger *slog.Logger,
	inventoryProjector inventoryProjector,
	host *hosts.Host,
	pass *detailDispatchPass,
) error {
	if softwareRows, ok := successfulSoftwareRows(pass); ok {
		if err := inventoryProjector.IngestSoftware(ctx, host.ID, softwareRows); err != nil {
			return fmt.Errorf("ingest software inventory: %w", err)
		}
	}
	if !pass.allSucceeded || !sawEveryRequiredDetailQuery(pass) {
		return nil
	}
	if err := inventoryProjector.MarkFresh(ctx, host.ID); err != nil {
		return err
	}
	logger.DebugContext(
		ctx,
		"osquery detail inventory refreshed", "operation", "inventory_refresh",
		"host_id", host.ID,
		"query_count", len(pass.results),
	)
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
		if !ok || !statusOK(result.status) {
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
		if !ok || !statusOK(result.status) {
			return false
		}
	}
	return true
}

func handleCheckResult(
	ctx context.Context,
	logger *slog.Logger,
	checkStore checkStore,
	hostID int64,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	message string,
) error {
	checkID, ok := parsePositiveSuffix(suffix)
	if !ok {
		return nil
	}
	var passes *bool
	if statusOK(status) {
		passes = new(len(rows) > 0)
	} else {
		logger.WarnContext(
			ctx,
			"osquery check query failed", "operation", "check_evaluation",
			"host_id", hostID,
			"check_id", checkID,
			"message", message,
		)
	}
	return checkStore.UpsertMembership(ctx, checkID, hostID, passes)
}

func handleLiveResult(
	liveQueries liveQueries,
	host *hosts.Host,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	message string,
) error {
	queryID, ok := parsePositiveSuffix(suffix)
	if !ok {
		return nil
	}
	resultStatus := livequery.StatusSuccess
	var data json.RawMessage
	if statusOK(status) {
		encoded, err := json.Marshal(rows)
		if err != nil {
			return fmt.Errorf("marshal live query rows: %w", err)
		}
		data = encoded
	} else {
		resultStatus = livequery.StatusError
	}
	liveQueries.RecordResult(queryID, host.ID, host.DisplayName, resultStatus, data, message)
	return nil
}
