package targeting

import "fmt"

// ValidateLabelSets enforces shared include/exclude label targeting rules.
func ValidateLabelSets(includes, excludes []LabelRef) error {
	return ValidateTargets(includes, excludes, func(ref LabelRef) int64 {
		return ref.LabelID
	})
}

// ValidateTargets enforces shared include/exclude label targeting rules.
func ValidateTargets[T any](includes []T, excludes []LabelRef, includeLabelID func(T) int64) error {
	if err := ValidateUniqueLabels(Include, includes, includeLabelID); err != nil {
		return err
	}
	if err := ValidateUniqueLabelRefs(Exclude, excludes); err != nil {
		return err
	}
	return ValidateNoLabelOverlap(includes, excludes, includeLabelID)
}

// ValidateUniqueLabelRefs rejects duplicate label IDs within one target set.
func ValidateUniqueLabelRefs(direction Direction, refs []LabelRef) error {
	return ValidateUniqueLabels(direction, refs, func(ref LabelRef) int64 {
		return ref.LabelID
	})
}

// ValidateUniqueLabels rejects duplicate label IDs within one target set.
func ValidateUniqueLabels[T any](direction Direction, rows []T, labelID func(T) int64) error {
	if !ValidDirection(direction) {
		return fmt.Errorf("targeting: unsupported direction %q", direction)
	}

	seen := make(map[int64]struct{}, len(rows))
	for _, row := range rows {
		id := labelID(row)
		if _, ok := seen[id]; ok {
			return fmt.Errorf("targeting: duplicate %s label_id %d", direction, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

// ValidateNoLabelOverlap rejects labels that appear in both include and exclude sets.
func ValidateNoLabelOverlap[T any](includes []T, excludes []LabelRef, includeLabelID func(T) int64) error {
	excluded := make(map[int64]struct{}, len(excludes))
	for _, exclude := range excludes {
		excluded[exclude.LabelID] = struct{}{}
	}

	for _, include := range includes {
		labelID := includeLabelID(include)
		if _, ok := excluded[labelID]; ok {
			return fmt.Errorf("targeting: label_id %d is both included and excluded", labelID)
		}
	}
	return nil
}
