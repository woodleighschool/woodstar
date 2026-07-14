package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/directory"
)

const (
	groupsTag     = "Groups"
	groupResource = "group"
	groupIDPath   = "/api/groups/{id}"
)

type groupListInput struct {
	ListQueryInput

	Values []string `query:"values,omitempty"`
}

type groupGetInput struct {
	ID int64 `path:"id"`
}

type groupListOutput struct {
	Body Page[directory.Group]
}

type groupOutput struct {
	Body directory.Group
}

func (i groupListInput) params() directory.GroupListParams {
	return directory.GroupListParams{
		ListParams: i.ListQueryInput.params(),
		Values:     dbutil.NormalizeListValues(i.Values),
	}
}

func registerListGroups(api huma.API, groupStore *directory.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-groups",
		Method:      http.MethodGet,
		Path:        "/api/groups",
		Tags:        []string{groupsTag},
		Summary:     "List directory groups",
	}, func(ctx context.Context, input *groupListInput) (*groupListOutput, error) {
		list, count, err := groupStore.ListGroups(ctx, input.params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-groups", groupResource, err)
		}
		return &groupListOutput{Body: Page[directory.Group]{Items: list, Count: count}}, nil
	})
}

func registerGetGroup(api huma.API, groupStore *directory.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-group",
		Method:      http.MethodGet,
		Path:        groupIDPath,
		Tags:        []string{groupsTag},
		Summary:     "Get a directory group",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *groupGetInput) (*groupOutput, error) {
		group, err := groupStore.GetGroupByID(ctx, input.ID)
		if err != nil {
			return nil, resourceError(ctx, logger, "get-group", groupResource, err, "group_id", input.ID)
		}
		return &groupOutput{Body: *group}, nil
	})
}
