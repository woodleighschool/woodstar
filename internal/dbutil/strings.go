package dbutil

// NullString returns nil when value is empty.
func NullString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
