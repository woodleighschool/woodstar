package mdp

import "github.com/woodleighschool/woodstar/internal/dbutil"

// DistributionPointListParams is the list contract for distribution points.
type DistributionPointListParams struct {
	dbutil.ListParams
}

func distributionPointListWhere(params DistributionPointListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			c.name ILIKE ` + search + `
			OR c.client_base_url ILIKE ` + search + `
		)`)
	}
	return where.Build()
}

func distributionPointListQuery(
	params DistributionPointListParams,
	where string,
	args []any,
) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL:    distributionPointSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    distributionPointOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{{SQL: "c.position"}, {SQL: "c.id"}},
		Params:       params.ListParams,
	}
}

func distributionPointOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"name":     {SQL: "lower(c.name)"},
		"position": {SQL: "c.position"},
	}
}

// distributionPointSelectSQL lists every column so the row maps onto the
// generated struct; the key is read but never serialized by the admin model.
const distributionPointSelectSQL = `
SELECT
	c.id,
	c.name,
	c.enabled,
	c.position,
	c.client_cidrs,
	c.client_base_url,
	c."key",
	c.created_at,
	c.updated_at
FROM munki_distribution_points c`
