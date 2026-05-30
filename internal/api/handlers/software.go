package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa/references"
	"github.com/woodleighschool/woodstar/internal/software"
)

const softwareTag = "Software"

type softwareListInput struct {
	ListQueryInput
	Source []string `query:"source,omitempty"`
}

func (i softwareListInput) params() software.SoftwareTitleListParams {
	return software.SoftwareTitleListParams{
		ListParams:      i.ListQueryInput.params(),
		SoftwareSources: i.Source,
	}
}

type softwareGetInput struct {
	ID int64 `path:"id"`
}

type softwareListOutput struct {
	Body Page[software.SoftwareTitle]
}

type softwareGetOutput struct {
	Body software.SoftwareTitle
}

type softwareSantaGetOutput struct {
	Body references.SoftwareReference
}

func RegisterSoftware(api huma.API, softwareStore *software.Store, santaReferences *references.Store) {
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
			return nil, resourceMutationError("software", err)
		}
		return &softwareListOutput{Body: Page[software.SoftwareTitle]{Items: titles, Count: count}}, nil
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
