// Package account owns the application's account HTTP endpoints.
package account

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/woodleighschool/goodies/auth/authn"
	"github.com/woodleighschool/goodies/auth/authz"
	authhuma "github.com/woodleighschool/goodies/auth/huma"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/directory"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/rbac"
)

// Dependencies supplies the account and session handlers.
type Dependencies struct {
	Users  *directory.UserService
	Authn  *authn.Service
	Authz  *authz.Service
	Logger *slog.Logger
}

// Account is the signed-in user's self-view.
type Account struct {
	User                 directory.User                  `json:"user"`
	APIKey               string                          `json:"api_key,omitempty"`
	APIKeyCreatedAt      *time.Time                      `json:"api_key_created_at,omitempty"`
	EffectivePermissions map[authz.Resource]authz.Access `json:"effective_permissions"`
}

type output struct {
	Body Account
}

// TransformSchema constrains effective-permission keys to the application catalogue.
func (Account) TransformSchema(registry huma.Registry, schema *huma.Schema) *huma.Schema {
	registry.Map()["AuthzResource"] = rbac.ResourceSchema()
	schema.Properties["effective_permissions"].Extensions = map[string]any{
		"propertyNames": &huma.Schema{Ref: "#/components/schemas/AuthzResource"},
	}
	return schema
}

type updateInput struct {
	Body directory.AccountMutation
}

// RegisterAPI mounts account, session and OIDC endpoints.
func RegisterAPI(routes api.AppRoutes, deps Dependencies) {
	authhuma.RegisterSessions(routes.Session, routes.PasswordLogin, routes.Logout, deps.Authn, deps.Logger, api.TagSession)
	registerAccount(routes.Protected, deps)
	routes.Router.Get("/api/auth/sso/start", deps.Authn.SSOStart)
	routes.Router.Get("/api/auth/sso/callback", deps.Authn.SSOCallback)
}

// RegisterOpenAPI documents account and session endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	authhuma.RegisterSessions(routes.Session, routes.PasswordLogin, routes.Logout, nil, nil, api.TagSession)
	registerAccount(routes.Protected, Dependencies{})
}

func registerAccount(humaAPI huma.API, deps Dependencies) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-account",
		Method:      http.MethodGet,
		Path:        "/api/account",
		Tags:        []string{api.TagAccount},
		Summary:     "Get account",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, _ *struct{}) (*output, error) {
		principal, err := authn.RequirePrincipal(ctx)
		if err != nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		account, err := loadAccount(ctx, deps, principal.ID)
		if err != nil {
			return nil, api.HandlerError(ctx, deps.Logger, "get-account", err, "user_id", principal.ID)
		}
		return &output{Body: *account}, nil
	})

	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-account",
		Method:      http.MethodPut,
		Path:        "/api/account",
		Tags:        []string{api.TagAccount},
		Summary:     "Update account",
		Errors:      []int{http.StatusBadRequest, http.StatusConflict, http.StatusNotFound},
	}, func(ctx context.Context, input *updateInput) (*output, error) {
		principal, err := authn.RequirePrincipal(ctx)
		if err != nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		stored, err := deps.Users.UpdateAccount(ctx, principal.ID, input.Body)
		if err != nil {
			return nil, api.HandlerError(ctx, deps.Logger, "update-account", mutationError(err), "user_id", principal.ID)
		}
		account, err := accountView(ctx, deps.Authz, stored)
		if err != nil {
			return nil, api.HandlerError(ctx, deps.Logger, "update-account", err, "user_id", principal.ID)
		}
		return &output{Body: *account}, nil
	})

	huma.Register(humaAPI, huma.Operation{
		OperationID:   "rotate-account-api-key",
		Method:        http.MethodPost,
		Path:          "/api/account/api-key",
		Tags:          []string{api.TagAccount},
		Summary:       "Rotate API key",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusNotFound},
	}, func(ctx context.Context, _ *struct{}) (*output, error) {
		principal, err := authn.RequirePrincipal(ctx)
		if err != nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if err := deps.Authn.RotateAPIKey(ctx, principal.ID); err != nil {
			return nil, api.HandlerError(ctx, deps.Logger, "rotate-account-api-key", err, "user_id", principal.ID)
		}
		account, err := loadAccount(ctx, deps, principal.ID)
		if err != nil {
			return nil, api.HandlerError(ctx, deps.Logger, "rotate-account-api-key", err, "user_id", principal.ID)
		}
		return &output{Body: *account}, nil
	})

	huma.Register(humaAPI, huma.Operation{
		OperationID: "revoke-account-api-key",
		Method:      http.MethodDelete,
		Path:        "/api/account/api-key",
		Tags:        []string{api.TagAccount},
		Summary:     "Revoke API key",
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, _ *struct{}) (*output, error) {
		principal, err := authn.RequirePrincipal(ctx)
		if err != nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if err := deps.Authn.RevokeAPIKey(ctx, principal.ID); err != nil {
			return nil, api.HandlerError(ctx, deps.Logger, "revoke-account-api-key", err, "user_id", principal.ID)
		}
		account, err := loadAccount(ctx, deps, principal.ID)
		if err != nil {
			return nil, api.HandlerError(ctx, deps.Logger, "revoke-account-api-key", err, "user_id", principal.ID)
		}
		return &output{Body: *account}, nil
	})
}

func loadAccount(ctx context.Context, deps Dependencies, principalID int64) (*Account, error) {
	stored, err := deps.Users.GetAccount(ctx, principalID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	return accountView(ctx, deps.Authz, stored)
}

func accountView(ctx context.Context, permissions *authz.Service, stored *directory.Account) (*Account, error) {
	effectivePermissions, err := permissions.EffectivePermissions(ctx, stored.User.ID)
	if err != nil {
		return nil, fmt.Errorf("get effective permissions: %w", err)
	}
	return &Account{
		User:                 stored.User,
		APIKey:               stored.APIKey,
		APIKeyCreatedAt:      stored.APIKeyCreatedAt,
		EffectivePermissions: effectivePermissions,
	}, nil
}

func mutationError(err error) error {
	switch {
	case errors.Is(err, fault.ErrAlreadyExists):
		return huma.Error409Conflict("email already in use")
	case errors.Is(err, directory.ErrWeakPassword):
		return huma.Error400BadRequest(directory.ErrWeakPassword.Error())
	default:
		return api.ResourceMutationError("user", err)
	}
}
