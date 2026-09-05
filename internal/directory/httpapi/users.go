package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	authhuma "github.com/woodleighschool/goodies/auth/huma"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/directory/entra"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/listing"
	"github.com/woodleighschool/woodstar/internal/rbac"
)

const (
	userResource = "user"
	userIDPath   = "/api/users/{id}"
)

type userListOutput struct {
	Body api.Page[directory.User]
}

type departmentListOutput struct {
	Body api.Page[directory.Department]
}

type userOutput struct {
	Body directory.User
}

type userListInput struct {
	api.ListQueryInput

	Values  []string `query:"values,omitempty"`
	Role    []string `query:"role,omitempty"     enum:"admin,viewer,none"`
	Source  string   `query:"source,omitempty"   enum:"local,entra"`
	GroupID int64    `query:"group_id,omitempty"                          minimum:"1"`
}

type departmentListInput struct {
	api.ListQueryInput

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

// RegisterAPI mounts user and group admin endpoints.
func RegisterAPI(
	routes api.AppRoutes,
	userService *directory.UserService,
	store *directory.Store,
	syncJobs *entra.SyncJobs,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	registerAPI(routes, userService, store, syncJobs, authorizer, logger)
}

// RegisterOpenAPI documents directory endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerAPI(routes, nil, nil, nil, nil, nil)
}

func registerAPI(
	routes api.AppRoutes,
	userService *directory.UserService,
	store *directory.Store,
	syncJobs *entra.SyncJobs,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	directoryAPI := authhuma.ResourceAPI(routes.Protected, authorizer, logger, rbac.ResourceDirectory)
	usersAPI := authhuma.ResourceAPI(routes.Protected, authorizer, logger, rbac.ResourceUsers)
	groupsAPI := authhuma.ResourceAPI(routes.Protected, authorizer, logger, rbac.ResourceGroups)
	registerDirectorySync(directoryAPI, syncJobs, logger)
	registerListUsers(usersAPI, userService, logger)
	registerListUserDepartments(usersAPI, userService, logger)
	registerCreateUser(usersAPI, userService, logger)
	registerGetUser(usersAPI, userService, logger)
	registerPutUser(usersAPI, userService, logger)
	registerDeleteUser(usersAPI, userService, logger)
	registerListGroups(groupsAPI, store, logger)
	registerGetGroup(groupsAPI, store, logger)
}

func (i userListInput) params() directory.UserListParams {
	return directory.UserListParams{
		ListParams: i.Params(),
		Values:     listing.NormalizeValues(i.Values),
		Roles:      i.Role,
		Source:     i.Source,
		GroupID:    i.GroupID,
	}
}

func (i departmentListInput) params() directory.UserListParams {
	return directory.UserListParams{
		ListParams: i.Params(),
		Values:     listing.NormalizeValues(i.Values),
	}
}

func registerListUsers(humaAPI huma.API, userService *directory.UserService, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-users",
		Method:      http.MethodGet,
		Path:        "/api/users",
		Tags:        []string{api.TagDirectoryUsers},
		Summary:     "List users",
	}, func(ctx context.Context, input *userListInput) (*userListOutput, error) {
		list, count, err := userService.List(ctx, input.params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-users", userResource, err)
		}
		return &userListOutput{Body: api.Page[directory.User]{Items: list, Count: count}}, nil
	})
}

func registerListUserDepartments(humaAPI huma.API, userService *directory.UserService, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-user-departments",
		Method:      http.MethodGet,
		Path:        "/api/users/departments",
		Tags:        []string{api.TagDirectoryUsers},
		Summary:     "List user departments",
	}, func(ctx context.Context, input *departmentListInput) (*departmentListOutput, error) {
		list, count, err := userService.ListDepartments(ctx, input.params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-user-departments", "department", err)
		}
		return &departmentListOutput{Body: api.Page[directory.Department]{Items: list, Count: count}}, nil
	})
}

func registerCreateUser(humaAPI huma.API, userService *directory.UserService, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-user",
		Method:        http.MethodPost,
		Path:          "/api/users",
		Tags:          []string{api.TagDirectoryUsers},
		Summary:       "Create a user",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *userCreateInput) (*userOutput, error) {
		user, err := userService.Create(ctx, input.Body)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "create-user", userMutationError(err))
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerGetUser(humaAPI huma.API, userService *directory.UserService, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-user",
		Method:      http.MethodGet,
		Path:        userIDPath,
		Tags:        []string{api.TagDirectoryUsers},
		Summary:     "Get a user",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *userGetInput) (*userOutput, error) {
		user, err := userService.Get(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "get-user", userResource, err, "user_id", input.ID)
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerPutUser(humaAPI huma.API, userService *directory.UserService, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-user",
		Method:      http.MethodPut,
		Path:        userIDPath,
		Tags:        []string{api.TagDirectoryUsers},
		Summary:     "Update a user",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *userPutInput) (*userOutput, error) {
		user, err := userService.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "update-user", userMutationError(err), "user_id", input.ID)
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerDeleteUser(humaAPI huma.API, userService *directory.UserService, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "delete-user",
		Method:      http.MethodDelete,
		Path:        userIDPath,
		Tags:        []string{api.TagDirectoryUsers},
		Summary:     "Delete a user",
		Errors: []int{
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *userDeleteInput) (*struct{}, error) {
		if err := userService.Delete(ctx, input.ID); err != nil {
			return nil, api.HandlerError(ctx, logger, "delete-user", userMutationError(err), "user_id", input.ID)
		}
		return &struct{}{}, nil
	})
}

func userMutationError(err error) error {
	switch {
	case errors.Is(err, fault.ErrAlreadyExists):
		return huma.Error409Conflict("email already in use")
	case errors.Is(err, directory.ErrWeakPassword):
		return huma.Error400BadRequest(directory.ErrWeakPassword.Error())
	default:
		return api.ResourceMutationError(userResource, err)
	}
}
