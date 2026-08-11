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
	"github.com/woodleighschool/woodstar/internal/fault"
)

type accountOutput struct {
	Body directory.Account
}

type accountPutInput struct {
	Body directory.AccountMutation
}

func registerAccountAction(
	humaAPI huma.API,
	op huma.Operation,
	action func(context.Context, int64) (*directory.Account, error),
	logger *slog.Logger,
) {
	op.Errors = append(op.Errors, http.StatusNotFound)
	huma.Register(humaAPI, op, func(ctx context.Context, _ *struct{}) (*accountOutput, error) {
		user, err := ctxkeys.RequireUser(ctx)
		if err != nil {
			return nil, err
		}
		account, err := action(ctx, user.ID)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, op.OperationID, err, "user_id", user.ID)
		}
		return &accountOutput{Body: *account}, nil
	})
}

func registerGetAccount(humaAPI huma.API, authService *auth.Service, logger *slog.Logger) {
	registerAccountAction(humaAPI, huma.Operation{
		OperationID: "get-account",
		Method:      http.MethodGet,
		Path:        "/api/account",
		Tags:        []string{api.TagAccount},
		Summary:     "Get account",
	}, authService.Account, logger)
}

func registerPutAccount(humaAPI huma.API, userService *directory.UserService, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-account",
		Method:      http.MethodPut,
		Path:        "/api/account",
		Tags:        []string{api.TagAccount},
		Summary:     "Update account",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusConflict,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *accountPutInput) (*accountOutput, error) {
		user, err := ctxkeys.RequireUser(ctx)
		if err != nil {
			return nil, err
		}
		account, err := userService.UpdateAccount(ctx, user.ID, input.Body)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "update-account", accountMutationError(err), "user_id", user.ID)
		}
		return &accountOutput{Body: *account}, nil
	})
}

func registerRotateAPIKey(humaAPI huma.API, authService *auth.Service, logger *slog.Logger) {
	registerAccountAction(humaAPI, huma.Operation{
		OperationID:   "rotate-account-api-key",
		Method:        http.MethodPost,
		Path:          "/api/account/api-key",
		Tags:          []string{api.TagAccount},
		Summary:       "Rotate API key",
		DefaultStatus: http.StatusCreated,
	}, authService.RotateAPIKey, logger)
}

func registerRevokeAPIKey(humaAPI huma.API, authService *auth.Service, logger *slog.Logger) {
	registerAccountAction(humaAPI, huma.Operation{
		OperationID: "revoke-account-api-key",
		Method:      http.MethodDelete,
		Path:        "/api/account/api-key",
		Tags:        []string{api.TagAccount},
		Summary:     "Revoke API key",
	}, authService.RevokeAPIKey, logger)
}

func accountMutationError(err error) error {
	switch {
	case errors.Is(err, fault.ErrAlreadyExists):
		return huma.Error409Conflict("email already in use")
	case errors.Is(err, directory.ErrWeakPassword):
		return huma.Error400BadRequest(directory.ErrWeakPassword.Error())
	default:
		return api.ResourceMutationError("user", err)
	}
}
