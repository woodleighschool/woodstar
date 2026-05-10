package osquery

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
)

type labelStore interface {
	ListApplicableDynamic(context.Context, string) ([]labels.Label, error)
	ApplicableDynamicIDs(context.Context, []int64, string) (map[int64]struct{}, error)
	SetMembership(context.Context, int64, int64, bool) error
	MarkHostLabelsFresh(context.Context, int64) error
}

func (s *Service) queueLabelQueries(ctx context.Context, host *hosts.Host, queries map[string]string) (int, error) {
	if s.labels == nil {
		return 0, nil
	}
	labels, err := s.labels.ListApplicableDynamic(ctx, host.Platform)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, label := range labels {
		if label.Query == nil {
			continue
		}
		queries[queryNameID(kindLabel, label.ID)] = *label.Query
		count++
	}
	return count, nil
}

// handleLabelResult accumulates per-label result state for finalize. It never
// errors at this stage because applicability checks and DB writes happen after
// the full request has been parsed.
func (s *Service) handleLabelResult(
	ctx context.Context,
	hostID int64,
	suffix string,
	rows []map[string]string,
	status json.RawMessage,
	message string,
	pass *dispatchPass,
) {
	if s.labels == nil {
		return
	}
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
	pass.labelResults = append(pass.labelResults, labelQueryResult{labelID: labelID, matched: len(rows) > 0})
	pass.labelIDs = append(pass.labelIDs, labelID)
}

func (s *Service) finalizeLabelPass(ctx context.Context, host *hosts.Host, pass *dispatchPass) error {
	if s.labels == nil || len(pass.labelResults) == 0 {
		return nil
	}
	slices.SortFunc(pass.labelResults, func(a, b labelQueryResult) int {
		return int(a.labelID - b.labelID)
	})
	applicable, err := s.labels.ApplicableDynamicIDs(ctx, pass.labelIDs, host.Platform)
	if err != nil {
		return err
	}
	handled := 0
	for _, result := range pass.labelResults {
		if _, ok := applicable[result.labelID]; !ok {
			continue
		}
		if err := s.labels.SetMembership(ctx, result.labelID, host.ID, result.matched); err != nil {
			return err
		}
		handled++
	}
	if handled == 0 {
		return nil
	}
	if err := s.labels.MarkHostLabelsFresh(ctx, host.ID); err != nil {
		return err
	}
	s.logger.DebugContext(
		ctx,
		"osquery label results ingested", "operation", "label_evaluation",
		"host_id", host.ID,
		"result_count", handled,
	)
	return nil
}
