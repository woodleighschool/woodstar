package apihelpers

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// HostsTag is the Huma operation tag shared across host-adjacent handler files.
const HostsTag = "Hosts"

// ParseResourceID parses a resource path ID. Returns 404 with a resource-named
// message on failure so unauthenticated probes never see "invalid id" errors.
func ParseResourceID(id string, resource string) (int64, error) {
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, huma.Error404NotFound(resource + " not found")
	}
	return parsed, nil
}

// ParseIDList validates every element as a positive int64, returning a 400 if
// any element fails.
func ParseIDList(values []int64, name string) ([]int64, error) {
	ids := make([]int64, 0, len(values))
	for _, id := range values {
		if id <= 0 {
			return nil, huma.Error400BadRequest(name + " includes a non-positive ID")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// CleanBulkIDs validates, deduplicates, and sorts a list of IDs for bulk operations.
func CleanBulkIDs(values []int64, name string) ([]int64, error) {
	ids, err := ParseIDList(values, name)
	if err != nil {
		return nil, err
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if len(ids) == 0 {
		return nil, huma.Error400BadRequest(name + " must include at least one ID")
	}
	return ids, nil
}

// ResourceMutationError translates store errors into Huma HTTP errors using
// resource-named messages.
func ResourceMutationError(resource string, err error) error {
	switch {
	case errors.Is(err, dbutil.ErrNotFound):
		return huma.Error404NotFound(resource + " not found")
	case errors.Is(err, dbutil.ErrAlreadyExists):
		return huma.Error409Conflict(resource + " already exists")
	case errors.Is(err, dbutil.ErrInvalidInput):
		return huma.Error400BadRequest(strings.TrimPrefix(err.Error(), dbutil.ErrInvalidInput.Error()+": "))
	default:
		return err
	}
}

// ParseHostID parses a host path ID.
func ParseHostID(id string) (int64, error) {
	return ParseResourceID(id, "host")
}
