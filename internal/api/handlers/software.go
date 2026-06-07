package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/inventory"
	"github.com/woodleighschool/woodstar/internal/santa/references"
)

const softwareTag = "Software"

type softwareListInput struct {
	apitypes.ListQueryInput
	Source []string `query:"source,omitempty"`
}

func (i softwareListInput) params() inventory.SoftwareTitleListParams {
	return inventory.SoftwareTitleListParams{
		ListParams:      i.ListQueryInput.Params(),
		SoftwareSources: i.Source,
	}
}

type softwareGetInput struct {
	ID int64 `path:"id"`
}

type softwareListOutput struct {
	Body apitypes.Page[inventory.SoftwareTitle]
}

type softwareGetOutput struct {
	Body inventory.SoftwareTitle
}

type softwareSantaGetOutput struct {
	Body references.SoftwareReference
}

func RegisterSoftware(api huma.API, softwareStore *inventory.Store, santaReferences *references.Store) {
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
		return &softwareListOutput{Body: apitypes.Page[inventory.SoftwareTitle]{Items: titles, Count: count}}, nil
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

	huma.Register(api, huma.Operation{
		OperationID: "get-software-santa-reference",
		Method:      http.MethodGet,
		Path:        "/api/software/{id}/santa",
		Tags:        []string{softwareTag},
		Summary:     "Get Santa reference data for a software title",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *softwareGetInput) (*softwareSantaGetOutput, error) {
		ref, err := santaReferences.GetSoftwareReference(ctx, input.ID)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("software title not found")
		}
		if err != nil {
			return nil, err
		}
		return &softwareSantaGetOutput{Body: *ref}, nil
	})
}
