package munki

import (
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
)

type repositoryProjection struct {
	effectivePackages []munkisoftware.EffectivePackage
	packages          []packages.Package
	packageIDs        map[int64]struct{}
	iconObjectIDs     map[int64]struct{}
}

func buildRepositoryProjection(
	effective []munkisoftware.EffectivePackage,
	repository []packages.Package,
) repositoryProjection {
	packageIDs := make(map[int64]struct{}, len(effective))
	for _, pkg := range effective {
		packageIDs[pkg.Package.ID] = struct{}{}
	}

	for changed := true; changed; {
		changed = false
		includedSoftwareIDs := includedSoftwareIDs(repository, packageIDs)
		for _, pkg := range repository {
			if _, included := packageIDs[pkg.ID]; included {
				changed = addReferences(packageIDs, repository, pkg.Requires) || changed
				changed = addReferences(packageIDs, repository, pkg.UpdateFor) || changed
			}
			if _, included := packageIDs[pkg.ID]; included {
				continue
			}
			for _, ref := range pkg.UpdateFor {
				if referenceMatchesIncluded(ref, packageIDs, includedSoftwareIDs) {
					packageIDs[pkg.ID] = struct{}{}
					changed = true
					break
				}
			}
		}
	}

	projection := repositoryProjection{
		effectivePackages: effective,
		packages:          make([]packages.Package, 0, len(packageIDs)),
		packageIDs:        packageIDs,
		iconObjectIDs:     make(map[int64]struct{}),
	}
	for _, pkg := range repository {
		if _, included := packageIDs[pkg.ID]; !included {
			continue
		}
		projection.packages = append(projection.packages, pkg)
		if pkg.Software.IconObjectID != nil {
			projection.iconObjectIDs[*pkg.Software.IconObjectID] = struct{}{}
		}
	}
	return projection
}

func includedSoftwareIDs(repository []packages.Package, packageIDs map[int64]struct{}) map[int64]struct{} {
	softwareIDs := make(map[int64]struct{}, len(packageIDs))
	for _, pkg := range repository {
		if _, included := packageIDs[pkg.ID]; included {
			softwareIDs[pkg.Software.ID] = struct{}{}
		}
	}
	return softwareIDs
}

func addReferences(
	packageIDs map[int64]struct{},
	repository []packages.Package,
	references []packages.PackageReference,
) bool {
	changed := false
	for _, ref := range references {
		if ref.PackageID > 0 {
			if _, included := packageIDs[ref.PackageID]; !included {
				packageIDs[ref.PackageID] = struct{}{}
				changed = true
			}
			continue
		}
		for _, pkg := range repository {
			if pkg.Software.ID != ref.SoftwareID {
				continue
			}
			if _, included := packageIDs[pkg.ID]; !included {
				packageIDs[pkg.ID] = struct{}{}
				changed = true
			}
		}
	}
	return changed
}

func referenceMatchesIncluded(
	ref packages.PackageReference,
	packageIDs map[int64]struct{},
	softwareIDs map[int64]struct{},
) bool {
	if ref.PackageID > 0 {
		_, included := packageIDs[ref.PackageID]
		return included
	}
	_, included := softwareIDs[ref.SoftwareID]
	return included
}
