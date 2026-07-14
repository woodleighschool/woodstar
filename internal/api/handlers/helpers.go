package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func handlerError(ctx context.Context, logger *slog.Logger, operation string, err error, attrs ...any) error {
	if err == nil {
		return nil
	}

	status := http.StatusInternalServerError
	if statusErr, ok := errors.AsType[huma.StatusError](err); ok {
		status = statusErr.GetStatus()
	}
	if status >= http.StatusInternalServerError {
		args := []any{
			"operation", operation,
			"status", status,
		}
		args = append(args, attrs...)
		args = append(args, "err", err)
		logger.ErrorContext(ctx, "api handler failed", args...)
	}
	return err
}

func resourceError(
	ctx context.Context,
	logger *slog.Logger,
	operation string,
	resource string,
	err error,
	attrs ...any,
) error {
	return handlerError(ctx, logger, operation, resourceMutationError(resource, err), attrs...)
}

// resourceMutationError translates store errors into resource-shaped HTTP errors.
func resourceMutationError(resource string, err error) error {
	switch {
	case errors.Is(err, dbutil.ErrNotFound):
		return huma.Error404NotFound(resource + " not found")
	case errors.Is(err, dbutil.ErrAlreadyExists):
		return huma.Error409Conflict(resource + " already exists")
	case errors.Is(err, dbutil.ErrConflict):
		return huma.Error409Conflict(
			strings.TrimPrefix(err.Error(), dbutil.ErrConflict.Error()+": "),
		)
	case errors.Is(err, dbutil.ErrInvalidInput):
		return huma.Error400BadRequest(
			strings.TrimPrefix(err.Error(), dbutil.ErrInvalidInput.Error()+": "),
		)
	default:
		return err
	}
}

// BulkIDsBody is the shared request body for bulk-delete operations.
type BulkIDsBody struct {
	IDs []int64 `json:"ids"`
}

type Page[T any] struct {
	Items []T `json:"items" nullable:"false"`
	Count int `json:"count"`
}

// ListQueryInput is the shared query contract for paginated list endpoints. It
// carries optional q, 1-based page, per_page, and a single sort token such as
// name.asc or last_seen_at.desc. Per-resource filters are added as their own
// query fields, keyed by column ID.
type ListQueryInput struct {
	Q       string `query:"q,omitempty"`
	Page    int32  `query:"page,omitempty"     minimum:"1"`
	PerPage int32  `query:"per_page,omitempty" minimum:"1" maximum:"1000"`
	Sort    string `query:"sort,omitempty"`
}

func (input ListQueryInput) params() dbutil.ListParams {
	var pageIndex int32
	if input.Page > 1 {
		pageIndex = input.Page - 1
	}
	return dbutil.NormalizeListParams(dbutil.ListParams{
		Q:         input.Q,
		PageIndex: pageIndex,
		PageSize:  input.PerPage,
		Sort:      input.Sort,
	})
}
