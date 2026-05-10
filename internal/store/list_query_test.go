package store

import (
	"errors"
	"strings"
	"testing"
)

func TestListQueryOrderByAllowlist(t *testing.T) {
	params := CleanListParams(ListParams{
		Page:           2,
		PerPage:        25,
		OrderKey:       "last_seen_at",
		OrderDirection: "desc",
	})

	query, args, err := ListQuery{
		SelectSQL: "SELECT * FROM hosts",
		WhereSQL:  "WHERE deleted_at IS NULL",
		Args:      []any{"existing"},
		OrderKeys: map[string]OrderExpr{
			"display_name": {SQL: "lower(display_name)"},
			"last_seen_at": {SQL: "last_seen_at", NullsLast: true},
		},
		DefaultOrder: []OrderExpr{{SQL: "lower(display_name)"}, {SQL: "id"}},
		Params:       params,
	}.Build()
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if !strings.Contains(query, "ORDER BY last_seen_at DESC NULLS LAST, lower(display_name) ASC, id ASC") {
		t.Fatalf("query = %s", query)
	}
	if !strings.Contains(query, "LIMIT $2 OFFSET $3") {
		t.Fatalf("query = %s", query)
	}
	if len(args) != 3 || args[0] != "existing" || args[1] != int32(25) || args[2] != int32(25) {
		t.Fatalf("args = %#v", args)
	}
}

func TestListQueryRejectsUnknownOrderKey(t *testing.T) {
	_, _, err := ListQuery{
		SelectSQL: "SELECT * FROM hosts",
		OrderKeys: map[string]OrderExpr{
			"display_name": {SQL: "lower(display_name)"},
		},
		Params: CleanListParams(ListParams{OrderKey: "orbit_node_key"}),
	}.Build()
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if !strings.Contains(err.Error(), "unknown order key") {
		t.Fatalf("err = %v, want unknown order key", err)
	}
}
