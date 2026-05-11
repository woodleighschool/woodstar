package osquery

import (
	"encoding/json"
	"testing"

	"github.com/woodleighschool/woodstar/internal/agents/catalog"
)

func TestQueryNameRoundTrips(t *testing.T) {
	tests := []struct {
		name   string
		kind   queryKind
		suffix string
	}{
		{name: "detail", kind: kindDetail, suffix: "system_info"},
		{name: "label", kind: kindLabel, suffix: "42"},
		{name: "check", kind: kindCheck, suffix: "7"},
		{name: "live", kind: kindLive, suffix: "3"},
		{name: "report", kind: kindReport, suffix: "15"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			name := queryName(tt.kind, tt.suffix)
			gotKind, gotSuffix, ok := parseQueryName(name)
			if !ok {
				t.Fatalf("parseQueryName(%q) ok = false, want true", name)
			}
			if gotKind != tt.kind || gotSuffix != tt.suffix {
				t.Fatalf("parseQueryName(%q) = %q, %q; want %q, %q", name, gotKind, gotSuffix, tt.kind, tt.suffix)
			}
		})
	}
}

func TestParseQueryNameRejectsUnknownNames(t *testing.T) {
	for _, name := range []string{
		"system_info",
		"woodstar_label_query_",
		"woodstar_unknown_query_1",
		"fleet_detail_query_system_info",
	} {
		if kind, suffix, ok := parseQueryName(name); ok || kind != "" || suffix != "" {
			t.Fatalf("parseQueryName(%q) = %q, %q, %t; want zero values", name, kind, suffix, ok)
		}
	}
}

func TestSawEveryRequiredDetailQueryRequiresPresenceAndStatus(t *testing.T) {
	registry := map[string]catalog.DetailQuery{
		"required": {},
		"optional": {Optional: true},
	}
	if sawEveryRequiredDetailQuery(DistributedWriteRequest{Queries: map[string][]map[string]string{}}, registry) {
		t.Fatal("missing required query was treated as complete")
	}
	if sawEveryRequiredDetailQuery(
		DistributedWriteRequest{
			Queries:  map[string][]map[string]string{detailQueryName("required"): {}},
			Statuses: map[string]json.RawMessage{detailQueryName("required"): json.RawMessage(`1`)},
		},
		registry,
	) {
		t.Fatal("failed required query was treated as complete")
	}
	if !sawEveryRequiredDetailQuery(
		DistributedWriteRequest{Queries: map[string][]map[string]string{detailQueryName("required"): {}}},
		registry,
	) {
		t.Fatal("empty successful required query was not treated as complete")
	}
}
