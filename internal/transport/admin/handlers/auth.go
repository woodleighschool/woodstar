package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/models"
	"github.com/woodleighschool/woodstar/internal/transport/admin/adminctx"
)

type userBody struct {
	ID        int64           `json:"id"`
	Email     string          `json:"email"`
	Name      string          `json:"name"`
	Role      models.UserRole `json:"role"       enum:"admin,viewer"`
	CreatedAt time.Time       `json:"created_at"`
}

type sessionOutput struct {
	Body sessionBody
}

type sessionBody struct {
	SetupComplete bool      `json:"setup_complete"`
	User          *userBody `json:"user,omitempty"`
}

type authUserOutput struct {
	Body userBody
}

type setupInput struct {
	Body struct {
		Email    string `json:"email"    format:"email"`
		Name     string `json:"name,omitempty"`
		Password string `json:"password" minLength:"12"`
	}
}

type loginInput struct {
	Body struct {
		Email    string `json:"email"    format:"email"`
		Password string `json:"password" minLength:"1"`
	}
}

const (
	authTag  = "Auth"
	setupTag = "Setup"
)

// RegisterPublicAuth registers setup and browser session endpoints.
func RegisterPublicAuth(api huma.API, authService *auth.Service) {
	registerSetup(api, authService)
	registerSession(api, authService)
	registerLogin(api, authService)
}

func registerSetup(api huma.API, authService *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID:   "complete-setup",
		Method:        http.MethodPost,
		Path:          "/api/setup",
		Tags:          []string{setupTag},
		Summary:       "Create the first administrator account",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusConflict},
	}, func(ctx context.Context, input *setupInput) (*authUserOutput, error) {
		user, err := authService.Setup(ctx, auth.SetupParams{
			Email:    input.Body.Email,
			Name:     input.Body.Name,
			Password: input.Body.Password,
		})
		if err != nil {
			return nil, authError(err)
		}
		return &authUserOutput{Body: userResponse(user)}, nil
	})
}

func registerSession(api huma.API, authService *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "get-session",
		Method:      http.MethodGet,
		Path:        "/api/auth/session",
		Tags:        []string{authTag},
		Summary:     "Get setup state and the current signed-in user, if any",
	}, func(ctx context.Context, _ *struct{}) (*sessionOutput, error) {
		complete, err := authService.SetupComplete(ctx)
		if err != nil {
			return nil, err
		}
		out := &sessionOutput{Body: sessionBody{SetupComplete: complete}}
		if !complete {
			return out, nil
		}
		user, err := authService.CurrentUser(ctx)
		if err != nil {
			if errors.Is(err, auth.ErrNotAuthenticated) {
				return out, nil
			}
			return nil, err
		}
		body := userResponse(user)
		out.Body.User = &body
		return out, nil
	})
}

func registerLogin(api huma.API, authService *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/api/auth/login",
		Tags:        []string{authTag},
		Summary:     "Create a local admin session",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusConflict},
	}, func(ctx context.Context, input *loginInput) (*authUserOutput, error) {
		user, err := authService.Login(ctx, input.Body.Email, input.Body.Password)
		if err != nil {
			return nil, authError(err)
		}
		return &authUserOutput{Body: userResponse(user)}, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      http.MethodPost,
		Path:        "/api/auth/logout",
		Tags:        []string{authTag},
		Summary:     "Revoke the current session",
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		if err := authService.Logout(ctx); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}

func userResponse(user *models.User) userBody {
	return userBody{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}

// requireAdmin returns the authenticated admin user from ctx.
// It returns a Huma 401 if no user is attached and 403 if the user is not an admin.
func requireAdmin(ctx context.Context) (*models.User, error) {
	user, ok := adminctx.UserFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	if user.Role != models.RoleAdmin {
		return nil, huma.Error403Forbidden("admin role required")
	}
	return user, nil
}

func authError(err error) error {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrNotAuthenticated):
		return huma.Error401Unauthorized("not authenticated")
	case errors.Is(err, auth.ErrNotSetup):
		return huma.Error409Conflict("setup required")
	case errors.Is(err, auth.ErrAlreadySetup):
		return huma.Error409Conflict("woodstar is already set up")
	case errors.Is(err, auth.ErrWeakPassword):
		return huma.Error400BadRequest(auth.ErrWeakPassword.Error())
	case strings.TrimSpace(err.Error()) == "":
		return huma.Error500InternalServerError("request failed")
	default:
		return err
	}
}
