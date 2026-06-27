package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/hosts"
)

type hostStateLookupInput struct {
	HostID int64 `path:"id"`
}

type hostStateOutput[T any] struct {
	Body T
}

func registerHostState[T any](
	api huma.API,
	operationID string,
	path string,
	summary string,
	missingMessage string,
	hostStore *hosts.Store,
	load func(context.Context, int64) (*T, error),
	logger *slog.Logger,
) {
	huma.Register(api, huma.Operation{
		OperationID: operationID,
		Method:      http.MethodGet,
		Path:        path,
		Tags:        []string{hostsTag},
		Summary:     summary,
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *hostStateLookupInput) (*hostStateOutput[T], error) {
		if _, err := hostStore.GetByID(ctx, input.HostID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("host not found")
		} else if err != nil {
			return nil, handlerError(ctx, logger, operationID, err, "host_id", input.HostID)
		}
		state, err := load(ctx, input.HostID)
		if err != nil {
			return nil, handlerError(ctx, logger, operationID, err, "host_id", input.HostID)
		}
		if state == nil {
			return nil, huma.Error404NotFound(missingMessage)
		}
		return &hostStateOutput[T]{Body: *state}, nil
	})
}
