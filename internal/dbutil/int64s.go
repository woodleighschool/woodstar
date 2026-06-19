package dbutil

func Dedup[T comparable](values []T) []T {
	seen := make(map[T]struct{}, len(values))
	out := make([]T, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// SameInt64Set reports whether a and b contain the same values, ignoring order.
func SameInt64Set(a []int64, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[int64]int, len(a))
	for _, value := range a {
		counts[value]++
	}
	for _, value := range b {
		if counts[value] == 0 {
			return false
		}
		counts[value]--
	}
	return true
}
