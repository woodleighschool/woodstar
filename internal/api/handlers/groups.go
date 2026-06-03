package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/groups"
	"github.com/woodleighschool/woodstar/internal/users"
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

type groupMemberListInput struct {
	ID int64 `path:"id"`
	ListQueryInput
	Values []string `          query:"values,omitempty"`
	Role   string   `          query:"role,omitempty"   enum:"admin,viewer,none"`
	Source string   `          query:"source,omitempty" enum:"local,synced"`
	Status string   `          query:"status,omitempty" enum:"active,inactive"`
}

type groupListOutput struct {
	Body Page[groups.Group]
}

type groupOutput struct {
	Body groups.Group
}

type groupMembersOutput struct {
	Body Page[users.User]
}

func RegisterGroups(api huma.API, groupStore *groups.Store, userService *users.Service) {
	registerListGroups(api, groupStore)
	registerGetGroup(api, groupStore)
	registerListGroupMembers(api, groupStore, userService)
}

func (i groupListInput) params() groups.ListParams {
	return groups.ListParams{
		ListParams: i.ListQueryInput.params(),
		Values:     dbutil.SplitListValues(i.Values),
	}
}

func (i groupMemberListInput) params() users.ListParams {
	return users.ListParams{
		ListParams: i.ListQueryInput.params(),
		Values:     dbutil.SplitListValues(i.Values),
		Role:       i.Role,
		Source:     i.Source,
		Status:     i.Status,
	}
}

func registerListGroups(api huma.API, groupStore *groups.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-groups",
		Method:      http.MethodGet,
		Path:        "/api/groups",
		Tags:        []string{groupsTag},
		Summary:     "List directory groups",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *groupListInput) (*groupListOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		list, count, err := groupStore.List(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(groupResource, err)
		}
		return &groupListOutput{Body: Page[groups.Group]{Items: list, Count: count}}, nil
	})
}

func registerGetGroup(api huma.API, groupStore *groups.Store) {
	huma.Register(api, huma.Operation{
		OperationID: "get-group",
		Method:      http.MethodGet,
		Path:        groupIDPath,
		Tags:        []string{groupsTag},
		Summary:     "Get a directory group",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *groupGetInput) (*groupOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		group, err := groupStore.GetByID(ctx, input.ID)
		if err != nil {
			return nil, resourceMutationError(groupResource, err)
		}
		return &groupOutput{Body: *group}, nil
	})
}

func registerListGroupMembers(api huma.API, groupStore *groups.Store, userService *users.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "list-group-members",
		Method:      http.MethodGet,
		Path:        "/api/groups/{id}/members",
		Tags:        []string{groupsTag, usersTag},
		Summary:     "List directory group members",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *groupMemberListInput) (*groupMembersOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		if _, err := groupStore.GetByID(ctx, input.ID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("group not found")
		} else if err != nil {
			return nil, err
		}
		list, count, err := userService.ListGroupMembers(ctx, input.ID, input.params())
		if err != nil {
			return nil, resourceMutationError(userResource, err)
		}
		return &groupMembersOutput{Body: Page[users.User]{Items: list, Count: count}}, nil
	})
}
