package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/models"
)

const usersTag = "Users"

type userListOutput struct {
	Body []userBody
}

type userOutput struct {
	Body userBody
}

type userCreateInput struct {
	Body struct {
		Email    string          `json:"email" format:"email"`
		Name     string          `json:"name,omitempty"`
		Role     models.UserRole `json:"role" enum:"admin,viewer"`
		Password string          `json:"password" minLength:"12"`
	}
}

type userUpdateInput struct {
	ID   string `path:"id"`
	Body struct {
		Name     *string          `json:"name,omitempty"`
		Role     *models.UserRole `json:"role,omitempty" enum:"admin,viewer"`
		Password *string          `json:"password,omitempty" minLength:"12"`
	}
}

type userDeleteInput struct {
	ID string `path:"id"`
}

// RegisterUsers registers admin user management endpoints.
func RegisterUsers(api huma.API, authService *auth.Service) {
	registerListUsers(api, authService)
	registerCreateUser(api, authService)
	registerUpdateUser(api, authService)
	registerDeleteUser(api, authService)
}

func registerListUsers(api huma.API, authService *auth.Service) {
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
		users, err := authService.ListUsers(ctx)
		if err != nil {
			return nil, err
		}
		out := &userListOutput{Body: make([]userBody, 0, len(users))}
		for i := range users {
			out.Body = append(out.Body, userResponse(&users[i]))
		}
		return out, nil
	})
}

func registerCreateUser(api huma.API, authService *auth.Service) {
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
		user, err := authService.CreateUser(ctx, auth.CreateUserParams{
			Email:    input.Body.Email,
			Name:     input.Body.Name,
			Role:     input.Body.Role,
			Password: input.Body.Password,
		})
		if err != nil {
			return nil, userMutationError(err)
		}
		return &userOutput{Body: userResponse(user)}, nil
	})
}

func registerUpdateUser(api huma.API, authService *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "update-user",
		Method:      http.MethodPatch,
		Path:        "/api/users/{id}",
		Tags:        []string{usersTag},
		Summary:     "Update a Woodstar user",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *userUpdateInput) (*userOutput, error) {
		actor, err := requireAdmin(ctx)
		if err != nil {
			return nil, err
		}
		targetID, err := parseUserID(input.ID)
		if err != nil {
			return nil, err
		}
		user, err := authService.UpdateUser(ctx, actor.ID, targetID, auth.UpdateUserParams{
			Name:     input.Body.Name,
			Role:     input.Body.Role,
			Password: input.Body.Password,
		})
		if err != nil {
			return nil, userMutationError(err)
		}
		return &userOutput{Body: userResponse(user)}, nil
	})
}

func registerDeleteUser(api huma.API, authService *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-user",
		Method:      http.MethodDelete,
		Path:        "/api/users/{id}",
		Tags:        []string{usersTag},
		Summary:     "Delete a Woodstar user",
		Errors: []int{
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *userDeleteInput) (*struct{}, error) {
		actor, err := requireAdmin(ctx)
		if err != nil {
			return nil, err
		}
		targetID, err := parseUserID(input.ID)
		if err != nil {
			return nil, err
		}
		if err := authService.DeleteUser(ctx, actor.ID, targetID); err != nil {
			return nil, userMutationError(err)
		}
		return &struct{}{}, nil
	})
}

func parseUserID(id string) (int64, error) {
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, huma.Error404NotFound("user not found")
	}
	return parsed, nil
}

func userMutationError(err error) error {
	switch {
	case errors.Is(err, models.ErrNotFound):
		return huma.Error404NotFound("user not found")
	case errors.Is(err, models.ErrAlreadyExists):
		return huma.Error409Conflict("email already in use")
	case errors.Is(err, auth.ErrCannotChangeOwnRole),
		errors.Is(err, auth.ErrCannotDeleteSelf),
		errors.Is(err, auth.ErrCannotRemoveLastAdmin):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, auth.ErrWeakPassword):
		return huma.Error400BadRequest(auth.ErrWeakPassword.Error())
	default:
		return err
	}
}
