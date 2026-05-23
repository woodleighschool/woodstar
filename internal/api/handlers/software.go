package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/software"
)

const softwareTag = "Software"

type softwareListInput struct {
	Page           int      `query:"page,omitempty"`
	PerPage        int      `query:"per_page,omitempty"`
	Q              string   `query:"q,omitempty"`
	OrderKey       string   `query:"order_key,omitempty"`
	OrderDirection string   `query:"order_direction,omitempty"`
	Source         []string `query:"source,omitempty"`
}

func (i softwareListInput) params() software.SoftwareTitleListParams {
	return software.SoftwareTitleListParams{
		ListParams: dbutil.ListParams{
			Q:              i.Q,
			Page:           i.Page,
			PerPage:        i.PerPage,
			OrderKey:       i.OrderKey,
			OrderDirection: i.OrderDirection,
		},
		SoftwareSources: i.Source,
	}
}

type softwareGetInput struct {
	ID string `path:"id"`
}

type softwareListOutput struct {
	Body paginatedBody[software.SoftwareTitle]
}

type softwareGetOutput struct {
	Body software.SoftwareTitle
}

func RegisterSoftware(api huma.API, softwareStore *software.Store) {
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
		return &softwareListOutput{Body: paginatedBody[software.SoftwareTitle]{Items: titles, Count: count}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-software",
		Method:      http.MethodGet,
		Path:        "/api/software/{id}",
		Tags:        []string{softwareTag},
		Summary:     "Get a software title",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *softwareGetInput) (*softwareGetOutput, error) {
		id, err := parseResourceID(input.ID, "software title")
		if err != nil {
			return nil, err
		}
		title, err := softwareStore.GetTitle(ctx, id)
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("software title not found")
		}
		if err != nil {
			return nil, err
		}
		return &softwareGetOutput{Body: *title}, nil
	})
}
