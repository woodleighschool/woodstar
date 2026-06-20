package software

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

const (
	softwareTargetDirectionInclude = "include"
	softwareTargetDirectionExclude = "exclude"
)

func (s *Store) replaceTargets(
	ctx context.Context,
	tx pgx.Tx,
	softwareID int64,
	targets Targets,
) error {
	targets = normalizeTargets(targets)
	if err := targets.validate(); err != nil {
		return err
	}
	if err := s.validatePackageSelectors(ctx, softwareID, targets.Include); err != nil {
		return err
	}
	if err := validateExcludedLabels(ctx, tx, targets.Exclude); err != nil {
		return err
	}
	rows := make([]softwareTargetWrite, 0, len(targets.Include)+len(targets.Exclude))
	for i, include := range targets.Include {
		rows = append(rows, softwareTargetWrite{
			SoftwareID:       softwareID,
			Direction:        softwareTargetDirectionInclude,
			Position:         int32(i),
			LabelID:          include.LabelID,
			Actions:          storageActions(include.Actions),
			PackageSelection: storagePackageSelection(include.Package),
			PinnedPackageID:  include.Package.PackageID,
		})
	}
	for i, exclude := range targets.Exclude {
		rows = append(rows, softwareTargetWrite{
			SoftwareID: softwareID,
			Direction:  softwareTargetDirectionExclude,
			Position:   int32(i),
			LabelID:    exclude.LabelID,
		})
	}
	if err := dbutil.ReplaceChildren(
		ctx, tx,
		deleteSoftwareTargetsSQL, []any{softwareID},
		insertSoftwareTargetSQL, rows,
	); err != nil {
		return dbutil.MutationError(err)
	}
	return nil
}

func (s *Store) validatePackageSelectors(
	ctx context.Context,
	softwareID int64,
	includes []Include,
) error {
	for _, include := range includes {
		if include.Package.Strategy != PackageSpecific {
			continue
		}
		pkg, err := s.packages.GetByID(ctx, *include.Package.PackageID)
		if err != nil {
			return err
		}
		if pkg.SoftwareID != softwareID {
			return fmt.Errorf("%w: package.package_id must belong to software", dbutil.ErrInvalidInput)
		}
	}
	return nil
}

func validateExcludedLabels(ctx context.Context, q dbutil.Queryer, excludes []targeting.LabelRef) error {
	if len(excludes) == 0 {
		return nil
	}
	ids := targeting.LabelRefIDs(excludes)
	rows, err := q.Query(ctx, `
		SELECT id
		FROM labels
		WHERE id = ANY($1::bigint[]) AND label_type = 'builtin'`, ids)
	if err != nil {
		return err
	}
	builtinIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return err
	}
	if len(builtinIDs) > 0 {
		return fmt.Errorf("%w: builtin labels cannot be excluded from Munki software", dbutil.ErrInvalidInput)
	}
	return nil
}

func storagePackageSelection(selector PackageSelector) string {
	switch selector.Strategy {
	case PackageSpecific:
		return string(PackageSpecific)
	default:
		return string(PackageLatest)
	}
}

func packageSelectorFromStorage(selection string, packageID *int64) PackageSelector {
	switch PackageStrategy(selection) {
	case PackageSpecific:
		return PackageSelector{Strategy: PackageSpecific, PackageID: packageID}
	default:
		return PackageSelector{Strategy: PackageLatest}
	}
}

func storageActions(actions []Action) []string {
	out := make([]string, len(actions))
	for i, action := range actions {
		out[i] = string(action)
	}
	return out
}

func actionsFromStorage(actions []string) []Action {
	out := make([]Action, len(actions))
	for i, action := range actions {
		out[i] = Action(action)
	}
	return out
}

// TargetsForSoftware loads include/exclude target rows for one software.
func (s *Store) TargetsForSoftware(ctx context.Context, softwareID int64) (Targets, error) {
	if softwareID <= 0 {
		return Targets{}, dbutil.ErrNotFound
	}
	type targetRow struct {
		Direction        string   `db:"direction"`
		LabelID          int64    `db:"label_id"`
		Actions          []string `db:"actions"`
		PackageSelection string   `db:"package_selection"`
		PinnedPackageID  *int64   `db:"pinned_package_id"`
	}
	qrows, err := s.db.Pool().Query(
		ctx,
		`SELECT
			direction::text AS direction,
			label_id,
			COALESCE(actions::text[], ARRAY[]::text[]) AS actions,
			COALESCE(package_selection::text, '') AS package_selection,
			pinned_package_id
		FROM munki_software_targets
		WHERE software_id = $1
		ORDER BY direction, position, label_id`,
		softwareID,
	)
	if err != nil {
		return Targets{}, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[targetRow])
	if err != nil {
		return Targets{}, err
	}
	targets := emptyTargets()
	for _, row := range rows {
		switch targeting.Direction(row.Direction) {
		case targeting.Include:
			targets.Include = append(targets.Include, Include{
				LabelID: row.LabelID,
				Package: packageSelectorFromStorage(row.PackageSelection, row.PinnedPackageID),
				Actions: actionsFromStorage(row.Actions),
			})
		case targeting.Exclude:
			targets.Exclude = append(targets.Exclude, targeting.LabelRef{LabelID: row.LabelID})
		}
	}
	return targets, nil
}

// effectivePackageRow scans the host-effective query: the include target shape
// plus the canonical package projection reused from the packages store.
type effectivePackageRow struct {
	packages.PackageRow

	TargetID         int64    `db:"target_id"`
	TargetSoftwareID int64    `db:"target_software_id"`
	Actions          []string `db:"actions"`
	PackageSelection string   `db:"package_selection"`
	PinnedPackageID  *int64   `db:"pinned_package_id"`
}

// EffectivePackagesForHost resolves Munki package candidates for one host.
func (s *Store) EffectivePackagesForHost(ctx context.Context, hostID int64) ([]EffectivePackage, error) {
	qrows, err := s.db.Pool().Query(ctx, effectivePackagesForHostSQL, hostID)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[effectivePackageRow])
	if err != nil {
		return nil, err
	}
	effective := make([]EffectivePackage, 0, len(rows))
	for _, row := range rows {
		pkg := packages.PackageFromRow(row.PackageRow)
		effective = append(effective, EffectivePackage{
			TargetID:             row.TargetID,
			SoftwareID:           row.TargetSoftwareID,
			Actions:              actionsFromStorage(row.Actions),
			Package:              pkg,
			SoftwareIconObjectID: pkg.SoftwareIconObjectID,
			Selector:             packageSelectorFromStorage(row.PackageSelection, row.PinnedPackageID),
		})
	}
	resolved := ResolveEffectivePackages(effective)
	return s.attachPackageRelations(ctx, resolved)
}

const effectivePackagesForHostSQL = `
SELECT
	(a.position + 1)::bigint AS target_id,
	a.software_id AS target_software_id,
	a.actions::text[] AS actions,
	a.package_selection::text AS package_selection,
	a.pinned_package_id,` + packages.PackageColumnsSQL + `
FROM munki_software_targets a
JOIN label_membership lm ON lm.label_id = a.label_id AND lm.host_id = $1
JOIN munki_software s ON s.id = a.software_id
JOIN munki_packages p ON p.software_id = a.software_id
	AND (
		(a.package_selection = 'latest' AND a.pinned_package_id IS NULL)
		OR (a.package_selection = 'specific' AND p.id = a.pinned_package_id)
	)
LEFT JOIN storage_objects installer_obj ON installer_obj.id = p.installer_object_id
WHERE a.direction = 'include'
  AND p.eligible
  AND (p.installer_type = 'nopkg' OR installer_obj.available_at IS NOT NULL)
  AND NOT EXISTS (
      SELECT 1
      FROM munki_software_targets excluded
      JOIN label_membership excluded_lm
        ON excluded_lm.label_id = excluded.label_id
       AND excluded_lm.host_id = $1
      WHERE excluded.software_id = a.software_id
        AND excluded.direction = 'exclude'
  )
ORDER BY a.software_id, a.position, p.id`

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

type softwareTargetWrite struct {
	SoftwareID       int64    `db:"software_id"`
	Direction        string   `db:"direction"`
	Position         int32    `db:"position"`
	LabelID          int64    `db:"label_id"`
	Actions          []string `db:"actions"`
	PackageSelection string   `db:"package_selection"`
	PinnedPackageID  *int64   `db:"pinned_package_id"`
}

const deleteSoftwareTargetsSQL = `DELETE FROM munki_software_targets WHERE software_id = $1`

const insertSoftwareTargetSQL = `
INSERT INTO munki_software_targets (software_id, direction, position, label_id, actions, package_selection, pinned_package_id)
VALUES (
	@software_id,
	@direction::target_direction,
	@position,
	@label_id,
	NULLIF(@actions::text[]::munki_manifest_action[], ARRAY[]::munki_manifest_action[]),
	NULLIF(@package_selection, '')::munki_package_selection,
	@pinned_package_id
)`
