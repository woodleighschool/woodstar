package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

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
	User       *auth.Principal `json:"user,omitempty"`
}

type principalOutput struct {
	Body auth.Principal
}

type sessionCreateInput struct {
	Body struct {
		Email    string `json:"email"    format:"email"`
		Password string `json:"password" minLength:"1"`
	}
}

// AuthHandlerDeps are the route groups and services used by auth handlers.
type AuthHandlerDeps struct {
	Public      huma.API
	Session     huma.API
	Protected   huma.API
	Router      chi.Router
	AuthService *auth.Service
	Users       *directory.UserService
	Logger      *slog.Logger
}

// RegisterAuth mounts session, account, and OIDC endpoints.
func RegisterAuth(deps AuthHandlerDeps) {
	registerGetSession(deps.Session, deps.AuthService)
	registerCreateSession(deps.Public, deps.AuthService, deps.Logger)
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
		Summary:     "Get the current signed-in principal, if any",
	}, func(ctx context.Context, _ *struct{}) (*sessionOutput, error) {
		out := &sessionOutput{Body: sessionBody{
			SSOEnabled: authService.SSOEnabled(),
		}}
		if principal, ok := ctxkeys.Principal(ctx); ok {
			out.Body.User = principal
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
		Summary:     "Create a local admin session",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusTooManyRequests},
	}, func(ctx context.Context, input *sessionCreateInput) (*principalOutput, error) {
		principal, err := authService.Login(ctx, auth.LoginParams{
			ClientIP: chimiddleware.GetClientIP(ctx),
			Email:    input.Body.Email,
			Password: input.Body.Password,
		})
		if err != nil {
			return nil, handlerError(ctx, logger, "create-session", authError(err))
		}
		return &principalOutput{Body: *principal}, nil
	})
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
	case errors.Is(err, auth.ErrTooManyAttempts):
		return huma.Error429TooManyRequests("too many login attempts; try again shortly")
	case errors.Is(err, directory.ErrWeakPassword):
		return huma.Error400BadRequest(directory.ErrWeakPassword.Error())
	default:
		return err
	}
}
