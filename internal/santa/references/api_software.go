package references

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const softwareTag = "Software"

type softwareGetInput struct {
	SoftwareID int64 `path:"software_id"`
}

type softwareSantaGetOutput struct {
	Body SoftwareReference
}

// RegisterSoftwareAdminRoutes registers Santa reference data for software titles.
func RegisterSoftwareAdminRoutes(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-software-santa-reference",
		Method:      http.MethodGet,
		Path:        "/api/software/{software_id}/santa",
		Tags:        []string{softwareTag},
		Summary:     "Get Santa reference data for a software title",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *softwareGetInput) (*softwareSantaGetOutput, error) {
		ref, err := store.GetSoftwareReference(ctx, input.SoftwareID)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("software title not found")
		}
		if err != nil {
			return nil, err
		}
		return &softwareSantaGetOutput{Body: *ref}, nil
	})
}
