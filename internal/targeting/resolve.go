package targeting

// Resolve applies include/exclude label target semantics for one host.
func Resolve[T any](
	includes []T,
	excludes []LabelRef,
	hostLabelIDs []int64,
	labelID func(T) int64,
) Result[T] {
	hostLabels := labelIDSet(hostLabelIDs)

	for _, exclude := range excludes {
		if _, ok := hostLabels[exclude.LabelID]; ok {
			return Result[T]{Excluded: true}
		}
	}

	for i := range includes {
		include := includes[i]
		if _, ok := hostLabels[labelID(include)]; ok {
			return Result[T]{
				Matched: true,
				Include: include,
			}
		}
	}

	return Result[T]{}
}

func labelIDSet(labelIDs []int64) map[int64]struct{} {
	out := make(map[int64]struct{}, len(labelIDs))
	for _, labelID := range labelIDs {
		out[labelID] = struct{}{}
	}
	return out
}
