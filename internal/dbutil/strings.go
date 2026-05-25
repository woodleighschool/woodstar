package dbutil

// NullString returns a *string that is nil when value is empty. It exists so
// callers can map a model's plain string into a nullable Postgres column
// without inlining the same two-line check at every store call site.
func NullString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
