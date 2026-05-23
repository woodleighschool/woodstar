package handlers

import (
	"context"
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
	Body paginatedBody[users.User]
}

type userOutput struct {
	Body users.User
}

type userCreateInput struct {
	Body struct {
		Email    string     `json:"email"    format:"email"`
		Name     string     `json:"name,omitempty"`
		Role     users.Role `json:"role"     enum:"admin,viewer"`
		Password string     `json:"password" minLength:"12"`
	}
}

type userGetInput struct {
	ID string `path:"id"`
}

type userPutBody struct {
	Name     string     `json:"name"`
	Role     users.Role `json:"role"               enum:"admin,viewer"`
	Password *string    `json:"password,omitempty"`
}

type userPutInput struct {
	ID   string `path:"id"`
	Body userPutBody
}

type userDeleteInput struct {
	ID string `path:"id"`
}

func RegisterUsers(api huma.API, userService *users.Service) {
	registerListUsers(api, userService)
	registerCreateUser(api, userService)
	registerGetUser(api, userService)
	registerPutUser(api, userService)
	registerDeleteUser(api, userService)
}

func registerListUsers(api huma.API, userService *users.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "list-users",
		Method:      http.MethodGet,
		Path:        "/api/users",
		Tags:        []string{usersTag},
		Summary:     "List Woodstar users",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*userListOutput, error) {
		if _, err := requireAdmin(ctx); err != nil {
			return nil, err
		}
		list, err := userService.List(ctx)
		if err != nil {
			return nil, err
		}
		return &userListOutput{Body: paginatedBody[users.User]{Items: list, Count: len(list)}}, nil
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
		user, err := userService.Create(ctx, users.CreateParams{
			Email:    input.Body.Email,
			Name:     input.Body.Name,
			Role:     input.Body.Role,
			Password: input.Body.Password,
		})
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
		id, err := parseUserID(input.ID)
		if err != nil {
			return nil, err
		}
		user, err := userService.Get(ctx, id)
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
		targetID, err := parseUserID(input.ID)
		if err != nil {
			return nil, err
		}
		user, err := userService.Update(ctx, targetID, users.UpdateParams{
			Name:     input.Body.Name,
			Role:     input.Body.Role,
			Password: input.Body.Password,
		})
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
		targetID, err := parseUserID(input.ID)
		if err != nil {
			return nil, err
		}
		if err := userService.Delete(ctx, targetID); err != nil {
			return nil, userMutationError(err)
		}
		return &struct{}{}, nil
	})
}

func parseUserID(id string) (int64, error) {
	return parseResourceID(id, userResource)
}

// userMutationError extends resourceMutationError with user-owned mutation
// errors that don't apply to other resources.
func userMutationError(err error) error {
	if ok, mapped := mapSentinelHTTPError(err,
		staticSentinelHTTPError(dbutil.ErrAlreadyExists, huma.Error409Conflict("email already in use")),
		sentinelHTTPError{
			sentinel: users.ErrCannotDeleteInitialUser,
			response: func(err error) error {
				return huma.Error422UnprocessableEntity(err.Error())
			},
		},
		sentinelHTTPError{
			sentinel: users.ErrCannotModifyInitialUser,
			response: func(err error) error {
				return huma.Error422UnprocessableEntity(err.Error())
			},
		},
		staticSentinelHTTPError(users.ErrWeakPassword, huma.Error400BadRequest(users.ErrWeakPassword.Error())),
	); ok {
		return mapped
	}
	return resourceMutationError(userResource, err)
}
