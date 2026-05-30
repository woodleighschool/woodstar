package ingest

import (
	"context"
	"log/slog"
	"testing"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
)

func TestLabelEvaluatorFinalizeUpdatesOnlyApplicableSuccessfulLabels(t *testing.T) {
	store := &fakelabelStore{applicable: map[int64]struct{}{1: {}, 2: {}}}
	evaluator := NewLabelEvaluator(store, slog.New(slog.DiscardHandler))
	host := &hosts.Host{ID: 9}

	results := []LabelResult{
		{LabelID: 1, Matched: true},
		{LabelID: 2, Matched: false},
		{LabelID: 3, Matched: true}, // not applicable — must be skipped
	}

	if err := evaluator.Finalize(context.Background(), host, results); err != nil {
		t.Fatalf("Finalize returned error: %v", err)
	}

	if len(store.setCalls) != 2 {
		t.Fatalf("set calls = %#v, want 2 applicable labels", store.setCalls)
	}
	if store.setCalls[0] != (fakeSetCall{labelID: 1, hostID: 9, matched: true}) {
		t.Fatalf("first set call = %#v", store.setCalls[0])
	}
	if store.setCalls[1] != (fakeSetCall{labelID: 2, hostID: 9, matched: false}) {
		t.Fatalf("second set call = %#v", store.setCalls[1])
	}
}

func TestLabelEvaluatorFinalizeNoOpOnEmpty(t *testing.T) {
	store := &fakelabelStore{applicable: map[int64]struct{}{1: {}}}
	evaluator := NewLabelEvaluator(store, slog.New(slog.DiscardHandler))
	host := &hosts.Host{ID: 5}

	if err := evaluator.Finalize(context.Background(), host, nil); err != nil {
		t.Fatalf("Finalize returned error: %v", err)
	}
	if len(store.setCalls) != 0 {
		t.Fatalf("expected no set calls on empty results, got %d", len(store.setCalls))
	}
}

type fakelabelStore struct {
	applicable map[int64]struct{}
	setCalls   []fakeSetCall
}

type fakeSetCall struct {
	labelID int64
	hostID  int64
	matched bool
}

func (s *fakelabelStore) ListApplicableDynamic(context.Context) ([]labels.Label, error) {
	return nil, nil
}

func (s *fakelabelStore) ApplicableDynamicIDs(
	_ context.Context,
	_ []int64,
) (map[int64]struct{}, error) {
	return s.applicable, nil
}

func (s *fakelabelStore) SetMembership(_ context.Context, labelID int64, hostID int64, matched bool) error {
	s.setCalls = append(s.setCalls, fakeSetCall{labelID: labelID, hostID: hostID, matched: matched})
	return nil
}
