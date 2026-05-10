package handlers

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/store"
)

// parseResourceID parses a resource path ID. Returns 404 with a resource-named
// message on failure so unauthenticated probes never see "invalid id" errors.
func parseResourceID(id string, resource string) (int64, error) {
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, huma.Error404NotFound(resource + " not found")
	}
	return parsed, nil
}

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

// parseIDList validates every element as a positive int64, returning a 400 if
// any element fails. Silent dropping is unacceptable: a client sending bad IDs
// would otherwise get a narrower scope than they intended with no signal.
func parseIDList(values []int64, name string) ([]int64, error) {
	ids := make([]int64, 0, len(values))
	for _, id := range values {
		if id <= 0 {
			return nil, huma.Error400BadRequest(name + " includes a non-positive ID")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func cleanBulkIDs(values []int64, name string) ([]int64, error) {
	ids, err := parseIDList(values, name)
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

// resourceMutationError translates store errors into Huma HTTP errors using
// resource-named messages. The not-found / already-exists / invalid-input
// triple is identical for every CRUD resource; we share it here.
func resourceMutationError(resource string, err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return huma.Error404NotFound(resource + " not found")
	case errors.Is(err, store.ErrAlreadyExists):
		return huma.Error409Conflict(resource + " already exists")
	case errors.Is(err, store.ErrInvalidInput):
		return huma.Error400BadRequest(strings.TrimPrefix(err.Error(), store.ErrInvalidInput.Error()+": "))
	default:
		return err
	}
}
