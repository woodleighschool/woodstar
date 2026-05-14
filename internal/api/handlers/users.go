package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/apihelpers"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/users"
)

const (
	usersTag     = "Users"
	userResource = "user"
	userIDPath   = "/api/users/{id}"
)

type userListOutput struct {
	Body []users.User
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

// RegisterUsers registers admin user management endpoints.
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
		return &userListOutput{Body: list}, nil
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
		OperationID: "put-user",
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
	return apihelpers.ParseResourceID(id, userResource)
}

// userMutationError extends ResourceMutationError with the initial-user
// lockout and weak-password cases that don't apply to other resources.
func userMutationError(err error) error {
	switch {
	case errors.Is(err, dbutil.ErrAlreadyExists):
		return huma.Error409Conflict("email already in use")
	case errors.Is(err, users.ErrCannotDeleteInitialUser),
		errors.Is(err, users.ErrCannotModifyInitialUser):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, users.ErrWeakPassword):
		return huma.Error400BadRequest(users.ErrWeakPassword.Error())
	default:
		return apihelpers.ResourceMutationError(userResource, err)
	}
}
