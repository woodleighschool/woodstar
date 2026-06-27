package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/ctxkeys"
	"github.com/woodleighschool/woodstar/internal/apitypes"
	"github.com/woodleighschool/woodstar/internal/auth"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/directory"
)

type accountOutput struct {
	Body directory.Account
}

type accountPutInput struct {
	Body directory.AccountMutation
}

func registerAccountAction(
	api huma.API,
	op huma.Operation,
	action func(context.Context, int64) (*directory.Account, error),
) {
	huma.Register(api, op, func(ctx context.Context, _ *struct{}) (*accountOutput, error) {
		user, err := ctxkeys.RequireUser(ctx)
		if err != nil {
			return nil, err
		}
		account, err := action(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		return &accountOutput{Body: *account}, nil
	})
}

func registerGetAccount(api huma.API, authService *auth.Service) {
	registerAccountAction(api, huma.Operation{
		OperationID: "get-account",
		Method:      http.MethodGet,
		Path:        "/api/account",
		Tags:        []string{accountTag},
		Summary:     "Get the signed-in user's account, including any API key",
		Errors:      []int{http.StatusUnauthorized},
	}, authService.Account)
}

func registerPutAccount(api huma.API, userService *directory.UserService) {
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
		user, err := ctxkeys.RequireUser(ctx)
		if err != nil {
			return nil, err
		}
		account, err := userService.UpdateAccount(ctx, user.ID, input.Body)
		if err != nil {
			return nil, accountMutationError(err)
		}
		return &accountOutput{Body: *account}, nil
	})
}

func registerRotateAPIKey(api huma.API, authService *auth.Service) {
	registerAccountAction(api, huma.Operation{
		OperationID:   "rotate-account-api-key",
		Method:        http.MethodPost,
		Path:          "/api/account/api-key",
		Tags:          []string{accountTag},
		Summary:       "Generate a new API key for the signed-in user, replacing any prior key",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnauthorized},
	}, authService.RotateAPIKey)
}

func registerRevokeAPIKey(api huma.API, authService *auth.Service) {
	registerAccountAction(api, huma.Operation{
		OperationID: "revoke-account-api-key",
		Method:      http.MethodDelete,
		Path:        "/api/account/api-key",
		Tags:        []string{accountTag},
		Summary:     "Clear the API key on the signed-in user's account",
		Errors:      []int{http.StatusUnauthorized},
	}, authService.RevokeAPIKey)
}

func accountMutationError(err error) error {
	switch {
	case errors.Is(err, dbutil.ErrAlreadyExists):
		return huma.Error409Conflict("email already in use")
	case errors.Is(err, directory.ErrWeakPassword):
		return huma.Error400BadRequest(directory.ErrWeakPassword.Error())
	default:
		return apitypes.ResourceMutationError("user", err)
	}
}
