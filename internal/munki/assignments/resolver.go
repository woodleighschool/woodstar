package assignments

// ResolveEffectivePackages applies assignment precedence to package candidates.
func ResolveEffectivePackages(packages []EffectivePackage) []EffectivePackage {
	resolved := make([][]EffectivePackage, 0, len(packages))
	selectedAssignments := make(map[int64]int64, len(packages))
	selectedIndexes := make(map[int64]int, len(packages))
	for _, pkg := range packages {
		if pkg.SoftwareID <= 0 {
			continue
		}
		assignmentID, exists := selectedAssignments[pkg.SoftwareID]
		if !exists {
			selectedAssignments[pkg.SoftwareID] = pkg.AssignmentID
			selectedIndexes[pkg.SoftwareID] = len(resolved)
			resolved = append(resolved, []EffectivePackage{pkg})
			continue
		}
		index := selectedIndexes[pkg.SoftwareID]
		if assignmentID == pkg.AssignmentID && index >= 0 {
			resolved[index] = appendUniqueEffectivePackage(resolved[index], pkg)
		}
	}
	out := make([]EffectivePackage, 0, len(packages))
	for _, group := range resolved {
		out = append(out, group...)
	}
	return out
}

func appendUniqueEffectivePackage(packages []EffectivePackage, pkg EffectivePackage) []EffectivePackage {
	for _, existing := range packages {
		if existing.Package.ID == pkg.Package.ID {
			return packages
		}
	}
	return append(packages, pkg)
}
