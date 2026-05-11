package osquery

import (
	"context"
	"encoding/json"

	"github.com/woodleighschool/woodstar/internal/agents/ingest"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

func (s *Service) queueLabelQueries(ctx context.Context, host *hosts.Host, queryMap map[string]string) (int, error) {
	labelRows, err := s.labelEvaluator.ApplicableLabels(ctx, host)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, label := range labelRows {
		if label.Query == nil {
			continue
		}
		queryMap[queryNameID(kindLabel, label.ID)] = *label.Query
		count++
	}
	return count, nil
}

func (s *Service) handleLabelResult(
	ctx context.Context,
	hostID int64,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	message string,
	pass *dispatchPass,
) {
	labelID, ok := parsePositiveSuffix(suffix)
	if !ok {
		return
	}
	if !statusOK(status) {
		s.logger.WarnContext(
			ctx,
			"osquery label query failed", "operation", "label_evaluation",
			"host_id", hostID,
			"label_id", labelID,
			"query", queryNameID(kindLabel, labelID),
			"message", message,
		)
		return
	}
	pass.labelResults = append(pass.labelResults, ingest.LabelResult{LabelID: labelID, Matched: len(rows) > 0})
}

func (s *Service) finalizeLabelPass(ctx context.Context, host *hosts.Host, pass *dispatchPass) error {
	return s.labelEvaluator.Finalize(ctx, host, pass.labelResults)
}
