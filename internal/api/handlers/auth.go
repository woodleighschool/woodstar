package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/users"
)

type sessionOutput struct {
	Body sessionBody
}

type sessionBody struct {
	SetupComplete bool        `json:"setup_complete"`
	SSOEnabled    bool        `json:"sso_enabled"`
	User          *users.User `json:"user,omitempty"`
}

type authUserOutput struct {
	Body users.User
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

type contextKey int

const userContextKey contextKey = 0

const (
	authTag  = "Auth"
	setupTag = "Setup"
)

// RequireAuth attaches the signed-in user to protected admin Huma operations.
// Accepts an "Authorization: Bearer <api-key>" header first and falls back to
// the scs session cookie when no Bearer token is present.
func RequireAuth(api huma.API, authService *auth.Service) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		user, err := authService.Authenticate(ctx.Context(), ctx.Header("Authorization"))
		if err != nil {
			if errors.Is(err, auth.ErrNotAuthenticated) {
				_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "not authenticated")
				return
			}
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "request failed")
			return
		}

		next(huma.WithContext(ctx, withUser(ctx.Context(), user)))
	}
}

// RequireAdmin rejects authenticated users that are not administrators.
func RequireAdmin(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if _, err := requireAdmin(ctx.Context()); err != nil {
			var statusErr huma.StatusError
			if errors.As(err, &statusErr) {
				_ = huma.WriteErr(api, ctx, statusErr.GetStatus(), err.Error())
				return
			}
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "request failed")
			return
		}
		next(ctx)
	}
}

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
		return &authUserOutput{Body: *user}, nil
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
		out := &sessionOutput{Body: sessionBody{
			SetupComplete: complete,
			SSOEnabled:    authService.SSOEnabled(),
		}}
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
		out.Body.User = user
		return out, nil
	})
}

func registerLogin(api huma.API, authService *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "create-session",
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
		return &authUserOutput{Body: *user}, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "delete-session",
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

// requireAdmin returns the authenticated admin user from ctx.
// It returns a Huma 401 if no user is attached and 403 if the user is not an admin.
func requireAdmin(ctx context.Context) (*users.User, error) {
	user, ok := userFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	if user.Role != users.RoleAdmin {
		return nil, huma.Error403Forbidden("admin role required")
	}
	return user, nil
}

func withUser(ctx context.Context, user *users.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func userFromContext(ctx context.Context) (*users.User, bool) {
	user, ok := ctx.Value(userContextKey).(*users.User)
	return user, ok && user != nil
}

func authError(err error) error {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		return huma.Error401Unauthorized("invalid email or password")
	case errors.Is(err, auth.ErrNotAuthenticated):
		return huma.Error401Unauthorized("not authenticated")
	case errors.Is(err, auth.ErrNotSetup):
		return huma.Error409Conflict("setup required")
	case errors.Is(err, auth.ErrAlreadySetup):
		return huma.Error409Conflict("woodstar is already set up")
	case errors.Is(err, users.ErrWeakPassword):
		return huma.Error400BadRequest(users.ErrWeakPassword.Error())
	default:
		return err
	}
}
