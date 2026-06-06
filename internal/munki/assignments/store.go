package assignments

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/softwaretitles"
)

type softwareTitleStore interface {
	GetByID(context.Context, int64) (*softwaretitles.SoftwareTitle, error)
}

type packageStore interface {
	GetByID(context.Context, int64) (*packages.Package, error)
	AttachRelations(context.Context, []packages.Package) ([]packages.Package, error)
}

type Store struct {
	db             *database.DB
	q              *sqlc.Queries
	softwareTitles softwareTitleStore
	packages       packageStore
}

func NewStore(db *database.DB, softwareTitles softwareTitleStore, packages packageStore) *Store {
	return &Store{
		db:             db,
		q:              db.Queries(),
		softwareTitles: softwareTitles,
		packages:       packages,
	}
}

func (s *Store) Create(ctx context.Context, params AssignmentMutation) (*Assignment, error) {
	var err error
	params, err = s.normalizeMutation(ctx, params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.CreateMunkiAssignment(ctx, sqlc.CreateMunkiAssignmentParams{
		SoftwareID:       params.SoftwareID,
		Priority:         params.Priority,
		LabelID:          params.LabelID,
		Action:           sqlc.MunkiAssignmentAction(params.Action),
		OptionalInstall:  params.OptionalInstall,
		FeaturedItem:     params.FeaturedItem,
		PackageSelection: sqlc.MunkiPackageSelection(params.PackageSelection),
		PinnedPackageID:  params.PinnedPackageID,
	})
	if err != nil {
		return nil, mapMutationError(err)
	}
	return s.GetByID(ctx, row.ID)
}

func (s *Store) Update(ctx context.Context, id int64, params AssignmentMutation) (*Assignment, error) {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if params.SoftwareID != 0 && params.SoftwareID != existing.SoftwareID {
		return nil, fmt.Errorf("%w: software_id cannot be changed", dbutil.ErrInvalidInput)
	}
	params.SoftwareID = existing.SoftwareID
	params, err = s.normalizeMutation(ctx, params)
	if err != nil {
		return nil, err
	}
	row, err := s.q.UpdateMunkiAssignment(ctx, sqlc.UpdateMunkiAssignmentParams{
		ID:               id,
		Priority:         params.Priority,
		LabelID:          params.LabelID,
		Action:           sqlc.MunkiAssignmentAction(params.Action),
		OptionalInstall:  params.OptionalInstall,
		FeaturedItem:     params.FeaturedItem,
		PackageSelection: sqlc.MunkiPackageSelection(params.PackageSelection),
		PinnedPackageID:  params.PinnedPackageID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, mapMutationError(err)
	}
	return s.GetByID(ctx, row.ID)
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Assignment, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.db.Pool().Query(ctx, assignmentSelectSQL+"\nWHERE a.id = $1", id)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(row, pgx.RowToStructByName[assignmentRecord])
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, dbutil.ErrNotFound
	}
	assignment := assignmentFromRecord(records[0])
	return &assignment, nil
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

func (s *Store) Reorder(ctx context.Context, softwareID int64, orderedIDs []int64) error {
	if softwareID <= 0 {
		return fmt.Errorf("%w: software_id is required", dbutil.ErrInvalidInput)
	}
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		currentIDs, err := q.ListMunkiAssignmentIDsBySoftware(
			ctx,
			sqlc.ListMunkiAssignmentIDsBySoftwareParams{SoftwareID: softwareID},
		)
		if err != nil {
			return err
		}
		if !dbutil.SameInt64Set(orderedIDs, currentIDs) {
			return fmt.Errorf("%w: ordered_ids must exactly match existing assignment IDs", dbutil.ErrInvalidInput)
		}
		if err := q.SetMunkiAssignmentPriorities(ctx, sqlc.SetMunkiAssignmentPrioritiesParams{
			SoftwareID: softwareID,
			OrderedIds: orderedIDs,
		}); err != nil {
			return err
		}
		return nil
	})
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

func (s *Store) ReplaceExcludeLabelIDs(ctx context.Context, softwareID int64, labelIDs []int64) ([]int64, error) {
	if softwareID <= 0 {
		return nil, fmt.Errorf("%w: software_id is required", dbutil.ErrInvalidInput)
	}
	if err := validateExcludeLabelIDs(labelIDs); err != nil {
		return nil, err
	}
	if _, err := s.softwareTitles.GetByID(ctx, softwareID); err != nil {
		return nil, err
	}
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := validateExcludeLabelsForSoftware(ctx, q, softwareID, labelIDs); err != nil {
			return err
		}
		if err := q.DeleteMunkiAssignmentExcludeLabels(
			ctx,
			sqlc.DeleteMunkiAssignmentExcludeLabelsParams{SoftwareID: softwareID},
		); err != nil {
			return err
		}
		if len(labelIDs) == 0 {
			return nil
		}
		return q.InsertMunkiAssignmentExcludeLabels(
			ctx,
			sqlc.InsertMunkiAssignmentExcludeLabelsParams{SoftwareID: softwareID, LabelIds: labelIDs},
		)
	})
	if err != nil {
		return nil, mapMutationError(err)
	}
	return s.ExcludeLabelIDs(ctx, softwareID)
}

func validateExcludeLabelIDs(labelIDs []int64) error {
	seen := make(map[int64]struct{}, len(labelIDs))
	for _, labelID := range labelIDs {
		if labelID <= 0 {
			return fmt.Errorf("%w: exclude_label_ids contains an invalid label_id", dbutil.ErrInvalidInput)
		}
		if _, ok := seen[labelID]; ok {
			return fmt.Errorf("%w: duplicate exclude_label_ids entry", dbutil.ErrInvalidInput)
		}
		seen[labelID] = struct{}{}
	}
	return nil
}

func validateExcludeLabelsForSoftware(
	ctx context.Context,
	q *sqlc.Queries,
	softwareID int64,
	labelIDs []int64,
) error {
	if len(labelIDs) == 0 {
		return nil
	}
	includeLabelIDs, err := q.ListMunkiAssignmentLabelIDsBySoftware(
		ctx,
		sqlc.ListMunkiAssignmentLabelIDsBySoftwareParams{SoftwareID: softwareID},
	)
	if err != nil {
		return err
	}
	includes := make(map[int64]struct{}, len(includeLabelIDs))
	for _, labelID := range includeLabelIDs {
		includes[labelID] = struct{}{}
	}
	for _, labelID := range labelIDs {
		if _, ok := includes[labelID]; ok {
			return fmt.Errorf("%w: label_id is already assigned to this software", dbutil.ErrInvalidInput)
		}
	}
	builtinExcludeIDs, err := q.ListBuiltinLabelIDs(ctx, sqlc.ListBuiltinLabelIDsParams{LabelIds: labelIDs})
	if err != nil {
		return err
	}
	if len(builtinExcludeIDs) > 0 {
		return fmt.Errorf("%w: builtin labels cannot be excluded from Munki assignments", dbutil.ErrInvalidInput)
	}
	return nil
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

func cleanMutation(params AssignmentMutation) AssignmentMutation {
	params.Action = AssignmentAction(strings.TrimSpace(string(params.Action)))
	params.PackageSelection = PackageSelection(strings.TrimSpace(string(params.PackageSelection)))
	return params
}

func (s *Store) normalizeMutation(
	ctx context.Context,
	params AssignmentMutation,
) (AssignmentMutation, error) {
	params = cleanMutation(params)
	if err := params.Validate(); err != nil {
		return params, err
	}
	if _, err := s.softwareTitles.GetByID(ctx, params.SoftwareID); err != nil {
		return params, err
	}
	if err := s.validateAssignmentLabelNotExcluded(ctx, params.SoftwareID, params.LabelID); err != nil {
		return params, err
	}
	if params.PackageSelection != PackageSelectionSpecific {
		return params, nil
	}
	pkg, err := s.packages.GetByID(ctx, *params.PinnedPackageID)
	if err != nil {
		return params, err
	}
	if pkg.SoftwareID != params.SoftwareID {
		return params, fmt.Errorf(
			"%w: pinned_package_id must belong to software_id",
			dbutil.ErrInvalidInput,
		)
	}
	return params, nil
}

func (s *Store) validateAssignmentLabelNotExcluded(ctx context.Context, softwareID int64, labelID int64) error {
	excludeLabelIDs, err := s.ExcludeLabelIDs(ctx, softwareID)
	if err != nil {
		return err
	}
	if slices.Contains(excludeLabelIDs, labelID) {
		return fmt.Errorf("%w: label_id is already excluded from this software", dbutil.ErrInvalidInput)
	}
	return nil
}

func assignmentFromRecord(row assignmentRecord) Assignment {
	return Assignment{
		ID:                   row.ID,
		SoftwareID:           row.SoftwareID,
		SoftwareDisplayName:  row.SoftwareDisplayName,
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
			p.name ILIKE ` + search + `
			OR p.version ILIKE ` + search + `
			OR p.display_name ILIKE ` + search + `
			OR s.name ILIKE ` + search + `
			OR s.display_name ILIKE ` + search + `
			OR a.action::text ILIKE ` + search + `
			OR a.package_selection::text ILIKE ` + search + `
		)`)
	}
	return where.Build()
}

func assignmentOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"priority":   {SQL: "a.priority"},
		"name":       {SQL: "lower(COALESCE(NULLIF(s.display_name, ''), s.name))"},
		"action":     {SQL: "a.action"},
		"optional":   {SQL: "a.optional_install"},
		"featured":   {SQL: "a.featured_item"},
		"updated_at": {SQL: "a.updated_at"},
	}
}

type assignmentRecord struct {
	ID                   int64
	SoftwareID           int64
	SoftwareDisplayName  string
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
	COALESCE(NULLIF(s.display_name, ''), s.name) AS software_display_name,
	a.priority,
	a.label_id,
	a.action,
	a.optional_install,
	a.featured_item,
	a.package_selection,
	a.pinned_package_id,
	p.name AS pinned_package_name,
	p.version AS pinned_package_version,
	a.created_at,
	a.updated_at
FROM munki_assignments a
JOIN munki_software_titles s ON s.id = a.software_id
LEFT JOIN munki_packages p ON p.id = a.pinned_package_id`

func mapMutationError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	switch database.SQLState(err) {
	case pgerrcode.ForeignKeyViolation:
		return dbutil.ErrNotFound
	case pgerrcode.UniqueViolation:
		return dbutil.ErrAlreadyExists
	case pgerrcode.InvalidTextRepresentation,
		pgerrcode.NotNullViolation,
		pgerrcode.CheckViolation:
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return err
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
