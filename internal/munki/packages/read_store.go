package packages

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/postgres"
)

func (s *Store) GetByID(ctx context.Context, id int64) (*Package, error) {
	if id <= 0 {
		return nil, fault.ErrNotFound
	}
	row, err := postgres.GetOne[packageRow](ctx, s.pool, packageSelectSQL()+"\nWHERE p.id = $1", id)
	if err != nil {
		return nil, err
	}
	packages, err := s.attachRelations(ctx, []Package{packageFromRow(row)})
	if err != nil {
		return nil, err
	}
	return &packages[0], nil
}

func (s *Store) List(ctx context.Context, params PackageListParams) ([]Package, int, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, 0, err
	}
	where, args := packageListWhere(params)
	listQuery := postgres.ListQuery{
		SelectSQL: packageSelectSQL(),
		WhereSQL:  where,
		Args:      args,
		OrderKeys: packageOrderKeys(),
		DefaultOrder: []postgres.OrderExpr{
			{SQL: "lower(s.name)"},
			{SQL: "lower(p.version)"},
			{SQL: "p.id"},
		},
		Params: params.ListParams,
	}
	rows, count, err := postgres.ListWithCount[packageRow](ctx, s.pool, listQuery)
	if err != nil {
		return nil, 0, err
	}
	packages, err := s.attachRelations(ctx, packagesFromRows(rows))
	if err != nil {
		return nil, 0, err
	}
	return packages, count, nil
}

// ListRepositoryPackages returns every package that may appear in the shared
// Munki catalog.
func (s *Store) ListRepositoryPackages(ctx context.Context) ([]Package, error) {
	records, err := postgres.GetAll[packageRow](ctx, s.pool, packageSelectSQL()+`
ORDER BY lower(s.name), s.id, p.id`)
	if err != nil {
		return nil, err
	}
	return s.attachRelations(ctx, packagesFromRows(records))
}

// PackagesByID assembles the given packages with relations attached. The result
// order is unspecified; callers index it by Package.ID.
func (s *Store) PackagesByID(ctx context.Context, ids []int64) ([]Package, error) {
	if len(ids) == 0 {
		return []Package{}, nil
	}
	records, err := postgres.GetAll[packageRow](
		ctx,
		s.pool,
		packageSelectSQL()+"\nWHERE p.id = ANY($1::bigint[])",
		ids,
	)
	if err != nil {
		return nil, err
	}
	return s.attachRelations(ctx, packagesFromRows(records))
}

func packageListWhere(params PackageListParams) (string, []any) {
	var where postgres.WhereBuilder
	if params.SoftwareID > 0 {
		where.Add("p.software_id = " + where.Arg(params.SoftwareID))
	}
	if len(params.InstallerTypes) > 0 {
		where.Add("p.installer_type = ANY(" + where.Arg(params.InstallerTypes) + "::text[])")
	}
	if params.ListParams.Q != "" {
		search := where.Arg("%" + params.ListParams.Q + "%")
		where.Add(`(
			p.version ILIKE ` + search + `
			OR p.installer_type ILIKE ` + search + `
			OR s.name ILIKE ` + search + `
			OR s.display_name ILIKE ` + search + `
			OR s.description ILIKE ` + search + `
			OR s.category ILIKE ` + search + `
			OR s.developer ILIKE ` + search + `
		)`)
	}
	return where.Build()
}

func packageOrderKeys() map[string]postgres.OrderExpr {
	return map[string]postgres.OrderExpr{
		"software_name":         {SQL: "lower(s.name)"},
		"software_display_name": {SQL: "lower(s.display_name)"},
		"version":               {SQL: "lower(p.version)"},
		"type":                  {SQL: "lower(p.installer_type)"},
		"size":                  {SQL: "COALESCE(installer_obj.size_bytes, 0)"},
		"updated_at":            {SQL: "p.updated_at"},
	}
}
