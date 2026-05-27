package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const (
	agentSecretsTag = "Agent secrets"
	agentSecretPath = "/api/agent-secrets"
)

type agentSecretListOutput struct {
	Body []agentauth.AgentSecret
}

type agentSecretCreateInput struct {
	Body struct {
		Agent agentauth.Agent `json:"agent" enum:"orbit,santa"`
		Value string          `json:"value" minLength:"32"`
	}
}

type agentSecretCreateOutput struct {
	Body agentauth.AgentSecret
}

type agentSecretUpdateInput struct {
	ID   int64 `path:"id"`
	Body struct {
		Value string `json:"value" minLength:"32"`
	}
}

type agentSecretDeleteInput struct {
	ID int64 `path:"id"`
}

func RegisterAgentSecrets(api huma.API, store *agentauth.Store) {
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

	huma.Register(api, huma.Operation{
		OperationID:   "create-agent-secret",
		Method:        http.MethodPost,
		Path:          agentSecretPath,
		Tags:          []string{agentSecretsTag},
		Summary:       "Create agent secret",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, input *agentSecretCreateInput) (*agentSecretCreateOutput, error) {
		agent, err := agentauth.ParseAgent(string(input.Body.Agent))
		if err != nil {
			return nil, huma.Error400BadRequest("invalid agent")
		}
		secret, err := store.Create(ctx, agent, input.Body.Value)
		if errors.Is(err, agentauth.ErrInvalidSecret) {
			return nil, huma.Error400BadRequest("invalid agent secret")
		}
		if err != nil {
			return nil, err
		}
		return &agentSecretCreateOutput{Body: secret}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-agent-secret",
		Method:      http.MethodPatch,
		Path:        agentSecretPath + "/{id}",
		Tags:        []string{agentSecretsTag},
		Summary:     "Update agent secret",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
		},
	}, func(ctx context.Context, input *agentSecretUpdateInput) (*agentSecretCreateOutput, error) {
		secret, err := store.Update(ctx, input.ID, input.Body.Value)
		if errors.Is(err, agentauth.ErrInvalidSecret) {
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

	huma.Register(api, huma.Operation{
		OperationID: "delete-agent-secret",
		Method:      http.MethodDelete,
		Path:        agentSecretPath + "/{id}",
		Tags:        []string{agentSecretsTag},
		Summary:     "Delete agent secret",
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, input *agentSecretDeleteInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.ID); errors.Is(err, dbutil.ErrNotFound) {
			return nil, huma.Error404NotFound("agent secret not found")
		} else if err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
}
