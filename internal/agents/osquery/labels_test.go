package osquery

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/woodleighschool/woodstar/internal/agents/ingest"
	"github.com/woodleighschool/woodstar/internal/labels"
)

func TestHandleLabelResultStatusFilter(t *testing.T) {
	svc := &Service{
		labelEvaluator: ingest.NewLabelEvaluator(&fakeLabelEvaluatorStore{}, slog.New(slog.DiscardHandler)),
		logger:         slog.New(slog.DiscardHandler),
	}
	rows := []map[string]string{{"col": "val"}}

	t.Run("failed status skips accumulation", func(t *testing.T) {
		pass := &dispatchPass{}
		svc.handleLabelResult(context.Background(), 1, "10", rows, json.RawMessage("1"), "", pass)
		if len(pass.labelResults) != 0 {
			t.Fatalf("labelResults = %v, want empty", pass.labelResults)
		}
	})

	t.Run("success status appends matched result", func(t *testing.T) {
		pass := &dispatchPass{}
		svc.handleLabelResult(context.Background(), 1, "10", rows, json.RawMessage("0"), "", pass)
		if len(pass.labelResults) != 1 {
			t.Fatalf("labelResults len = %d, want 1", len(pass.labelResults))
		}
		if pass.labelResults[0].LabelID != 10 || !pass.labelResults[0].Matched {
			t.Fatalf("labelResults[0] = %+v, want {LabelID:10 Matched:true}", pass.labelResults[0])
		}
	})
}

// fakeLabelEvaluatorStore satisfies ingest.LabelStore with no-op behaviour.
type fakeLabelEvaluatorStore struct{}

func (f *fakeLabelEvaluatorStore) ListApplicableDynamic(context.Context, string) ([]labels.Label, error) {
	return nil, nil
}

func (f *fakeLabelEvaluatorStore) ApplicableDynamicIDs(context.Context, []int64, string) (map[int64]struct{}, error) {
	return nil, nil
}

func (f *fakeLabelEvaluatorStore) SetMembership(context.Context, int64, int64, bool) error {
	return nil
}

func (f *fakeLabelEvaluatorStore) MarkHostLabelsFresh(context.Context, int64) error {
	return nil
}
