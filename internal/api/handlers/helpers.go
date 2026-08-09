package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/listing"
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
		args := make([]any, 0, 6+len(attrs))
		args = append(args,
			"operation", operation,
			"status", status,
		)
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

// resourceMutationError translates store errors into HTTP errors.
func resourceMutationError(resource string, err error) error {
	switch {
	case errors.Is(err, fault.ErrNotFound):
		return huma.Error404NotFound("")
	case errors.Is(err, fault.ErrAlreadyExists):
		return huma.Error409Conflict(resource + " already exists")
	case errors.Is(err, fault.ErrConflict):
		return huma.Error409Conflict(
			strings.TrimPrefix(err.Error(), fault.ErrConflict.Error()+": "),
		)
	case errors.Is(err, fault.ErrInvalidInput):
		return huma.Error400BadRequest(
			strings.TrimPrefix(err.Error(), fault.ErrInvalidInput.Error()+": "),
		)
	default:
		return err
	}
}

type deleteManyInput struct {
	IDs []int64 `query:"ids" required:"true" minItems:"1"`
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

func (input ListQueryInput) params() listing.Params {
	var pageIndex int32
	if input.Page > 1 {
		pageIndex = input.Page - 1
	}
	return listing.Normalize(listing.Params{
		Q:         input.Q,
		PageIndex: pageIndex,
		PageSize:  input.PerPage,
		Sort:      input.Sort,
	})
}
