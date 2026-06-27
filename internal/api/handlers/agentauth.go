package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const (
	agentSecretsTag   = "Agent secrets"
	agentSecretPath   = "/api/agent-secrets" //nolint:gosec // G101: URL path, not a credential
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
	AgentSecretID int64 `path:"id"`
	Body          agentauth.AgentSecretMutation
}

type agentSecretDeleteInput struct {
	AgentSecretID int64 `path:"id"`
}

func registerAgentAuth(g Groups, deps Dependencies) {
	registerListAgentSecrets(g.Sensitive, deps.Secrets, deps.Logger)
	registerCreateAgentSecret(g.Sensitive, deps.Secrets, deps.Logger)
	registerUpdateAgentSecret(g.Sensitive, deps.Secrets, deps.Logger)
	registerDeleteAgentSecret(g.Sensitive, deps.Secrets, deps.Logger)
}

func registerListAgentSecrets(api huma.API, store *agentauth.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "list-agent-secrets",
		Method:      http.MethodGet,
		Path:        agentSecretPath,
		Tags:        []string{agentSecretsTag},
		Summary:     "List agent secrets",
		Errors:      []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, _ *struct{}) (*agentSecretListOutput, error) {
		secrets, err := store.List(ctx)
		if err != nil {
			return nil, handlerError(ctx, logger, "list-agent-secrets", err)
		}
		return &agentSecretListOutput{Body: secrets}, nil
	})
}

func registerCreateAgentSecret(api huma.API, store *agentauth.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-agent-secret",
		Method:        http.MethodPost,
		Path:          agentSecretPath,
		Tags:          []string{agentSecretsTag},
		Summary:       "Create agent secret",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *agentSecretCreateInput) (*agentSecretCreateOutput, error) {
		secret, err := store.Create(ctx, input.Body)
		if errors.Is(err, agentauth.ErrInvalidAgent) {
			return nil, huma.Error400BadRequest("invalid agent")
		}
		if errors.Is(err, agentauth.ErrInvalidSecret) {
			return nil, huma.Error400BadRequest("invalid agent secret")
		}
		if err != nil {
			return nil, handlerError(ctx, logger, "create-agent-secret", err, "agent", input.Body.Agent)
		}
		return &agentSecretCreateOutput{Body: *secret}, nil
	})
}

func registerUpdateAgentSecret(api huma.API, store *agentauth.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "update-agent-secret",
		Method:      http.MethodPut,
		Path:        agentSecretIDPath,
		Tags:        []string{agentSecretsTag},
		Summary:     "Update agent secret",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *agentSecretUpdateInput) (*agentSecretCreateOutput, error) {
		secret, err := store.Update(ctx, input.AgentSecretID, input.Body)
		if errors.Is(err, agentauth.ErrInvalidSecret) {
			return nil, huma.Error400BadRequest("invalid agent secret")
		}
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("agent secret not found")
		}
		if err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"update-agent-secret",
				err,
				"agent_secret_id", input.AgentSecretID,
			)
		}
		return &agentSecretCreateOutput{Body: *secret}, nil
	})
}

func registerDeleteAgentSecret(api huma.API, store *agentauth.Store, logger *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-agent-secret",
		Method:      http.MethodDelete,
		Path:        agentSecretIDPath,
		Tags:        []string{agentSecretsTag},
		Summary:     "Delete agent secret",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *agentSecretDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.AgentSecretID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("agent secret not found")
		} else if err != nil {
			return nil, handlerError(
				ctx,
				logger,
				"delete-agent-secret",
				err,
				"agent_secret_id", input.AgentSecretID,
			)
		}
		return &struct{}{}, nil
	})
}
