package software

// ResolveEffectivePackages applies include order to package candidates.
func ResolveEffectivePackages(packages []EffectivePackage) []EffectivePackage {
	resolved := make([]EffectivePackage, 0, len(packages))
	selectedTarget := make(map[int64]int64, len(packages))
	seen := make(map[softwarePackage]bool, len(packages))
	for _, pkg := range packages {
		softwareID := pkg.Package.Software.ID
		if softwareID <= 0 {
			continue
		}
		targetID, exists := selectedTarget[softwareID]
		if exists && targetID != pkg.TargetID {
			continue
		}
		selectedTarget[softwareID] = pkg.TargetID
		key := softwarePackage{softwareID: softwareID, packageID: pkg.Package.ID}
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
