package queries

import (
	"strings"

	"github.com/woodleighschool/woodstar/internal/platform"
)

func cleanStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := strings.TrimSpace(*value)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

func cleanPlatformPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := platform.CleanPlatform(*value)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}
