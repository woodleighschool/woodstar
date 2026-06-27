package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/directory"
)

const (
	authTag    = "Auth"
	setupTag   = "Setup"
	accountTag = "Account"
)

type sessionOutput struct {
	Body sessionBody
}

type sessionBody struct {
	SetupComplete bool            `json:"setup_complete"`
	SSOEnabled    bool            `json:"sso_enabled"`
	User          *directory.User `json:"user,omitempty"`
}

type authUserOutput struct {
	Body directory.User
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

// RegisterAuth mounts setup, session, login, logout, and account endpoints.
func RegisterAuth(g Groups, authService *auth.Service, userService *directory.UserService, logger *slog.Logger) {
	registerSetup(g.Public, authService, logger)
	registerSession(g.Session, authService, logger)
	registerLogin(g.Public, authService, logger)
	registerLogout(g.Protected, authService, logger)
	auth.RegisterSSO(g.Router, authService)

	registerGetAccount(g.Protected, authService, logger)
	registerPutAccount(g.Protected, userService, logger)
	registerRotateAPIKey(g.Protected, authService, logger)
	registerRevokeAPIKey(g.Protected, authService, logger)
}

func registerSetup(api huma.API, authService *auth.Service, logger *slog.Logger) {
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
			return nil, handlerError(ctx, logger, "complete-setup", authError(err))
		}
		return &authUserOutput{Body: *user}, nil
	})
}

func registerSession(api huma.API, authService *auth.Service, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-session",
		Method:      http.MethodGet,
		Path:        "/api/auth/session",
		Tags:        []string{authTag},
		Summary:     "Get setup state and the current signed-in user, if any",
	}, func(ctx context.Context, _ *struct{}) (*sessionOutput, error) {
		complete, err := authService.SetupComplete(ctx)
		if err != nil {
			return nil, handlerError(ctx, logger, "get-session", err)
		}
		out := &sessionOutput{Body: sessionBody{
			SetupComplete: complete,
			SSOEnabled:    authService.SSOEnabled(),
		}}
		if user, ok := ctxkeys.User(ctx); ok {
			out.Body.User = user
		}
		return out, nil
	})
}

func registerLogin(api huma.API, authService *auth.Service, logger *slog.Logger) {
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
			return nil, handlerError(ctx, logger, "create-session", authError(err))
		}
		return &authUserOutput{Body: *user}, nil
	})
}

func registerLogout(api huma.API, authService *auth.Service, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-session",
		Method:      http.MethodPost,
		Path:        "/api/auth/logout",
		Tags:        []string{authTag},
		Summary:     "Revoke the current session",
		Errors:      []int{http.StatusUnauthorized},
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
	case errors.Is(err, auth.ErrNotSetup):
		return huma.Error409Conflict("setup required")
	case errors.Is(err, auth.ErrAlreadySetup):
		return huma.Error409Conflict("woodstar is already set up")
	case errors.Is(err, directory.ErrWeakPassword):
		return huma.Error400BadRequest(directory.ErrWeakPassword.Error())
	default:
		return err
	}
}
