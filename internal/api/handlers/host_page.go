package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

type hostPageInput struct {
	ListQueryInput

	ID int64 `path:"id"`
}

type hostPageOutput[T any] struct {
	Body Page[T]
}

func registerHostPage[T any](
	api huma.API,
	operationID string,
	path string,
	summary string,
	list func(context.Context, int64, dbutil.ListParams) ([]T, int, error),
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: operationID,
		Method:      http.MethodGet,
		Path:        path,
		Tags:        []string{hostsTag},
		Summary:     summary,
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *hostPageInput) (*hostPageOutput[T], error) {
		rows, count, err := list(ctx, input.ID, input.params())
		if err != nil {
			return nil, resourceError(
				ctx,
				logger,
				operationID,
				hostResource,
				err,
				"host_id", input.ID,
			)
		}
		return &hostPageOutput[T]{Body: Page[T]{Items: rows, Count: count}}, nil
	})
}
