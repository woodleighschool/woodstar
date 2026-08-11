package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/directory"
)

const (
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

// Dependencies are the services used by the authentication HTTP boundary.
type Dependencies struct {
	AuthService *auth.Service
	Users       *directory.UserService
	Logger      *slog.Logger
}

// RegisterAPI mounts session, account, and OIDC endpoints.
func RegisterAPI(routes api.AppRoutes, deps Dependencies) {
	registerOperations(routes, deps)
	registerOIDC(routes.Router, deps.AuthService, deps.Logger)
}

// RegisterOpenAPI documents authentication endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerOperations(routes, Dependencies{})
}

func registerOperations(routes api.AppRoutes, deps Dependencies) {
	registerGetSession(routes.Session, deps.AuthService)
	registerCreateSession(routes.PasswordLogin, deps.AuthService, deps.Logger)
	registerDeleteSession(routes.Protected, deps.AuthService, deps.Logger)

	registerGetAccount(routes.Protected, deps.AuthService, deps.Logger)
	registerPutAccount(routes.Protected, deps.Users, deps.Logger)
	registerRotateAPIKey(routes.Protected, deps.AuthService, deps.Logger)
	registerRevokeAPIKey(routes.Protected, deps.AuthService, deps.Logger)
}

func registerGetSession(humaAPI huma.API, authService *auth.Service) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-session",
		Method:      http.MethodGet,
		Path:        sessionPath,
		Tags:        []string{api.TagSession},
		Summary:     "Get session",
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

func registerCreateSession(humaAPI huma.API, authService *auth.Service, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "create-session",
		Method:      http.MethodPost,
		Path:        sessionPath,
		Tags:        []string{api.TagSession},
		Summary:     "Create a session",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusTooManyRequests},
	}, func(ctx context.Context, input *sessionCreateInput) (*sessionUserOutput, error) {
		user, err := authService.Login(ctx, auth.LoginParams{
			Email:    input.Body.Email,
			Password: input.Body.Password,
		})
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "create-session", authError(err))
		}
		return &sessionUserOutput{Body: *user}, nil
	})

	humaAPI.OpenAPI().Paths[sessionPath].Post.Responses["429"].Headers = map[string]*huma.Param{
		"Retry-After": {
			Description: "Seconds before another login attempt",
			Required:    true,
			Schema:      &huma.Schema{Type: "integer"},
		},
	}
}

func registerDeleteSession(humaAPI huma.API, authService *auth.Service, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "delete-session",
		Method:        http.MethodDelete,
		Path:          sessionPath,
		Tags:          []string{api.TagSession},
		Summary:       "Delete session",
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		if err := authService.Logout(ctx); err != nil {
			return nil, api.HandlerError(ctx, logger, "delete-session", err)
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
