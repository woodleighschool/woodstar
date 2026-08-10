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
	c.updated_at,
	CASE
		WHEN session.distribution_point_id IS NOT NULL THEN jsonb_build_object(
			'compatible', true,
			'protocol_version', session.protocol_version,
			'build_version', session.build_version
		)
		WHEN rejection.distribution_point_id IS NOT NULL THEN jsonb_strip_nulls(jsonb_build_object(
			'compatible', false,
			'protocol_version', rejection.protocol_version,
			'build_version', rejection.build_version
		))
	END AS worker
FROM munki_distribution_points c
LEFT JOIN munki_distribution_worker_sessions session
	ON session.distribution_point_id = c.id
	AND session.expires_at > now()
LEFT JOIN munki_distribution_worker_rejections rejection
	ON rejection.distribution_point_id = c.id
	AND rejection.expires_at > now()`
}
