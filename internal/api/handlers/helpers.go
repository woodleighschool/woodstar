package handlers

import (
	"strconv"

	"github.com/danielgtaylor/huma/v2"
)

// parseOptionalPositiveID parses an optional positive integer query/path value.
// Empty input returns (0, nil).
func parseOptionalPositiveID(id string, name string) (int64, error) {
	if id == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, huma.Error400BadRequest(name + " must be a positive integer")
	}
	return parsed, nil
}
