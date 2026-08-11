package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/listing"
)

const (
	groupResource = "group"
	groupIDPath   = "/api/groups/{id}"
)

type groupListInput struct {
	api.ListQueryInput

	Values []string `query:"values,omitempty"`
}

type groupGetInput struct {
	ID int64 `path:"id"`
}

type groupListOutput struct {
	Body api.Page[directory.Group]
}

type groupOutput struct {
	Body directory.Group
}

func (i groupListInput) params() directory.GroupListParams {
	return directory.GroupListParams{
		ListParams: i.Params(),
		Values:     listing.NormalizeValues(i.Values),
	}
}

func registerListGroups(humaAPI huma.API, groupStore *directory.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-groups",
		Method:      http.MethodGet,
		Path:        "/api/groups",
		Tags:        []string{api.TagDirectoryGroups},
		Summary:     "List groups",
	}, func(ctx context.Context, input *groupListInput) (*groupListOutput, error) {
		list, count, err := groupStore.ListGroups(ctx, input.params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-groups", groupResource, err)
		}
		return &groupListOutput{Body: api.Page[directory.Group]{Items: list, Count: count}}, nil
	})
}

func registerGetGroup(humaAPI huma.API, groupStore *directory.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-group",
		Method:      http.MethodGet,
		Path:        groupIDPath,
		Tags:        []string{api.TagDirectoryGroups},
		Summary:     "Get a group",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *groupGetInput) (*groupOutput, error) {
		group, err := groupStore.GetGroupByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "get-group", groupResource, err, "group_id", input.ID)
		}
		return &groupOutput{Body: *group}, nil
	})
}
