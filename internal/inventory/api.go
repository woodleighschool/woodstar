package inventory

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const softwareTag = "Software"

type softwareListInput struct {
	apitypes.ListQueryInput
	Source []string `query:"source,omitempty"`
}

func (i softwareListInput) params() SoftwareTitleListParams {
	return SoftwareTitleListParams{
		ListParams:      i.ListQueryInput.Params(),
		SoftwareSources: i.Source,
	}
}

type softwareGetInput struct {
	ID int64 `path:"id"`
}

type softwareListOutput struct {
	Body apitypes.Page[SoftwareTitle]
}

type softwareGetOutput struct {
	Body SoftwareTitle
}

// RegisterAdminRoutes registers observed software inventory admin endpoints.
func RegisterAdminRoutes(api huma.API, softwareStore *Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-software",
		Method:      http.MethodGet,
		Path:        "/api/software",
		Tags:        []string{softwareTag},
		Summary:     "List software titles",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *softwareListInput) (*softwareListOutput, error) {
		titles, count, err := softwareStore.ListTitles(ctx, input.params())
		if err != nil {
			return nil, apitypes.ResourceMutationError("software", err)
		}
		return &softwareListOutput{Body: apitypes.Page[SoftwareTitle]{Items: titles, Count: count}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-software",
		Method:      http.MethodGet,
		Path:        "/api/software/{id}",
		Tags:        []string{softwareTag},
		Summary:     "Get a software title",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *softwareGetInput) (*softwareGetOutput, error) {
		title, err := softwareStore.GetTitle(ctx, input.ID)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("software title not found")
		}
		if err != nil {
			return nil, err
		}
		return &softwareGetOutput{Body: *title}, nil
	})
}
