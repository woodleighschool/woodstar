package dbutil

import (
	"fmt"
	"slices"
	"strings"
)

type OrderExpr struct {
	SQL       string
	NullsLast bool
}

type ListQuery struct {
	SelectSQL    string
	WhereSQL     string
	GroupBySQL   string
	Args         []any
	OrderKeys    map[string]OrderExpr
	DefaultOrder []OrderExpr
	Params       ListParams
}

func (q ListQuery) Build() (string, []any, error) {
	params := CleanListParams(q.Params)
	orderSQL, err := OrderBy(params, q.OrderKeys, q.DefaultOrder)
	if err != nil {
		return "", nil, err
	}
	args := append([]any{}, q.Args...)
	limitIndex := len(args) + 1
	args = append(args, int32(params.PerPage), int32((params.Page-1)*params.PerPage))

	parts := []string{strings.TrimSpace(q.SelectSQL)}
	if q.WhereSQL != "" {
		parts = append(parts, strings.TrimSpace(q.WhereSQL))
	}
	if q.GroupBySQL != "" {
		parts = append(parts, strings.TrimSpace(q.GroupBySQL))
	}
	if orderSQL != "" {
		parts = append(parts, orderSQL)
	}
	parts = append(parts, fmt.Sprintf("LIMIT $%d OFFSET $%d", limitIndex, limitIndex+1))
	return strings.Join(parts, "\n"), args, nil
}

func OrderBy(params ListParams, orderKeys map[string]OrderExpr, defaultOrder []OrderExpr) (string, error) {
	order := make([]OrderExpr, 0, 1+len(defaultOrder))
	if params.OrderKey != "" {
		expr, ok := orderKeys[params.OrderKey]
		if !ok {
			return "", fmt.Errorf("%w: unknown order key %q", ErrInvalidInput, params.OrderKey)
		}
		order = append(order, expr)
	}
	for _, expr := range defaultOrder {
		if !orderContains(order, expr.SQL) {
			order = append(order, expr)
		}
	}
	if len(order) == 0 {
		return "", nil
	}

	direction := orderSQLAsc
	if params.OrderDirection == orderDesc {
		direction = orderSQLDesc
	}
	parts := make([]string, 0, len(order))
	for i, expr := range order {
		itemDirection := direction
		if i > 0 {
			itemDirection = orderSQLAsc
		}
		part := expr.SQL + " " + itemDirection
		if expr.NullsLast {
			part += " NULLS LAST"
		}
		parts = append(parts, part)
	}
	return "ORDER BY " + strings.Join(parts, ", "), nil
}

func orderContains(order []OrderExpr, sql string) bool {
	return slices.ContainsFunc(order, func(expr OrderExpr) bool {
		return expr.SQL == sql
	})
}
