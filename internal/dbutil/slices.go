package dbutil

func NonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}
