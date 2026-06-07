package assignments

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

type packageStore interface {
	AttachRelations(context.Context, []packages.Package) ([]packages.Package, error)
}

type Store struct {
	db       *database.DB
	q        *sqlc.Queries
	packages packageStore
}

func NewStore(db *database.DB, packages packageStore) *Store {
	return &Store{
		db:       db,
		q:        db.Queries(),
		packages: packages,
	}
}

func (s *Store) List(ctx context.Context, params AssignmentListParams) ([]Assignment, int, error) {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	where, args := assignmentListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL:    assignmentSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    assignmentOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{{SQL: "a.priority"}, {SQL: "a.id"}},
		Params:       params.ListParams,
	}
	var count int
	countSQL, countArgs := listQuery.BuildCount()
	if err := s.db.Pool().QueryRow(ctx, countSQL, countArgs...).Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := listQuery.Build()
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[assignmentRecord])
	if err != nil {
		return nil, 0, err
	}
	assignments := make([]Assignment, len(records))
	for i, row := range records {
		assignments[i] = assignmentFromRecord(row)
	}
	return assignments, count, nil
}

func (s *Store) ListForSoftwareTitle(ctx context.Context, softwareID int64) ([]Assignment, error) {
	if softwareID <= 0 {
		return nil, dbutil.ErrNotFound
	}
	rows, err := s.db.Pool().Query(
		ctx,
		assignmentSelectSQL+"\nWHERE a.software_id = $1\nORDER BY a.priority, a.id",
		softwareID,
	)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[assignmentRecord])
	if err != nil {
		return nil, err
	}
	assignments := make([]Assignment, len(records))
	for i, row := range records {
		assignments[i] = assignmentFromRecord(row)
	}
	return assignments, nil
}

func (s *Store) EffectivePackagesForHost(ctx context.Context, hostID int64) ([]EffectivePackage, error) {
	rows, err := s.q.ListEffectiveMunkiPackagesForHost(ctx, sqlc.ListEffectiveMunkiPackagesForHostParams{
		HostID: hostID,
	})
	if err != nil {
		return nil, err
	}
	effective := make([]EffectivePackage, 0, len(rows))
	for _, row := range rows {
		pkg, err := packages.FromEffectiveRow(row)
		if err != nil {
			return nil, err
		}
		effective = append(effective, EffectivePackage{
			AssignmentID:     row.AssignmentID,
			SoftwareID:       row.AssignmentSoftwareID,
			Action:           AssignmentAction(row.Action),
			OptionalInstall:  row.OptionalInstall,
			FeaturedItem:     row.FeaturedItem,
			PackageSelection: PackageSelection(row.PackageSelection),
			PinnedPackageID:  row.PinnedPackageID,
			Priority:         row.Priority,
			Package:          pkg,
		})
	}
	resolved := ResolveEffectivePackages(effective)
	return s.attachPackageRelations(ctx, resolved)
}

func (s *Store) ExcludeLabelIDs(ctx context.Context, softwareID int64) ([]int64, error) {
	if softwareID <= 0 {
		return nil, dbutil.ErrNotFound
	}
	rows, err := s.q.ListMunkiAssignmentExcludeLabels(
		ctx,
		sqlc.ListMunkiAssignmentExcludeLabelsParams{SoftwareIds: []int64{softwareID}},
	)
	if err != nil {
		return nil, err
	}
	out := make([]int64, len(rows))
	for i, row := range rows {
		out[i] = row.LabelID
	}
	return out, nil
}

func (s *Store) attachPackageRelations(
	ctx context.Context,
	effective []EffectivePackage,
) ([]EffectivePackage, error) {
	pkgs := make([]packages.Package, 0, len(effective))
	for _, pkg := range effective {
		if pkg.Package.ID > 0 {
			pkgs = append(pkgs, pkg.Package)
		}
	}
	pkgs, err := s.packages.AttachRelations(ctx, pkgs)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]packages.Package, len(pkgs))
	for _, pkg := range pkgs {
		byID[pkg.ID] = pkg
	}
	for i := range effective {
		if pkg, ok := byID[effective[i].Package.ID]; ok {
			effective[i].Package = pkg
		}
	}
	return effective, nil
}

func assignmentFromRecord(row assignmentRecord) Assignment {
	return Assignment{
		ID:                   row.ID,
		SoftwareID:           row.SoftwareID,
		SoftwareName:         row.SoftwareName,
		Priority:             row.Priority,
		LabelID:              row.LabelID,
		Action:               AssignmentAction(row.Action),
		OptionalInstall:      row.OptionalInstall,
		FeaturedItem:         row.FeaturedItem,
		PackageSelection:     PackageSelection(row.PackageSelection),
		PinnedPackageID:      row.PinnedPackageID,
		PinnedPackageName:    stringPtrValue(row.PinnedPackageName),
		PinnedPackageVersion: stringPtrValue(row.PinnedPackageVersion),
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func assignmentListWhere(params AssignmentListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.SoftwareID > 0 {
		where.Add("a.software_id = " + where.Arg(params.SoftwareID))
	}
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			p.version ILIKE ` + search + `
			OR s.name ILIKE ` + search + `
			OR a.action::text ILIKE ` + search + `
			OR a.package_selection::text ILIKE ` + search + `
		)`)
	}
	return where.Build()
}

func assignmentOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"priority":   {SQL: "a.priority"},
		"name":       {SQL: "lower(s.name)"},
		"action":     {SQL: "a.action"},
		"optional":   {SQL: "a.optional_install"},
		"featured":   {SQL: "a.featured_item"},
		"updated_at": {SQL: "a.updated_at"},
	}
}

type assignmentRecord struct {
	ID                   int64
	SoftwareID           int64
	SoftwareName         string
	Priority             int32
	LabelID              int64
	Action               sqlc.MunkiAssignmentAction
	OptionalInstall      bool
	FeaturedItem         bool
	PackageSelection     sqlc.MunkiPackageSelection
	PinnedPackageID      *int64
	PinnedPackageName    *string
	PinnedPackageVersion *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

const assignmentSelectSQL = `
SELECT
	a.id,
	a.software_id,
	s.name AS software_name,
	a.priority,
	a.label_id,
	a.action,
	a.optional_install,
	a.featured_item,
	a.package_selection,
	a.pinned_package_id,
	pinned_software.name AS pinned_package_name,
	p.version AS pinned_package_version,
	a.created_at,
	a.updated_at
FROM munki_assignments a
JOIN munki_software_titles s ON s.id = a.software_id
LEFT JOIN munki_packages p ON p.id = a.pinned_package_id
LEFT JOIN munki_software_titles pinned_software ON pinned_software.id = p.software_id`

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
