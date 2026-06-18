package software

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

func (s *Store) replaceTargets(
	ctx context.Context,
	qtx *sqlc.Queries,
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
	if err := validateExcludedLabels(ctx, qtx, targets.Exclude); err != nil {
		return err
	}
	if err := qtx.DeleteMunkiSoftwareTargetsBySoftware(
		ctx,
		sqlc.DeleteMunkiSoftwareTargetsBySoftwareParams{SoftwareID: softwareID},
	); err != nil {
		return err
	}
	for index, include := range targets.Include {
		if err := qtx.CreateMunkiSoftwareInclude(
			ctx,
			createIncludeParams(softwareID, int32(index+1), include),
		); err != nil {
			return err
		}
	}
	if len(targets.Exclude) == 0 {
		return nil
	}
	return qtx.InsertMunkiSoftwareExcludeLabels(
		ctx,
		sqlc.InsertMunkiSoftwareExcludeLabelsParams{SoftwareID: softwareID, LabelIds: labelRefIDs(targets.Exclude)},
	)
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

func createIncludeParams(
	softwareID int64,
	priority int32,
	include Include,
) sqlc.CreateMunkiSoftwareIncludeParams {
	params := sqlc.CreateMunkiSoftwareIncludeParams{
		SoftwareID:       softwareID,
		Priority:         priority,
		LabelID:          include.LabelID,
		Actions:          storageActions(include.Actions),
		PackageSelection: storagePackageSelection(include.Package),
		PinnedPackageID:  include.Package.PackageID,
	}
	return params
}

func validateExcludedLabels(ctx context.Context, qtx *sqlc.Queries, excludes []targeting.LabelRef) error {
	if len(excludes) == 0 {
		return nil
	}
	builtinExcludeIDs, err := qtx.ListBuiltinLabelIDs(ctx, sqlc.ListBuiltinLabelIDsParams{
		LabelIds: labelRefIDs(excludes),
	})
	if err != nil {
		return err
	}
	if len(builtinExcludeIDs) > 0 {
		return fmt.Errorf("%w: builtin labels cannot be excluded from Munki software", dbutil.ErrInvalidInput)
	}
	return nil
}

func storagePackageSelection(selector PackageSelector) sqlc.MunkiPackageSelection {
	switch selector.Strategy {
	case PackageSpecific:
		return sqlc.MunkiPackageSelectionSpecificPackage
	default:
		return sqlc.MunkiPackageSelectionLatestEligible
	}
}

func packageSelectorFromStorage(selection sqlc.MunkiPackageSelection, packageID *int64) PackageSelector {
	switch selection {
	case sqlc.MunkiPackageSelectionSpecificPackage:
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
	rows, err := s.db.Pool().Query(
		ctx,
		softwareIncludeSelectSQL+"\nWHERE a.direction = 'include' AND a.software_id = $1\nORDER BY a.position, a.label_id",
		softwareID,
	)
	if err != nil {
		return Targets{}, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[softwareIncludeRecord])
	if err != nil {
		return Targets{}, err
	}
	targets := emptyTargets()
	for _, record := range records {
		targets.Include = append(targets.Include, includeFromRecord(record))
	}
	excludes, err := s.q.ListMunkiSoftwareExcludeLabels(
		ctx,
		sqlc.ListMunkiSoftwareExcludeLabelsParams{SoftwareIds: []int64{softwareID}},
	)
	if err != nil {
		return Targets{}, err
	}
	for _, row := range excludes {
		targets.Exclude = append(targets.Exclude, targeting.LabelRef{LabelID: row.LabelID})
	}
	return targets, nil
}

// EffectivePackagesForHost resolves Munki package candidates for one host.
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

func includeFromRecord(row softwareIncludeRecord) Include {
	return Include{
		LabelID: row.LabelID,
		Package: packageSelectorFromStorage(
			row.PackageSelection,
			row.PinnedPackageID,
		),
		Actions: actionsFromStorage(row.Actions),
	}
}

type softwareIncludeRecord struct {
	LabelID          int64
	Actions          []string
	PackageSelection sqlc.MunkiPackageSelection
	PinnedPackageID  *int64
}

const softwareIncludeSelectSQL = `
SELECT
	a.label_id,
	a.actions::text[] AS actions,
	a.package_selection,
	a.pinned_package_id
FROM munki_software_targets a`
