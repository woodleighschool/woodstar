package osquery

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/hosts"
	"github.com/woodleighschool/woodstar/internal/labels"
)

func TestLabelQueryNameRoundTrips(t *testing.T) {
	name := queryNameID(kindLabel, 42)
	if name != "woodstar_label_query_42" {
		t.Fatalf("queryNameID = %q, want woodstar_label_query_42", name)
	}
	kind, suffix, ok := parseQueryName(name)
	if !ok || kind != kindLabel {
		t.Fatalf("parseQueryName(%q) = %q, %q, %t; want label query", name, kind, suffix, ok)
	}
	id, _ := parsePositiveSuffix(suffix)
	if id != 42 {
		t.Fatalf("parsePositiveSuffix id = %d, want 42", id)
	}
}

func TestParseLabelQueryNameRejectsOtherQueries(t *testing.T) {
	for _, name := range []string{
		"system_info",
		"woodstar_label_query_",
		"woodstar_label_query_nope",
		"woodstar_check_query_42",
	} {
		kind, suffix, ok := parseQueryName(name)
		if ok && kind == kindLabel {
			if id, ok := parsePositiveSuffix(suffix); ok || id != 0 {
				t.Fatalf("parsePositiveSuffix(%q) = %d, %t; want 0, false", suffix, id, ok)
			}
		}
	}
}

func TestDispatchLabelResultsUpdatesOnlyApplicableSuccessfulLabels(t *testing.T) {
	labelStore := &fakeLabelStore{applicable: map[int64]struct{}{1: {}, 2: {}}}
	svc := &Service{labelStore: labelStore, logger: slog.New(slog.DiscardHandler)}

	err := svc.dispatchWriteResults(
		context.Background(),
		&hosts.Host{Host: sqlc.Host{ID: 9, Platform: "darwin"}},
		DistributedWriteRequest{
			Queries: map[string][]map[string]string{
				queryNameID(kindLabel, 1): {{"matches": "yes"}},
				queryNameID(kindLabel, 2): {},
				queryNameID(kindLabel, 3): {{"stale": "ignored"}},
				queryNameID(kindLabel, 4): {{"failed": "preserved"}},
				"system_info":             {{"ignored_unprefixed": "true"}},
			},
			Statuses: map[string]json.RawMessage{
				queryNameID(kindLabel, 4): json.RawMessage(`1`),
			},
			Messages: map[string]string{
				queryNameID(kindLabel, 4): "constraint failed",
			},
		},
	)
	if err != nil {
		t.Fatalf("dispatchWriteResults returned error: %v", err)
	}

	if len(labelStore.setCalls) != 2 {
		t.Fatalf("set calls = %#v, want 2 applicable successful labels", labelStore.setCalls)
	}
	if labelStore.setCalls[0] != (fakeSetCall{labelID: 1, hostID: 9, matched: true}) {
		t.Fatalf("first set call = %#v", labelStore.setCalls[0])
	}
	if labelStore.setCalls[1] != (fakeSetCall{labelID: 2, hostID: 9, matched: false}) {
		t.Fatalf("second set call = %#v", labelStore.setCalls[1])
	}
	if labelStore.markedHostID != 9 {
		t.Fatalf("markedHostID = %d, want 9", labelStore.markedHostID)
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

func (s *fakeLabelStore) ListApplicableDynamic(context.Context, string) ([]labels.Label, error) {
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
