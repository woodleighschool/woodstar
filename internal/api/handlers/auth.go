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
)

type userBody struct {
	ID    string          `json:"id"`
	Email string          `json:"email"`
	Name  string          `json:"name"`
	Role  models.UserRole `json:"role"`
}

type setupStatusOutput struct {
	Body struct {
		Complete bool `json:"complete"`
	}
}

type authUserOutput struct {
	Body userBody
}

type authSessionOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      userBody
}

type setupInput struct {
	Body struct {
		Email    string `json:"email" format:"email"`
		Name     string `json:"name,omitempty"`
		Password string `json:"password" minLength:"12"`
	}
}

type loginInput struct {
	Body struct {
		Email    string `json:"email" format:"email"`
		Password string `json:"password" minLength:"1"`
	}
}

type sessionInput struct {
	Session string `cookie:"woodstar_session"`
}

const (
	authTag  = "Auth"
	setupTag = "Setup"
)

// CookieSettings control how browser sessions are scoped.
type CookieSettings struct {
	CookiePath   string
	SecureCookie bool
}

// RegisterAuth registers setup and browser session endpoints.
func RegisterAuth(api huma.API, authService *auth.Service, cookies CookieSettings) {
	registerSetup(api, authService, cookies)
	registerSessions(api, authService, cookies)
}

func registerSetup(api huma.API, authService *auth.Service, cookies CookieSettings) {
	huma.Register(api, huma.Operation{
		OperationID: "get-setup-status",
		Method:      http.MethodGet,
		Path:        "/api/setup/status",
		Tags:        []string{setupTag},
		Summary:     "Check whether initial setup is complete",
	}, func(ctx context.Context, _ *struct{}) (*setupStatusOutput, error) {
		complete, err := authService.SetupComplete(ctx)
		if err != nil {
			return nil, err
		}
		out := &setupStatusOutput{}
		out.Body.Complete = complete
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "complete-setup",
		Method:        http.MethodPost,
		Path:          "/api/setup",
		Tags:          []string{setupTag},
		Summary:       "Create the first administrator account",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusConflict},
	}, func(ctx context.Context, input *setupInput) (*authSessionOutput, error) {
		result, err := authService.Setup(ctx, auth.SetupParams{
			Email:    input.Body.Email,
			Name:     input.Body.Name,
			Password: input.Body.Password,
		})
		if err != nil {
			return nil, authError(err)
		}
		return sessionOutput(result, cookies), nil
	})
}

func registerSessions(api huma.API, authService *auth.Service, cookies CookieSettings) {
	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/api/auth/login",
		Tags:        []string{authTag},
		Summary:     "Create a local admin session",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusConflict},
	}, func(ctx context.Context, input *loginInput) (*authSessionOutput, error) {
		result, err := authService.Login(ctx, input.Body.Email, input.Body.Password)
		if err != nil {
			return nil, authError(err)
		}
		return sessionOutput(result, cookies), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      http.MethodPost,
		Path:        "/api/auth/logout",
		Tags:        []string{authTag},
		Summary:     "Revoke the current session",
	}, func(ctx context.Context, input *sessionInput) (*struct {
		SetCookie http.Cookie `header:"Set-Cookie"`
	}, error,
	) {
		if err := authService.Logout(ctx, input.Session); err != nil {
			return nil, err
		}
		return &struct {
			SetCookie http.Cookie `header:"Set-Cookie"`
		}{SetCookie: clearSessionCookie(cookies)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-current-user",
		Method:      http.MethodGet,
		Path:        "/api/auth/me",
		Tags:        []string{authTag},
		Summary:     "Get the current signed-in user",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *sessionInput) (*authUserOutput, error) {
		user, err := authService.CurrentUser(ctx, input.Session)
		if err != nil {
			return nil, authError(err)
		}
		return &authUserOutput{Body: userResponse(user)}, nil
	})
}

func sessionOutput(result *auth.LoginResult, cookies CookieSettings) *authSessionOutput {
	return &authSessionOutput{
		SetCookie: sessionCookie(result.Token, result.ExpiresAt, cookies),
		Body:      userResponse(result.User),
	}
}

func userResponse(user *models.User) userBody {
	return userBody{
		ID:    models.UserIDString(user.ID),
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}
}

func sessionCookie(token string, expiresAt time.Time, cookies CookieSettings) http.Cookie {
	//nolint:gosec // Secure is derived from WOODSTAR_BASE_URL so local HTTP and HTTPS reverse-proxy deployments both work.
	return http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    token,
		Path:     cookiePath(cookies.CookiePath),
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   cookies.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	}
}

func clearSessionCookie(cookies CookieSettings) http.Cookie {
	//nolint:gosec // Secure is derived from WOODSTAR_BASE_URL so local HTTP and HTTPS reverse-proxy deployments both work.
	return http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     cookiePath(cookies.CookiePath),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cookies.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	}
}

func cookiePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return "/"
	}
	return path
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
