package models

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

	query, args, err := listQuery{
		SelectSQL: "SELECT * FROM hosts",
		WhereSQL:  "WHERE deleted_at IS NULL",
		Args:      []any{"existing"},
		OrderKeys: map[string]orderExpr{
			"display_name": {SQL: "lower(display_name)"},
			"last_seen_at": {SQL: "last_seen_at", NullsLast: true},
		},
		DefaultOrder: []orderExpr{{SQL: "lower(display_name)"}, {SQL: "id"}},
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
	_, _, err := listQuery{
		SelectSQL: "SELECT * FROM hosts",
		OrderKeys: map[string]orderExpr{
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

func TestHostListSQLUsesSharedDynamicListBuilder(t *testing.T) {
	params := cleanHostListParams(HostListParams{
		ListParams: CleanListParams(ListParams{
			Q:              "amy",
			Page:           3,
			PerPage:        10,
			OrderKey:       "hardware_serial",
			OrderDirection: "desc",
		}),
		Platform: "darwin",
		Status:   "online",
		LabelID:  7,
	})

	where, args, err := hostListWhere(params)
	if err != nil {
		t.Fatalf("hostListWhere returned error: %v", err)
	}
	query, args, err := hostListSQLWithWhere(params, where, args)
	if err != nil {
		t.Fatalf("hostListSQLWithWhere returned error: %v", err)
	}
	if strings.Contains(query, "CASE WHEN") {
		t.Fatalf("query still uses sqlc-style CASE sorting: %s", query)
	}
	if !strings.Contains(query, "ORDER BY lower(hardware_serial) DESC") {
		t.Fatalf("query = %s", query)
	}
	if !strings.Contains(query, "LIMIT $4 OFFSET $5") {
		t.Fatalf("query = %s", query)
	}
	if len(args) != 5 {
		t.Fatalf("args len = %d, want 5: %#v", len(args), args)
	}
}

func TestLabelListSQLUsesSharedDynamicListBuilder(t *testing.T) {
	params := cleanLabelListParams(LabelListParams{
		ListParams: CleanListParams(ListParams{
			Q:              "mac",
			Page:           1,
			PerPage:        50,
			OrderKey:       "hosts_count",
			OrderDirection: "desc",
		}),
		LabelType:           LabelTypeRegular,
		LabelMembershipType: LabelMembershipTypeDynamic,
		Platform:            "darwin",
	})

	where, args := labelListWhere(params)
	query, args, err := labelListSQLWithWhere(params, where, args)
	if err != nil {
		t.Fatalf("labelListSQLWithWhere returned error: %v", err)
	}
	if strings.Contains(query, "CASE WHEN") {
		t.Fatalf("query still uses sqlc-style CASE sorting: %s", query)
	}
	if !strings.Contains(query, "ORDER BY hosts_count DESC, lower(l.name) ASC, l.id ASC") {
		t.Fatalf("query = %s", query)
	}
	if !strings.Contains(query, "LIMIT $5 OFFSET $6") {
		t.Fatalf("query = %s", query)
	}
	if len(args) != 6 {
		t.Fatalf("args len = %d, want 6: %#v", len(args), args)
	}
}
