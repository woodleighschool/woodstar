package dbutil

import (
	"errors"
	"strings"
	"testing"
)

func TestListQueryOrderByAllowlist(t *testing.T) {
	params := CleanListParams(ListParams{
		PageIndex: 1,
		PageSize:  25,
		Sort:      "last_seen_at.desc",
	})

	query, args, err := ListQuery{
		SelectSQL: "SELECT * FROM hosts",
		WhereSQL:  "WHERE deleted_at IS NULL",
		Args:      []any{"existing"},
		OrderKeys: map[string]OrderExpr{
			"display_name": {SQL: "lower(display_name)"},
			"last_seen_at": {SQL: "last_seen_at", NullOrder: NullsLast},
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

func TestListQueryRejectsUnknownSortKey(t *testing.T) {
	_, _, err := ListQuery{
		SelectSQL: "SELECT * FROM hosts",
		OrderKeys: map[string]OrderExpr{
			"display_name": {SQL: "lower(display_name)"},
		},
		Params: CleanListParams(ListParams{Sort: "orbit_node_key"}),
	}.Build()
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if !strings.Contains(err.Error(), "unknown sort key") {
		t.Fatalf("err = %v, want unknown sort key", err)
	}
}

func TestListQueryNestedSortKey(t *testing.T) {
	query, _, err := ListQuery{
		SelectSQL: "SELECT * FROM hosts",
		OrderKeys: map[string]OrderExpr{
			"hardware.serial": {SQL: "lower(hardware_serial)"},
		},
		Params: CleanListParams(ListParams{Sort: "hardware.serial.desc"}),
	}.Build()
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if !strings.Contains(query, "ORDER BY lower(hardware_serial) DESC") {
		t.Fatalf("query = %s", query)
	}
}

func TestWhereBuilderBuildsClausesWithStablePlaceholders(t *testing.T) {
	var where WhereBuilder
	search := where.Arg("%mac%")
	where.Add("(display_name ILIKE " + search + " OR hardware_serial ILIKE " + search + ")")
	status := where.Arg("online")
	where.Add("status = " + status)

	query, args := where.Build()

	wantQuery := "WHERE (display_name ILIKE $1 OR hardware_serial ILIKE $1) AND status = $2"
	if query != wantQuery {
		t.Fatalf("query = %q, want %q", query, wantQuery)
	}
	if len(args) != 2 || args[0] != "%mac%" || args[1] != "online" {
		t.Fatalf("args = %#v", args)
	}
}
