package mdp

import (
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/postgres"
)

// DistributionPointListParams is the list contract for distribution points.
type DistributionPointListParams struct {
	ListParams listing.Params
}

func distributionPointListWhere(params DistributionPointListParams) (string, []any) {
	var where postgres.WhereBuilder
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
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
) postgres.ListQuery {
	return postgres.ListQuery{
		SelectSQL:    distributionPointSelectSQL(),
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    distributionPointOrderKeys(),
		DefaultOrder: []postgres.OrderExpr{{SQL: "c.position"}, {SQL: "c.id"}},
		Params:       params.ListParams,
	}
}

func distributionPointOrderKeys() map[string]postgres.OrderExpr {
	return map[string]postgres.OrderExpr{
		"name":     {SQL: "lower(c.name)"},
		"position": {SQL: "c.position"},
	}
}

func distributionPointSelectSQL() string {
	return `
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
}
