package ingest

import (
	"cmp"
	"context"
	"log/slog"
	"slices"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
	"github.com/woodleighschool/woodstar/internal/scope"
)

// labelStore is what label evaluation needs.
type labelStore interface {
	ListApplicableDynamic(context.Context, scope.Platform) ([]labels.Label, error)
	ApplicableDynamicIDs(context.Context, []int64, scope.Platform) (map[int64]struct{}, error)
	SetMembership(context.Context, int64, int64, bool) error
	MarkHostLabelsFresh(context.Context, int64) error
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

// ApplicableLabels returns labels for the host family.
func (e *LabelEvaluator) ApplicableLabels(ctx context.Context, host *hosts.Host) ([]labels.Label, error) {
	return e.store.ListApplicableDynamic(ctx, host.Platform)
}

// Finalize saves label results.
func (e *LabelEvaluator) Finalize(ctx context.Context, host *hosts.Host, results []LabelResult) error {
	if len(results) == 0 {
		return nil
	}
	slices.SortFunc(results, func(a, b LabelResult) int {
		return cmp.Compare(a.LabelID, b.LabelID)
	})
	ids := make([]int64, 0, len(results))
	for _, r := range results {
		ids = append(ids, r.LabelID)
	}
	applicable, err := e.store.ApplicableDynamicIDs(ctx, ids, host.Platform)
	if err != nil {
		return err
	}
	handled := 0
	for _, r := range results {
		if _, ok := applicable[r.LabelID]; !ok {
			continue
		}
		if err := e.store.SetMembership(ctx, r.LabelID, host.ID, r.Matched); err != nil {
			return err
		}
		handled++
	}
	if handled == 0 {
		return nil
	}
	if err := e.store.MarkHostLabelsFresh(ctx, host.ID); err != nil {
		return err
	}
	e.logger.DebugContext(ctx, "osquery label results ingested",
		"operation", "label_evaluation",
		"host_id", host.ID,
		"result_count", handled,
	)
	return nil
}
