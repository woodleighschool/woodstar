package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/users"
)

const (
	usersTag     = "Users"
	userResource = "user"
	userIDPath   = "/api/users/{id}"
)

type userListOutput struct {
	Body Page[users.User]
}

type departmentListOutput struct {
	Body Page[users.Department]
}

type userOutput struct {
	Body users.User
}

type userListInput struct {
	ListQueryInput
	Values []string `query:"values,omitempty"`
	Role   string   `query:"role,omitempty"   enum:"admin,viewer,none"`
	Source string   `query:"source,omitempty" enum:"local,synced"`
	Status string   `query:"status,omitempty" enum:"active,inactive"`
}

type departmentListInput struct {
	ListQueryInput
	Values []string `query:"values,omitempty"`
}

type userCreateInput struct {
	Body users.UserCreate
}

type userGetInput struct {
	ID int64 `path:"id"`
}

type userPutInput struct {
	ID   int64 `path:"id"`
	Body users.UserMutation
}

type userDeleteInput struct {
	ID int64 `path:"id"`
}

func RegisterUsers(api huma.API, userService *users.Service) {
	registerListUsers(api, userService)
	registerListUserDepartments(api, userService)
	registerCreateUser(api, userService)
	registerGetUser(api, userService)
	registerPutUser(api, userService)
	registerDeleteUser(api, userService)
}

func (i userListInput) params() users.ListParams {
	return users.ListParams{
		ListParams: i.ListQueryInput.params(),
		Values:     dbutil.SplitListValues(i.Values),
		Role:       i.Role,
		Source:     i.Source,
		Status:     i.Status,
	}
}

func (i departmentListInput) params() users.ListParams {
	return users.ListParams{
		ListParams: i.ListQueryInput.params(),
		Values:     dbutil.SplitListValues(i.Values),
	}
}

func registerListUsers(api huma.API, userService *users.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "list-users",
		Method:      http.MethodGet,
		Path:        "/api/users",
		Tags:        []string{usersTag},
		Summary:     "List Woodstar users",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *userListInput) (*userListOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		list, count, err := userService.List(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError(userResource, err)
		}
		return &userListOutput{Body: Page[users.User]{Items: list, Count: count}}, nil
	})
}

func registerListUserDepartments(api huma.API, userService *users.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "list-user-departments",
		Method:      http.MethodGet,
		Path:        "/api/users/departments",
		Tags:        []string{usersTag},
		Summary:     "List synced user departments",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *departmentListInput) (*departmentListOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		list, count, err := userService.ListDepartments(ctx, input.params())
		if err != nil {
			return nil, resourceMutationError("department", err)
		}
		return &departmentListOutput{Body: Page[users.Department]{Items: list, Count: count}}, nil
	})
}

func registerCreateUser(api huma.API, userService *users.Service) {
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
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		user, err := userService.Create(ctx, input.Body)
		if err != nil {
			return nil, userMutationError(err)
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerGetUser(api huma.API, userService *users.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "get-user",
		Method:      http.MethodGet,
		Path:        userIDPath,
		Tags:        []string{usersTag},
		Summary:     "Get a Woodstar user",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *userGetInput) (*userOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		user, err := userService.Get(ctx, input.ID)
		if err != nil {
			return nil, userMutationError(err)
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerPutUser(api huma.API, userService *users.Service) {
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
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		user, err := userService.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, userMutationError(err)
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerDeleteUser(api huma.API, userService *users.Service) {
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
		if _, err := requireAdmin(ctx); err != nil {
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
	case errors.Is(err, users.ErrWeakPassword):
		return huma.Error400BadRequest(users.ErrWeakPassword.Error())
	default:
		return resourceMutationError(userResource, err)
	}
}
