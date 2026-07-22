package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type hostStateLookupInput struct {
	ID int64 `path:"id"`
}

type hostStateOutput[T any] struct {
	Body T
}

func registerHostState[T any](
	api huma.API,
	operationID string,
	path string,
	summary string,
	load func(context.Context, int64) (*T, error),
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: operationID,
		Method:      http.MethodGet,
		Path:        path,
		Tags:        []string{hostsTag},
		Summary:     summary,
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *hostStateLookupInput) (*hostStateOutput[T], error) {
		state, err := load(ctx, input.ID)
		if err != nil {
			return nil, handlerError(ctx, logger, operationID, err, "host_id", input.ID)
		}
		if state == nil {
			return nil, huma.Error404NotFound("")
		}
		return &hostStateOutput[T]{Body: *state}, nil
	})
}
