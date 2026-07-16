package ingest

import (
	"context"
	"log/slog"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
)

// labelStore is what label evaluation needs.
type labelStore interface {
	ListApplicableDynamic(ctx context.Context) ([]labels.DynamicLabel, error)
	SetDynamicMemberships(ctx context.Context, labelID int64, memberships []labels.DynamicMembership) (int, error)
}

// LabelResult is one label match.
type LabelResult struct {
	LabelID int64
	Matched bool
}

// LabelEvaluator handles dynamic label results.
type LabelEvaluator struct {
	store  labelStore
	logger *slog.Logger
}

func NewLabelEvaluator(store labelStore, logger *slog.Logger) *LabelEvaluator {
	return &LabelEvaluator{store: store, logger: logger}
}

// ApplicableLabels returns labels with dynamic membership. Every host evaluates
// the same dynamic label set; per-host membership is decided by the query results.
func (e *LabelEvaluator) ApplicableLabels(ctx context.Context) ([]labels.DynamicLabel, error) {
	return e.store.ListApplicableDynamic(ctx)
}

// Finalize saves label results.
func (e *LabelEvaluator) Finalize(ctx context.Context, host *hosts.Host, results []LabelResult) error {
	if len(results) == 0 {
		return nil
	}
	memberships := make([]labels.DynamicMembership, len(results))
	for i, result := range results {
		memberships[i] = labels.DynamicMembership{LabelID: result.LabelID, Matched: result.Matched}
	}
	handled, err := e.store.SetDynamicMemberships(ctx, host.ID, memberships)
	if err != nil {
		return err
	}
	if handled == 0 {
		return nil
	}
	e.logger.DebugContext(ctx, "osquery label results ingested",
		"operation", "label_evaluation",
		"host_id", host.ID,
		"result_count", handled,
	)
	return nil
}
