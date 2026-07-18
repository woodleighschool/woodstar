package events

import "github.com/woodleighschool/woodstar/internal/dbutil"

func normalizeListValues[T ~string](items []T) []T {
	raw := make([]string, len(items))
	for i, item := range items {
		raw[i] = string(item)
	}
	values := dbutil.NormalizeListValues(raw)
	out := make([]T, len(values))
	for i, value := range values {
		out[i] = T(value)
	}
	return out
}

func defaultEventOrder(alias string) []dbutil.OrderExpr {
	return []dbutil.OrderExpr{
		{SQL: alias + ".occurred_at", Descending: true},
		{SQL: alias + ".id", Descending: true},
	}
}

func hostEventSelectSQL() string {
	return `
	h.id AS host_id,
	h.display_name,
	h.hostname,
	h.computer_name,
	h.hardware_serial,
	h.hardware_model_identifier,
	COALESCE(sh.machine_id, '') AS santa_machine_id,
	COALESCE(sh.santa_version, '') AS santa_version,
	COALESCE(sh.client_mode_reported::text, '') AS santa_client_mode`
}
