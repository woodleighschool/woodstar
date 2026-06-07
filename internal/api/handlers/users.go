package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/adminapi/adminctx"
	"github.com/woodleighschool/woodstar/internal/adminapi/apitypes"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/directory"
)

const (
	usersTag     = "Users"
	userResource = "user"
	userIDPath   = "/api/users/{id}"
)

type userListOutput struct {
	Body apitypes.Page[directory.User]
}

type departmentListOutput struct {
	Body apitypes.Page[directory.Department]
}

type userOutput struct {
	Body directory.User
}

type userListInput struct {
	apitypes.ListQueryInput
	Values  []string `query:"values,omitempty"`
	Role    string   `query:"role,omitempty"     enum:"admin,viewer,none"`
	Source  string   `query:"source,omitempty"   enum:"local,entra"`
	GroupID int64    `query:"group_id,omitempty"                          minimum:"1"`
}

type departmentListInput struct {
	apitypes.ListQueryInput
	Values []string `query:"values,omitempty"`
}

type userCreateInput struct {
	Body directory.UserCreate
}

type userGetInput struct {
	ID int64 `path:"id"`
}

type userPutInput struct {
	ID   int64 `path:"id"`
	Body directory.UserMutation
}

type userDeleteInput struct {
	ID int64 `path:"id"`
}

func RegisterUsers(api huma.API, userService *directory.UserService) {
	registerListUsers(api, userService)
	registerListUserDepartments(api, userService)
	registerCreateUser(api, userService)
	registerGetUser(api, userService)
	registerPutUser(api, userService)
	registerDeleteUser(api, userService)
}

func (i userListInput) params() directory.UserListParams {
	return directory.UserListParams{
		ListParams: i.ListQueryInput.Params(),
		Values:     dbutil.SplitListValues(i.Values),
		Role:       i.Role,
		Source:     i.Source,
		GroupID:    i.GroupID,
	}
}

func (i departmentListInput) params() directory.UserListParams {
	return directory.UserListParams{
		ListParams: i.ListQueryInput.Params(),
		Values:     dbutil.SplitListValues(i.Values),
	}
}

func registerListUsers(api huma.API, userService *directory.UserService) {
	huma.Register(api, huma.Operation{
		OperationID: "list-users",
		Method:      http.MethodGet,
		Path:        "/api/users",
		Tags:        []string{usersTag},
		Summary:     "List Woodstar users",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *userListInput) (*userListOutput, error) {
		if _, err := adminctx.RequireAdmin(ctx); err != nil {
			return nil, err
		}
		list, count, err := userService.List(ctx, input.params())
		if err != nil {
			return nil, apitypes.ResourceMutationError(userResource, err)
		}
		return &userListOutput{Body: apitypes.Page[directory.User]{Items: list, Count: count}}, nil
	})
}

func registerListUserDepartments(api huma.API, userService *directory.UserService) {
	huma.Register(api, huma.Operation{
		OperationID: "list-user-departments",
		Method:      http.MethodGet,
		Path:        "/api/users/departments",
		Tags:        []string{usersTag},
		Summary:     "List directory user departments",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *departmentListInput) (*departmentListOutput, error) {
		if _, err := adminctx.RequireAdmin(ctx); err != nil {
			return nil, err
		}
		list, count, err := userService.ListDepartments(ctx, input.params())
		if err != nil {
			return nil, apitypes.ResourceMutationError("department", err)
		}
		return &departmentListOutput{Body: apitypes.Page[directory.Department]{Items: list, Count: count}}, nil
	})
}

func registerCreateUser(api huma.API, userService *directory.UserService) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-user",
		Method:        http.MethodPost,
		Path:          "/api/users",
		Tags:          []string{usersTag},
		Summary:       "Create a Woodstar user",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *userCreateInput) (*userOutput, error) {
		if _, err := adminctx.RequireAdmin(ctx); err != nil {
			return nil, err
		}
		user, err := userService.Create(ctx, input.Body)
		if err != nil {
			return nil, userMutationError(err)
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerGetUser(api huma.API, userService *directory.UserService) {
	huma.Register(api, huma.Operation{
		OperationID: "get-user",
		Method:      http.MethodGet,
		Path:        userIDPath,
		Tags:        []string{usersTag},
		Summary:     "Get a Woodstar user",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *userGetInput) (*userOutput, error) {
		if _, err := adminctx.RequireAdmin(ctx); err != nil {
			return nil, err
		}
		user, err := userService.Get(ctx, input.ID)
		if err != nil {
			return nil, userMutationError(err)
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerPutUser(api huma.API, userService *directory.UserService) {
	huma.Register(api, huma.Operation{
		OperationID: "update-user",
		Method:      http.MethodPut,
		Path:        userIDPath,
		Tags:        []string{usersTag},
		Summary:     "Replace a Woodstar user",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *userPutInput) (*userOutput, error) {
		if _, err := adminctx.RequireAdmin(ctx); err != nil {
			return nil, err
		}
		user, err := userService.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, userMutationError(err)
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerDeleteUser(api huma.API, userService *directory.UserService) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-user",
		Method:      http.MethodDelete,
		Path:        userIDPath,
		Tags:        []string{usersTag},
		Summary:     "Delete a Woodstar user",
		Errors: []int{
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *userDeleteInput) (*struct{}, error) {
		if _, err := adminctx.RequireAdmin(ctx); err != nil {
			return nil, err
		}
		if err := userService.Delete(ctx, input.ID); err != nil {
			return nil, userMutationError(err)
		}
		return &struct{}{}, nil
	})
}

// userMutationError extends resourceMutationError with user-owned mutation
// errors that don't apply to other resources.
func userMutationError(err error) error {
	switch {
	case errors.Is(err, dbutil.ErrAlreadyExists):
		return huma.Error409Conflict("email already in use")
	case errors.Is(err, directory.ErrWeakPassword):
		return huma.Error400BadRequest(directory.ErrWeakPassword.Error())
	default:
		return apitypes.ResourceMutationError(userResource, err)
	}
}
