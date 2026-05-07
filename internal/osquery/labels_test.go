package osquery

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/woodleighschool/woodstar/internal/models"
)

func TestLabelQueryNameRoundTrips(t *testing.T) {
	name := labelQueryName(42)
	if name != "woodstar_label_query_42" {
		t.Fatalf("labelQueryName = %q, want woodstar_label_query_42", name)
	}
	id, ok := parseLabelQueryName(name)
	if !ok {
		t.Fatalf("parseLabelQueryName(%q) ok = false, want true", name)
	}
	if id != 42 {
		t.Fatalf("parseLabelQueryName id = %d, want 42", id)
	}
}

func TestParseLabelQueryNameRejectsOtherQueries(t *testing.T) {
	for _, name := range []string{"system_info", "woodstar_label_query_", "woodstar_label_query_nope"} {
		if id, ok := parseLabelQueryName(name); ok || id != 0 {
			t.Fatalf("parseLabelQueryName(%q) = %d, %t; want 0, false", name, id, ok)
		}
	}
}

func TestIngestLabelResultsUpdatesOnlyApplicableSuccessfulLabels(t *testing.T) {
	labels := &fakeLabelStore{
		applicable: map[int64]struct{}{1: {}, 2: {}},
	}
	svc := &Service{labels: labels, logger: slog.New(slog.DiscardHandler)}

	err := svc.ingestLabelResults(
		context.Background(),
		&models.Host{ID: 9, Platform: "darwin"},
		DistributedWriteRequest{
			Queries: map[string][]map[string]string{
				labelQueryName(1): {{"matches": "yes"}},
				labelQueryName(2): {},
				labelQueryName(3): {{"stale": "ignored"}},
				labelQueryName(4): {{"failed": "preserved"}},
				"system_info":     {{"ignored": "true"}},
			},
			Statuses: map[string]json.RawMessage{
				labelQueryName(4): json.RawMessage(`1`),
			},
			Messages: map[string]string{
				labelQueryName(4): "constraint failed",
			},
		},
	)
	if err != nil {
		t.Fatalf("ingestLabelResults returned error: %v", err)
	}

	if len(labels.setCalls) != 2 {
		t.Fatalf("set calls = %#v, want 2 applicable successful labels", labels.setCalls)
	}
	if labels.setCalls[0] != (fakeSetCall{labelID: 1, hostID: 9, matched: true}) {
		t.Fatalf("first set call = %#v", labels.setCalls[0])
	}
	if labels.setCalls[1] != (fakeSetCall{labelID: 2, hostID: 9, matched: false}) {
		t.Fatalf("second set call = %#v", labels.setCalls[1])
	}
	if labels.markedHostID != 9 {
		t.Fatalf("markedHostID = %d, want 9", labels.markedHostID)
	}
}

type fakeLabelStore struct {
	applicable   map[int64]struct{}
	setCalls     []fakeSetCall
	markedHostID int64
}

type fakeSetCall struct {
	labelID int64
	hostID  int64
	matched bool
}

func (s *fakeLabelStore) ListApplicableDynamic(context.Context, string) ([]models.Label, error) {
	return nil, nil
}

func (s *fakeLabelStore) ApplicableDynamicIDs(
	_ context.Context,
	_ []int64,
	_ string,
) (map[int64]struct{}, error) {
	return s.applicable, nil
}

func (s *fakeLabelStore) SetMembership(_ context.Context, labelID int64, hostID int64, matched bool) error {
	s.setCalls = append(s.setCalls, fakeSetCall{labelID: labelID, hostID: hostID, matched: matched})
	return nil
}

func (s *fakeLabelStore) MarkHostLabelsFresh(_ context.Context, hostID int64) error {
	s.markedHostID = hostID
	return nil
}
