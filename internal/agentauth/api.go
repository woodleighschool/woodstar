package agentauth

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const (
	agentSecretsTag   = "Agent secrets"
	agentSecretPath   = "/api/agent-secrets"
	agentSecretIDPath = agentSecretPath + "/{id}"
)

type agentSecretListOutput struct {
	Body []AgentSecret
}

type agentSecretCreateInput struct {
	Body AgentSecretCreate
}

type agentSecretCreateOutput struct {
	Body AgentSecret
}

type agentSecretUpdateInput struct {
	AgentSecretID int64 `path:"id"`
	Body          AgentSecretMutation
}

type agentSecretDeleteInput struct {
	AgentSecretID int64 `path:"id"`
}

func RegisterAdminRoutes(api huma.API, store *Store) {
	registerListAgentSecrets(api, store)
	registerCreateAgentSecret(api, store)
	registerUpdateAgentSecret(api, store)
	registerDeleteAgentSecret(api, store)
}

func registerListAgentSecrets(api huma.API, store *Store) {
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
			return nil, err
		}
		return &agentSecretListOutput{Body: secrets}, nil
	})
}

func registerCreateAgentSecret(api huma.API, store *Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-agent-secret",
		Method:        http.MethodPost,
		Path:          agentSecretPath,
		Tags:          []string{agentSecretsTag},
		Summary:       "Create agent secret",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *agentSecretCreateInput) (*agentSecretCreateOutput, error) {
		if _, err := ParseAgent(string(input.Body.Agent)); err != nil {
			return nil, huma.Error400BadRequest("invalid agent")
		}
		secret, err := store.Create(ctx, input.Body)
		if errors.Is(err, ErrInvalidSecret) {
			return nil, huma.Error400BadRequest("invalid agent secret")
		}
		if err != nil {
			return nil, err
		}
		return &agentSecretCreateOutput{Body: secret}, nil
	})
}

func registerUpdateAgentSecret(api huma.API, store *Store) {
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
		if errors.Is(err, ErrInvalidSecret) {
			return nil, huma.Error400BadRequest("invalid agent secret")
		}
		if errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("agent secret not found")
		}
		if err != nil {
			return nil, err
		}
		return &agentSecretCreateOutput{Body: secret}, nil
	})
}

func registerDeleteAgentSecret(api huma.API, store *Store) {
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
			return nil, err
		}
		return &struct{}{}, nil
	})
}
