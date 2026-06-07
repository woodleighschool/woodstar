package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/adminctx"
	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/directory"
)

const (
	groupsTag     = "Groups"
	groupResource = "group"
	groupIDPath   = "/api/groups/{id}"
)

type groupListInput struct {
	apitypes.ListQueryInput
	Values []string `query:"values,omitempty"`
}

type groupGetInput struct {
	ID int64 `path:"id"`
}

type groupListOutput struct {
	Body apitypes.Page[directory.Group]
}

type groupOutput struct {
	Body directory.Group
}

func RegisterGroups(api huma.API, groupStore *directory.Store) {
	registerListGroups(api, groupStore)
	registerGetGroup(api, groupStore)
}

func (i groupListInput) params() directory.GroupListParams {
	return directory.GroupListParams{
		ListParams: i.ListQueryInput.Params(),
		Values:     dbutil.SplitListValues(i.Values),
	}
}

func registerListGroups(api huma.API, groupStore *directory.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-groups",
		Method:      http.MethodGet,
		Path:        "/api/groups",
		Tags:        []string{groupsTag},
		Summary:     "List directory groups",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *groupListInput) (*groupListOutput, error) {
		if _, err := adminctx.RequireAdmin(ctx); err != nil {
			return nil, err
		}
		list, count, err := groupStore.ListGroups(ctx, input.params())
		if err != nil {
			return nil, apitypes.ResourceMutationError(groupResource, err)
		}
		return &groupListOutput{Body: apitypes.Page[directory.Group]{Items: list, Count: count}}, nil
	})
}

func registerGetGroup(api huma.API, groupStore *directory.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-group",
		Method:      http.MethodGet,
		Path:        groupIDPath,
		Tags:        []string{groupsTag},
		Summary:     "Get a directory group",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *groupGetInput) (*groupOutput, error) {
		if _, err := adminctx.RequireAdmin(ctx); err != nil {
			return nil, err
		}
		group, err := groupStore.GetGroupByID(ctx, input.ID)
		if err != nil {
			return nil, apitypes.ResourceMutationError(groupResource, err)
		}
		return &groupOutput{Body: *group}, nil
	})
}
