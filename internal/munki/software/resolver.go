package software

// ResolveEffectivePackages applies include order to package candidates.
func ResolveEffectivePackages(packages []EffectivePackage) []EffectivePackage {
	resolved := make([]EffectivePackage, 0, len(packages))
	selectedTarget := make(map[int64]int64, len(packages))
	seen := make(map[softwarePackage]bool, len(packages))
	for _, pkg := range packages {
		if pkg.SoftwareID <= 0 {
			continue
		}
		targetID, exists := selectedTarget[pkg.SoftwareID]
		if exists && targetID != pkg.TargetID {
			continue
		}
		selectedTarget[pkg.SoftwareID] = pkg.TargetID
		key := softwarePackage{softwareID: pkg.SoftwareID, packageID: pkg.Package.ID}
		if seen[key] {
			continue
		}
		seen[key] = true
		resolved = append(resolved, pkg)
	}
	return resolved
}

type softwarePackage struct {
	softwareID int64
	packageID  int64
}
