package postgres

import (
	"errors"
	"strings"
	"testing"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/listing"
)

func TestListQueryOrderByAllowlist(t *testing.T) {
	params := listing.Normalize(listing.Params{
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
	if len(args) != 3 || args[0] != "existing" || args[1] != int32(25) || args[2] != int64(25) {
		t.Fatalf("args = %#v", args)
	}
}

func TestListQueryCalculatesOffsetWithoutInt32Overflow(t *testing.T) {
	_, args, err := ListQuery{
		SelectSQL: "SELECT * FROM hosts",
		Params: listing.Normalize(listing.Params{
			PageIndex: 2_147_483_647,
			PageSize:  1000,
		}),
	}.Build()
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if got, want := args[1], int64(2_147_483_647_000); got != want {
		t.Fatalf("offset = %#v, want %d", got, want)
	}
}

func TestListQueryUsesDefaultDescendingOrder(t *testing.T) {
	query, _, err := ListQuery{
		SelectSQL: "SELECT * FROM events",
		DefaultOrder: []OrderExpr{
			{SQL: "occurred_at", Descending: true},
			{SQL: "id", Descending: true},
		},
		Params: listing.Normalize(listing.Params{}),
	}.Build()
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if !strings.Contains(query, "ORDER BY occurred_at DESC, id DESC") {
		t.Fatalf("query = %s", query)
	}
}

func TestListQueryRejectsUnknownSortKey(t *testing.T) {
	_, _, err := ListQuery{
		SelectSQL: "SELECT * FROM hosts",
		OrderKeys: map[string]OrderExpr{
			"display_name": {SQL: "lower(display_name)"},
		},
		Params: listing.Normalize(listing.Params{Sort: "orbit_node_key"}),
	}.Build()
	if !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("err = %v, want fault.ErrInvalidInput", err)
	}
	if !strings.Contains(err.Error(), "unknown sort key") {
		t.Fatalf("err = %v, want unknown sort key", err)
	}
}

func TestListQueryRejectsMalformedSort(t *testing.T) {
	_, _, err := ListQuery{
		SelectSQL: "SELECT * FROM hosts",
		OrderKeys: map[string]OrderExpr{
			"display_name": {SQL: "lower(display_name)"},
		},
		Params: listing.Normalize(listing.Params{Sort: ".asc"}),
	}.Build()
	if !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("err = %v, want fault.ErrInvalidInput", err)
	}
}

func TestListQueryNestedSortKey(t *testing.T) {
	query, _, err := ListQuery{
		SelectSQL: "SELECT * FROM hosts",
		OrderKeys: map[string]OrderExpr{
			"hardware.serial": {SQL: "lower(hardware_serial)"},
		},
		Params: listing.Normalize(listing.Params{Sort: "hardware.serial.desc"}),
	}.Build()
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if !strings.Contains(query, "ORDER BY lower(hardware_serial) DESC") {
		t.Fatalf("query = %s", query)
	}
}

func TestListQueryRejectsMultiColumnSort(t *testing.T) {
	_, _, err := ListQuery{
		SelectSQL: "SELECT * FROM hosts",
		OrderKeys: map[string]OrderExpr{
			"display_name": {SQL: "lower(display_name)"},
			"last_seen_at": {SQL: "last_seen_at", NullOrder: NullsLast},
		},
		DefaultOrder: []OrderExpr{{SQL: "id"}},
		Params: listing.Normalize(listing.Params{
			Sort: "last_seen_at.desc,display_name.asc",
		}),
	}.Build()
	if !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("err = %v, want fault.ErrInvalidInput", err)
	}
	if !strings.Contains(err.Error(), "multi-column sort") {
		t.Fatalf("err = %v, want multi-column sort", err)
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

func TestWhereBuilderAddfRegistersArgsInOrder(t *testing.T) {
	var where WhereBuilder
	where.Addf("(name ILIKE %s OR description ILIKE %s)", "%munki%", "%munki%")
	where.Addf("resource = %s", "hosts")

	query, args := where.Build()

	wantQuery := "WHERE (name ILIKE $1 OR description ILIKE $2) AND resource = $3"
	if query != wantQuery {
		t.Fatalf("query = %q, want %q", query, wantQuery)
	}
	if len(args) != 3 || args[0] != "%munki%" || args[1] != "%munki%" || args[2] != "hosts" {
		t.Fatalf("args = %#v", args)
	}
}
