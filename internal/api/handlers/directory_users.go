package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/directory"
)

const (
	usersTag     = "Users"
	userResource = "user"
	userIDPath   = "/api/users/{id}"
)

type userListOutput struct {
	Body Page[directory.User]
}

type departmentListOutput struct {
	Body Page[directory.Department]
}

type userOutput struct {
	Body directory.User
}

type userListInput struct {
	ListQueryInput

	Values  []string `query:"values,omitempty"`
	Role    string   `query:"role,omitempty"     enum:"admin,viewer,none"`
	Source  string   `query:"source,omitempty"   enum:"local,entra"`
	GroupID int64    `query:"group_id,omitempty"                          minimum:"1"`
}

type departmentListInput struct {
	ListQueryInput

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

// RegisterDirectory mounts user and group admin endpoints.
func RegisterDirectory(
	api huma.API,
	userService *directory.UserService,
	store *directory.Store,
	logger *slog.Logger,
) {
	registerListUsers(api, userService, logger)
	registerListUserDepartments(api, userService, logger)
	registerCreateUser(api, userService, logger)
	registerGetUser(api, userService, logger)
	registerPutUser(api, userService, logger)
	registerDeleteUser(api, userService, logger)
	registerListGroups(api, store, logger)
	registerGetGroup(api, store, logger)
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

func registerListUsers(api huma.API, userService *directory.UserService, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-users",
		Method:      http.MethodGet,
		Path:        "/api/users",
		Tags:        []string{usersTag},
		Summary:     "List Woodstar users",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *userListInput) (*userListOutput, error) {
		list, count, err := userService.List(ctx, input.params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-users", userResource, err)
		}
		return &userListOutput{Body: Page[directory.User]{Items: list, Count: int32(count)}}, nil
	})
}

func registerListUserDepartments(api huma.API, userService *directory.UserService, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-user-departments",
		Method:      http.MethodGet,
		Path:        "/api/users/departments",
		Tags:        []string{usersTag},
		Summary:     "List directory user departments",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *departmentListInput) (*departmentListOutput, error) {
		list, count, err := userService.ListDepartments(ctx, input.params())
		if err != nil {
			return nil, resourceError(ctx, logger, "list-user-departments", "department", err)
		}
		return &departmentListOutput{Body: Page[directory.Department]{Items: list, Count: int32(count)}}, nil
	})
}

func registerCreateUser(api huma.API, userService *directory.UserService, logger *slog.Logger) {
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
		user, err := userService.Create(ctx, input.Body)
		if err != nil {
			return nil, handlerError(ctx, logger, "create-user", userMutationError(err))
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerGetUser(api huma.API, userService *directory.UserService, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-user",
		Method:      http.MethodGet,
		Path:        userIDPath,
		Tags:        []string{usersTag},
		Summary:     "Get a Woodstar user",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *userGetInput) (*userOutput, error) {
		user, err := userService.Get(ctx, input.ID)
		if err != nil {
			return nil, handlerError(ctx, logger, "get-user", userMutationError(err), "user_id", input.ID)
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerPutUser(api huma.API, userService *directory.UserService, logger *slog.Logger) {
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
		user, err := userService.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, handlerError(ctx, logger, "update-user", userMutationError(err), "user_id", input.ID)
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerDeleteUser(api huma.API, userService *directory.UserService, logger *slog.Logger) {
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
		if err := userService.Delete(ctx, input.ID); err != nil {
			return nil, handlerError(ctx, logger, "delete-user", userMutationError(err), "user_id", input.ID)
		}
		return &struct{}{}, nil
	})
}

func userMutationError(err error) error {
	switch {
	case errors.Is(err, dbutil.ErrAlreadyExists):
		return huma.Error409Conflict("email already in use")
	case errors.Is(err, directory.ErrWeakPassword):
		return huma.Error400BadRequest(directory.ErrWeakPassword.Error())
	default:
		return ResourceMutationError(userResource, err)
	}
}
