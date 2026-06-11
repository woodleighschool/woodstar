package dbutil

import (
	"encoding/json"
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

	parts := q.baseParts()
	if orderSQL != "" {
		parts = append(parts, orderSQL)
	}
	parts = append(parts, fmt.Sprintf("LIMIT $%d OFFSET $%d", limitIndex, limitIndex+1))
	return strings.Join(parts, "\n"), args, nil
}

func (q ListQuery) BuildCount() (string, []any) {
	return "SELECT count(*)::integer FROM (\n" + strings.Join(q.baseParts(), "\n") + "\n) list_count",
		slices.Clone(q.Args)
}

func (q ListQuery) baseParts() []string {
	parts := []string{q.SelectSQL}
	if q.WhereSQL != "" {
		parts = append(parts, q.WhereSQL)
	}
	if q.GroupBySQL != "" {
		parts = append(parts, q.GroupBySQL)
	}
	return parts
}

// OrderBy builds a multi-column ORDER BY from the requested sort columns,
// appending DefaultOrder columns (ascending) that the request did not already
// pin. Sort is the nuqs/TanStack Table wire format: a JSON array of {id, desc}.
func OrderBy(params ListParams, orderKeys map[string]OrderExpr, defaultOrder []OrderExpr) (string, error) {
	cols, err := parseSortColumns(params.Sort)
	if err != nil {
		return "", err
	}

	parts := make([]string, 0, len(cols)+len(defaultOrder))
	used := make([]string, 0, len(cols)+len(defaultOrder))

	for _, col := range cols {
		expr, ok := orderKeys[col.ID]
		if !ok {
			return "", fmt.Errorf("%w: unknown sort key %q", ErrInvalidInput, col.ID)
		}
		if slices.Contains(used, expr.SQL) {
			continue
		}
		used = append(used, expr.SQL)
		parts = append(parts, orderPart(expr, col.Desc))
	}

	for _, expr := range defaultOrder {
		if slices.Contains(used, expr.SQL) {
			continue
		}
		used = append(used, expr.SQL)
		parts = append(parts, orderPart(expr, false))
	}

	if len(parts) == 0 {
		return "", nil
	}
	return "ORDER BY " + strings.Join(parts, ", "), nil
}

type sortColumn struct {
	ID   string `json:"id"`
	Desc bool   `json:"desc"`
}

func parseSortColumns(sort string) ([]sortColumn, error) {
	trimmed := strings.TrimSpace(sort)
	if trimmed == "" {
		return nil, nil
	}
	var cols []sortColumn
	if err := json.Unmarshal([]byte(trimmed), &cols); err != nil {
		return nil, fmt.Errorf("%w: invalid sort %q", ErrInvalidInput, sort)
	}
	for _, col := range cols {
		if col.ID == "" {
			return nil, fmt.Errorf("%w: sort id is required", ErrInvalidInput)
		}
	}
	return cols, nil
}

func orderPart(expr OrderExpr, desc bool) string {
	direction := orderSQLAsc
	if desc {
		direction = orderSQLDesc
	}
	part := expr.SQL + " " + direction
	if expr.NullOrder != NullOrderDefault {
		part += " " + string(expr.NullOrder)
	}
	return part
}
