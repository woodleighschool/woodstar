package dbutil

import (
	"fmt"
	"slices"
	"strings"
)

type OrderExpr struct {
	SQL       string
	NullOrder NullOrder
}

type NullOrder string

const (
	NullOrderDefault NullOrder = ""
	NullsFirst       NullOrder = "NULLS FIRST"
	NullsLast        NullOrder = "NULLS LAST"
)

type ListQuery struct {
	SelectSQL    string
	WhereSQL     string
	GroupBySQL   string
	Args         []any
	OrderKeys    map[string]OrderExpr
	DefaultOrder []OrderExpr
	Params       ListParams
}

type WhereBuilder struct {
	clauses []string
	args    []any
}

func (b *WhereBuilder) Add(clause string) {
	if clause == "" {
		return
	}
	b.clauses = append(b.clauses, clause)
}

func (b *WhereBuilder) Arg(arg any) string {
	b.args = append(b.args, arg)
	return fmt.Sprintf("$%d", len(b.args))
}

func (b WhereBuilder) Build() (string, []any) {
	if len(b.clauses) == 0 {
		return "", slices.Clone(b.args)
	}
	return "WHERE " + strings.Join(b.clauses, " AND "), slices.Clone(b.args)
}

func (q ListQuery) Build() (string, []any, error) {
	params := CleanListParams(q.Params)
	orderSQL, err := OrderBy(params, q.OrderKeys, q.DefaultOrder)
	if err != nil {
		return "", nil, err
	}
	args := slices.Clone(q.Args)
	limitIndex := len(args) + 1
	args = append(args, int32(params.PageSize), int32(params.PageIndex*params.PageSize))

	parts := []string{q.SelectSQL}
	if q.WhereSQL != "" {
		parts = append(parts, q.WhereSQL)
	}
	if q.GroupBySQL != "" {
		parts = append(parts, q.GroupBySQL)
	}
	if orderSQL != "" {
		parts = append(parts, orderSQL)
	}
	parts = append(parts, fmt.Sprintf("LIMIT $%d OFFSET $%d", limitIndex, limitIndex+1))
	return strings.Join(parts, "\n"), args, nil
}

func OrderBy(params ListParams, orderKeys map[string]OrderExpr, defaultOrder []OrderExpr) (string, error) {
	order := make([]OrderExpr, 0, 1+len(defaultOrder))
	sortKey, sortDirection, err := parseSort(params.Sort)
	if err != nil {
		return "", err
	}
	if sortKey != "" {
		expr, ok := orderKeys[sortKey]
		if !ok {
			return "", fmt.Errorf("%w: unknown sort key %q", ErrInvalidInput, sortKey)
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
	if sortDirection == orderDesc {
		direction = orderSQLDesc
	}
	parts := make([]string, 0, len(order))
	for i, expr := range order {
		itemDirection := direction
		if i > 0 {
			itemDirection = orderSQLAsc
		}
		part := expr.SQL + " " + itemDirection
		if expr.NullOrder != NullOrderDefault {
			part += " " + string(expr.NullOrder)
		}
		parts = append(parts, part)
	}
	return "ORDER BY " + strings.Join(parts, ", "), nil
}

func parseSort(sort string) (string, string, error) {
	if sort == "" {
		return "", orderAsc, nil
	}
	dot := strings.LastIndex(sort, ".")
	if dot == -1 {
		return sort, orderAsc, nil
	}
	key, direction := sort[:dot], sort[dot+1:]
	if key == "" {
		return "", "", fmt.Errorf("%w: sort key is required", ErrInvalidInput)
	}
	if direction != orderAsc && direction != orderDesc {
		return sort, orderAsc, nil
	}
	return key, direction, nil
}

func orderContains(order []OrderExpr, sql string) bool {
	return slices.ContainsFunc(order, func(expr OrderExpr) bool {
		return expr.SQL == sql
	})
}
