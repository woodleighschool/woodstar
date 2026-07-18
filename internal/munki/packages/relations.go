package packages

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const (
	relationKindRequires  = "requires"
	relationKindUpdateFor = "update_for"
)

func writePackageRelations(ctx context.Context, tx pgx.Tx, packageID int64, params PackageMutation) error {
	if err := replacePackageRelations(ctx, tx, packageID, relationKindRequires, params.Requires); err != nil {
		return dbutil.MutationError(err)
	}
	if err := replacePackageRelations(ctx, tx, packageID, relationKindUpdateFor, params.UpdateFor); err != nil {
		return dbutil.MutationError(err)
	}
	return nil
}

type packageRelationWrite struct {
	PackageID        int64  `db:"package_id"`
	RelationKind     string `db:"relation_kind"`
	TargetSoftwareID int64  `db:"target_software_id"`
	TargetPackageID  *int64 `db:"target_package_id"`
	Position         int32  `db:"position"`
}

func replacePackageRelations(
	ctx context.Context,
	tx pgx.Tx,
	packageID int64,
	kind string,
	references []PackageReferenceMutation,
) error {
	rows := make([]packageRelationWrite, len(references))
	for i, ref := range references {
		rows[i] = packageRelationWrite{
			PackageID:        packageID,
			RelationKind:     kind,
			TargetSoftwareID: ref.SoftwareID,
			TargetPackageID:  optionalPositiveInt64(ref.PackageID),
			Position:         int32(i),
		}
	}
	return dbutil.ReplaceChildren(
		ctx, tx,
		`
DELETE FROM munki_package_relations
WHERE package_id = $1 AND relation_kind = $2::munki_package_relation_kind`,
		[]any{packageID, kind},
		`
INSERT INTO munki_package_relations (package_id, relation_kind, target_software_id, target_package_id, position)
VALUES (@package_id, @relation_kind::munki_package_relation_kind, @target_software_id, @target_package_id, @position)`,
		rows,
	)
}

type packageRelationRow struct {
	PackageID       int64
	RelationKind    string
	SoftwareID      int64
	SoftwareName    string
	TargetPackageID *int64
	TargetVersion   string
}

type packageRelations struct {
	requires  []PackageReference
	updateFor []PackageReference
}

// attachRelations loads requires and update_for references for package rows.
func (s *Store) attachRelations(ctx context.Context, packages []Package) ([]Package, error) {
	relations, err := s.packageRelationsByPackage(ctx, packageIDs(packages))
	if err != nil {
		return nil, err
	}
	for i := range packages {
		rel := relations[packages[i].ID]
		packages[i].Requires = nonNilReferences(rel.requires)
		packages[i].UpdateFor = nonNilReferences(rel.updateFor)
	}
	return packages, nil
}

func (s *Store) packageRelationsByPackage(
	ctx context.Context,
	packageIDs []int64,
) (map[int64]packageRelations, error) {
	if len(packageIDs) == 0 {
		return map[int64]packageRelations{}, nil
	}
	rows, err := s.db.Pool().Query(ctx, `
		SELECT
			r.package_id,
			r.relation_kind,
			r.target_software_id AS software_id,
			target_software.name AS software_name,
			r.target_package_id,
			COALESCE(target.version, '') AS target_version
		FROM munki_package_relations r
		JOIN munki_software target_software ON target_software.id = r.target_software_id
		LEFT JOIN munki_packages target ON target.id = r.target_package_id
		WHERE r.package_id = ANY($1::bigint[])
		ORDER BY r.package_id, r.relation_kind, r.position, r.id
	`, packageIDs)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[packageRelationRow])
	if err != nil {
		return nil, err
	}
	out := make(map[int64]packageRelations, len(packageIDs))
	for _, record := range records {
		reference := PackageReference{
			SoftwareID:     record.SoftwareID,
			SoftwareName:   record.SoftwareName,
			PackageVersion: record.TargetVersion,
		}
		if record.TargetPackageID != nil {
			reference.PackageID = *record.TargetPackageID
		}
		rel := out[record.PackageID]
		switch record.RelationKind {
		case relationKindRequires:
			rel.requires = append(rel.requires, reference)
		case relationKindUpdateFor:
			rel.updateFor = append(rel.updateFor, reference)
		}
		out[record.PackageID] = rel
	}
	return out, nil
}

func packageIDs(packages []Package) []int64 {
	ids := make([]int64, 0, len(packages))
	for _, pkg := range packages {
		if pkg.ID <= 0 {
			continue
		}
		ids = append(ids, pkg.ID)
	}
	return ids
}

func optionalPositiveInt64(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return &value
}

func nonNilReferences(values []PackageReference) []PackageReference {
	if values == nil {
		return []PackageReference{}
	}
	return values
}
