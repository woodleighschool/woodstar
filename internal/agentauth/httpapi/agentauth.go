package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/woodleighschool/goodies/auth/authz"
	authhuma "github.com/woodleighschool/goodies/auth/huma"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/rbac"
)

const (
	agentSecretPath   = "/api/agent-secrets" //nolint:gosec // API path, not a credential.
	agentSecretIDPath = agentSecretPath + "/{id}"
)

type agentSecretListOutput struct {
	Body []agentauth.AgentSecret
}

type agentSecretCreateInput struct {
	Body agentauth.AgentSecretCreate
}

type agentSecretCreateOutput struct {
	Body agentauth.AgentSecret
}

type agentSecretUpdateInput struct {
	ID   int64 `path:"id"`
	Body agentauth.AgentSecretMutation
}

type agentSecretDeleteInput struct {
	ID int64 `path:"id"`
}

// RegisterAPI mounts shared agent-secret management endpoints.
func RegisterAPI(
	routes api.AppRoutes,
	store *agentauth.Store,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	registerAPI(routes, store, authorizer, logger)
}

// RegisterOpenAPI documents agent-secret endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerAPI(routes, nil, nil, nil)
}

func registerAPI(
	routes api.AppRoutes,
	store *agentauth.Store,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	humaAPI := authhuma.RequireAPI(routes.Protected, authorizer, logger, authz.Requirement{
		Resource: rbac.ResourceAgentSecrets,
		Access:   authz.Edit,
	})
	registerListAgentSecrets(humaAPI, store, logger)
	registerCreateAgentSecret(humaAPI, store, logger)
	registerUpdateAgentSecret(humaAPI, store, logger)
	registerDeleteAgentSecret(humaAPI, store, logger)
}

func registerListAgentSecrets(humaAPI huma.API, store *agentauth.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-agent-secrets",
		Method:      http.MethodGet,
		Path:        agentSecretPath,
		Tags:        []string{api.TagAgentSecrets},
		Summary:     "List agent secrets",
	}, func(ctx context.Context, _ *struct{}) (*agentSecretListOutput, error) {
		secrets, err := store.List(ctx)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "list-agent-secrets", err)
		}
		return &agentSecretListOutput{Body: secrets}, nil
	})
}

func registerCreateAgentSecret(humaAPI huma.API, store *agentauth.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID:   "create-agent-secret",
		Method:        http.MethodPost,
		Path:          agentSecretPath,
		Tags:          []string{api.TagAgentSecrets},
		Summary:       "Create an agent secret",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *agentSecretCreateInput) (*agentSecretCreateOutput, error) {
		secret, err := store.Create(ctx, input.Body)
		if errors.Is(err, agentauth.ErrInvalidAgent) {
			return nil, huma.Error400BadRequest("invalid agent")
		}
		if errors.Is(err, agentauth.ErrInvalidSecret) {
			return nil, huma.Error400BadRequest("invalid agent secret")
		}
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "create-agent-secret", err, "agent", input.Body.Agent)
		}
		return &agentSecretCreateOutput{Body: *secret}, nil
	})
}

func registerUpdateAgentSecret(humaAPI huma.API, store *agentauth.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "update-agent-secret",
		Method:      http.MethodPut,
		Path:        agentSecretIDPath,
		Tags:        []string{api.TagAgentSecrets},
		Summary:     "Update an agent secret",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *agentSecretUpdateInput) (*agentSecretCreateOutput, error) {
		secret, err := store.Update(ctx, input.ID, input.Body)
		if errors.Is(err, agentauth.ErrInvalidSecret) {
			return nil, huma.Error400BadRequest("invalid agent secret")
		}
		if errors.Is(err, fault.ErrNotFound) {
			return nil, huma.Error404NotFound("agent secret not found")
		}
		if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"update-agent-secret",
				err,
				"id", input.ID,
			)
		}
		return &agentSecretCreateOutput{Body: *secret}, nil
	})
}

func registerDeleteAgentSecret(humaAPI huma.API, store *agentauth.Store, logger *slog.Logger) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "delete-agent-secret",
		Method:      http.MethodDelete,
		Path:        agentSecretIDPath,
		Tags:        []string{api.TagAgentSecrets},
		Summary:     "Delete an agent secret",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, func(ctx context.Context, input *agentSecretDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.ID); errors.Is(err, fault.ErrNotFound) {
			return nil, huma.Error404NotFound("agent secret not found")
		} else if err != nil {
			return nil, api.HandlerError(
				ctx,
				logger,
				"delete-agent-secret",
				err,
				"id", input.ID,
			)
		}
		return &struct{}{}, nil
	})
}
