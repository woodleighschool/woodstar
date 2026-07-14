package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/references"
)

type softwareSantaGetInput struct {
	ID int64 `path:"id"`
}

type softwareSantaGetOutput struct {
	Body references.SoftwareReference
}

func registerSoftwareSantaReference(api huma.API, store *references.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-software-santa-reference",
		Method:      http.MethodGet,
		Path:        "/api/software/{id}/santa",
		Tags:        []string{softwareTag},
		Summary:     "Get Santa reference data for a software title",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *softwareSantaGetInput) (*softwareSantaGetOutput, error) {
		ref, err := store.GetSoftwareReference(ctx, input.ID)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("software title not found")
		}
		if err != nil {
			return nil, handlerError(ctx, logger, "get-software-santa-reference", err, "software_id", input.ID)
		}
		return &softwareSantaGetOutput{Body: *ref}, nil
	})
}
