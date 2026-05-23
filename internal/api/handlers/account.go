package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/users"
)

const accountTag = "Account"

// accountBody is the self-view returned to the signed-in user. APIKey is
// the plaintext token; it is only present in /api/account responses, never
// in admin endpoints like ListUsers where one admin can see others.
type accountBody struct {
	User            users.User `json:"user"`
	APIKey          string     `json:"api_key,omitempty"`
	APIKeyCreatedAt *time.Time `json:"api_key_created_at,omitempty"`
}

type accountOutput struct {
	Body accountBody
}

type accountPutBody struct {
	Name     string  `json:"name"`
	Password *string `json:"password,omitempty"`
}

type accountPutInput struct {
	Body accountPutBody
}

// RegisterAccount registers self-service endpoints scoped to the signed-in
// user. The API key is intended for non-browser callers; the SPA continues to
// authenticate via the scs session cookie.
func RegisterAccount(api huma.API, authService *auth.Service, userService *users.Service) {
	registerGetAccount(api, authService)
	registerPutAccount(api, userService)
	registerRotateAPIKey(api, authService)
	registerRevokeAPIKey(api, authService)
}

func registerGetAccount(api huma.API, authService *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "get-account",
		Method:      http.MethodGet,
		Path:        "/api/account",
		Tags:        []string{accountTag},
		Summary:     "Get the signed-in user's account, including any API key",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*accountOutput, error) {
		user, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		account, err := authService.Account(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		return &accountOutput{Body: accountBodyFor(*account)}, nil
	})
}

func registerPutAccount(api huma.API, userService *users.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "update-account",
		Method:      http.MethodPut,
		Path:        "/api/account",
		Tags:        []string{accountTag},
		Summary:     "Update the signed-in user's account",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusConflict,
		},
	}, func(ctx context.Context, input *accountPutInput) (*accountOutput, error) {
		user, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		account, err := userService.UpdateAccount(ctx, user.ID, users.AccountUpdateParams{
			Name:     input.Body.Name,
			Password: input.Body.Password,
		})
		if err != nil {
			return nil, userMutationError(err)
		}
		return &accountOutput{Body: accountBodyFor(*account)}, nil
	})
}

func registerRotateAPIKey(api huma.API, authService *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID:   "rotate-account-api-key",
		Method:        http.MethodPost,
		Path:          "/api/account/api-key",
		Tags:          []string{accountTag},
		Summary:       "Generate a new API key for the signed-in user, replacing any prior key",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*accountOutput, error) {
		user, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		rotated, err := authService.RotateAPIKey(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		return &accountOutput{Body: accountBodyFor(*rotated)}, nil
	})
}

func registerRevokeAPIKey(api huma.API, authService *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "revoke-account-api-key",
		Method:      http.MethodDelete,
		Path:        "/api/account/api-key",
		Tags:        []string{accountTag},
		Summary:     "Clear the API key on the signed-in user's account",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*accountOutput, error) {
		user, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		cleared, err := authService.RevokeAPIKey(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		return &accountOutput{Body: accountBodyFor(*cleared)}, nil
	})
}

// requireUser returns the authenticated user from ctx regardless of role.
// Unlike requireAdmin this accepts both admin and viewer; the operations
// it gates (rotating one's own key, viewing one's own account) are open to
// every signed-in user.
func requireUser(ctx context.Context) (*users.User, error) {
	user, ok := userFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("not authenticated")
	}
	return user, nil
}

func accountBodyFor(account users.Account) accountBody {
	return accountBody{
		User:            account.User,
		APIKey:          account.APIKey,
		APIKeyCreatedAt: account.APIKeyCreatedAt,
	}
}
