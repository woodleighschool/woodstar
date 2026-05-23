package dbutil

import "strings"

func CleanStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	return StringPtrOrNil(*value)
}

func StringPtrOrNil(value string) *string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return nil
	}
	return new(cleaned)
}
