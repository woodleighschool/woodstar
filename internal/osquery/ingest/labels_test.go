package ingest

import (
	"context"
	"log/slog"
	"testing"

	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
)

func TestLabelEvaluatorFinalizeReconcilesReturnedResults(t *testing.T) {
	store := &fakelabelStore{handled: 2}
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

	if store.hostID != 9 {
		t.Fatalf("host ID = %d, want 9", store.hostID)
	}
	want := []labels.DynamicMembership{
		{LabelID: 1, Matched: true},
		{LabelID: 2, Matched: false},
		{LabelID: 3, Matched: true},
	}
	if len(store.memberships) != len(want) {
		t.Fatalf("memberships = %#v, want %#v", store.memberships, want)
	}
	for i := range want {
		if store.memberships[i] != want[i] {
			t.Fatalf("membership %d = %#v, want %#v", i, store.memberships[i], want[i])
		}
	}
}

func TestLabelEvaluatorFinalizeNoOpOnEmpty(t *testing.T) {
	store := &fakelabelStore{}
	evaluator := NewLabelEvaluator(store, slog.New(slog.DiscardHandler))
	host := &hosts.Host{ID: 5}

	if err := evaluator.Finalize(context.Background(), host, nil); err != nil {
		t.Fatalf("Finalize returned error: %v", err)
	}
	if store.memberships != nil {
		t.Fatalf("memberships = %#v, want no store call", store.memberships)
	}
}

type fakelabelStore struct {
	handled     int
	hostID      int64
	memberships []labels.DynamicMembership
}

func (s *fakelabelStore) ListApplicableDynamic(context.Context) ([]labels.DynamicLabel, error) {
	return nil, nil
}

func (s *fakelabelStore) SetDynamicMemberships(
	_ context.Context,
	hostID int64,
	memberships []labels.DynamicMembership,
) (int, error) {
	s.hostID = hostID
	s.memberships = memberships
	return s.handled, nil
}
