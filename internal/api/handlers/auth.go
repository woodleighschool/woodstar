package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/directory"
)

const (
	authTag     = "Auth"
	accountTag  = "Account"
	sessionPath = "/api/session"
)

type sessionOutput struct {
	Body sessionBody
}

type sessionBody struct {
	SSOEnabled bool            `json:"sso_enabled"`
	User       *directory.User `json:"user,omitempty"`
}

type sessionUserOutput struct {
	Body directory.User
}

type sessionCreateInput struct {
	Body struct {
		Email    string `json:"email"    format:"email"`
		Password string `json:"password" minLength:"1"`
	}
}

// AuthHandlerDeps are the route groups and services used by auth handlers.
type AuthHandlerDeps struct {
	PasswordLogin huma.API
	Session       huma.API
	Protected     huma.API
	Router        chi.Router
	AuthService   *auth.Service
	Users         *directory.UserService
	Logger        *slog.Logger
}

// RegisterAuth mounts session, account, and OIDC endpoints.
func RegisterAuth(deps AuthHandlerDeps) {
	registerGetSession(deps.Session, deps.AuthService)
	registerCreateSession(deps.PasswordLogin, deps.AuthService, deps.Logger)
	registerDeleteSession(deps.Protected, deps.AuthService, deps.Logger)

	registerGetAccount(deps.Protected, deps.AuthService, deps.Logger)
	registerPutAccount(deps.Protected, deps.Users, deps.Logger)
	registerRotateAPIKey(deps.Protected, deps.AuthService, deps.Logger)
	registerRevokeAPIKey(deps.Protected, deps.AuthService, deps.Logger)
	registerOIDC(deps.Router, deps.AuthService, deps.Logger)
}

func registerGetSession(api huma.API, authService *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "get-session",
		Method:      http.MethodGet,
		Path:        sessionPath,
		Tags:        []string{authTag},
		Summary:     "Get the current signed-in user, if any",
	}, func(ctx context.Context, _ *struct{}) (*sessionOutput, error) {
		out := &sessionOutput{Body: sessionBody{
			SSOEnabled: authService.SSOEnabled(),
		}}
		if user, ok := ctxkeys.User(ctx); ok {
			out.Body.User = user
		}
		return out, nil
	})
}

func registerCreateSession(api huma.API, authService *auth.Service, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "create-session",
		Method:      http.MethodPost,
		Path:        sessionPath,
		Tags:        []string{authTag},
		Summary:     "Create a local user session",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusTooManyRequests},
	}, func(ctx context.Context, input *sessionCreateInput) (*sessionUserOutput, error) {
		user, err := authService.Login(ctx, auth.LoginParams{
			Email:    input.Body.Email,
			Password: input.Body.Password,
		})
		if err != nil {
			return nil, handlerError(ctx, logger, "create-session", authError(err))
		}
		return &sessionUserOutput{Body: *user}, nil
	})

	api.OpenAPI().Paths[sessionPath].Post.Responses["429"].Headers = map[string]*huma.Param{
		"Retry-After": {
			Description: "Seconds until another password-login attempt may be admitted",
			Required:    true,
			Schema:      &huma.Schema{Type: "integer"},
		},
	}
}

func registerDeleteSession(api huma.API, authService *auth.Service, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID:   "delete-session",
		Method:        http.MethodDelete,
		Path:          sessionPath,
		Tags:          []string{authTag},
		Summary:       "Revoke the current session",
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		if err := authService.Logout(ctx); err != nil {
			return nil, handlerError(ctx, logger, "delete-session", err)
		}
		return &struct{}{}, nil
	})
}

func authError(err error) error {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		return huma.Error401Unauthorized("invalid email or password")
	case errors.Is(err, auth.ErrNotAuthenticated):
		return huma.Error401Unauthorized("not authenticated")
	case errors.Is(err, directory.ErrWeakPassword):
		return huma.Error400BadRequest(directory.ErrWeakPassword.Error())
	default:
		return err
	}
}
