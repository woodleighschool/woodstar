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
		`DELETE FROM munki_software_targets WHERE software_id = $1`, []any{softwareID},
		`
INSERT INTO munki_software_targets (software_id, direction, position, label_id, actions, package_selection, pinned_package_id)
VALUES (
	@software_id,
	@direction::target_direction,
	@position,
	@label_id,
	NULLIF(@actions::text[]::munki_manifest_action[], ARRAY[]::munki_manifest_action[]),
	NULLIF(@package_selection, '')::munki_package_selection,
	@pinned_package_id
)`, rows,
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
		if pkg.Software.ID != softwareID {
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
	return string(selector.Strategy)
}

func packageSelectorFromStorage(selection string, packageID *int64) PackageSelector {
	return PackageSelector{Strategy: PackageStrategy(selection), PackageID: packageID}
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
// plus the id of the package it resolves to. Package bodies are assembled by the
// packages store, which owns that projection.
type effectivePackageRow struct {
	PackageID        int64    `db:"package_id"`
	Actions          []string `db:"actions"`
	PackageSelection string   `db:"package_selection"`
	PinnedPackageID  *int64   `db:"pinned_package_id"`
}

// EffectivePackagesForHost resolves Munki package candidates for one host.
func (s *Store) EffectivePackagesForHost(ctx context.Context, hostID int64) ([]EffectivePackage, error) {
	qrows, err := s.db.Pool().Query(ctx, `
SELECT
	p.id AS package_id,
	resolved.actions,
	resolved.package_selection,
	resolved.pinned_package_id
FROM munki_resolved_software_for_host($1) resolved
JOIN munki_packages p ON p.software_id = resolved.software_id
	AND (
		(resolved.package_selection = 'latest' AND resolved.pinned_package_id IS NULL)
		OR (resolved.package_selection = 'specific' AND p.id = resolved.pinned_package_id)
	)
ORDER BY resolved.software_id, p.id`,
		hostID,
	)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[effectivePackageRow])
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.PackageID)
	}
	pkgs, err := s.packages.PackagesByID(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]packages.Package, len(pkgs))
	for _, pkg := range pkgs {
		byID[pkg.ID] = pkg
	}

	effective := make([]EffectivePackage, 0, len(rows))
	for _, row := range rows {
		pkg, ok := byID[row.PackageID]
		if !ok {
			continue
		}
		effective = append(effective, EffectivePackage{
			Actions:  actionsFromStorage(row.Actions),
			Package:  pkg,
			Selector: packageSelectorFromStorage(row.PackageSelection, row.PinnedPackageID),
		})
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
